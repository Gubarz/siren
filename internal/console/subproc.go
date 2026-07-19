//go:build unix

package console

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"
	stdlog "log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/bishopfox/sliver/client/assets"
	"github.com/bishopfox/sliver/client/command"
	sliverconsole "github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/client/transport"
	"github.com/creack/pty"
	reefconsole "github.com/reeflective/console"
	"github.com/spf13/cobra"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// ConsoleModeFlag names the CLI flag that puts a re-exec of this binary
// into "run a sliver client console" mode. main.go checks for it before
// spawning the wails app.
const ConsoleModeFlag = "--sliver-console"

const (
	subprocOutputInterval = 16 * time.Millisecond
	subprocOutputBatch    = 16 * 1024
)

const (
	shellOpenFramePrefix = "\x1b]777;sliver-gui-open-shell="
	shellOpenFrameSuffix = "\x07"
)

// subprocMgr owns the per-session console subprocesses. Zero value ready
// to use. Access under mu; each running job is independent otherwise.
//
// `bySession` and `pending` cooperate to route GUI-triggered commands
// into the session's live console (subprocess). If the console exists,
// SendToSessionConsole writes straight into its pty master. If not, the
// line queues in `pending[sessionID]` and gets picked up by the next
// StartConsole for that session as extra rcScript lines — so opening a
// console and issuing a command are naturally atomic from the caller's
// perspective (the subprocess sees `use <id>\n<queued...>\n`).
type subprocMgr struct {
	mu        sync.Mutex
	jobs      map[string]*subprocJob
	bySession map[string]string // sessionID -> jobID
	pending   map[string][]string
	next      atomic.Uint64
}

type subprocJob struct {
	id        string
	sessionID string
	cmd       *exec.Cmd
	pty       *os.File // master
}

// StartConsole spawns a subprocess that runs a real sliver client
// console (readline + all commands) with its stdio wired to a fresh PTY.
// Interactive prompts work because the subprocess owns fd 0/1/2 outright
// — no in-process hijack. Returns a jobID the frontend uses for I/O.
func (s *Service) StartConsole(sessionID string) (string, error) {
	self, err := os.Executable()
	if err != nil {
		return "", err
	}

	// The subprocess needs the operator config to reconnect. We pass the
	// serialized ClientConfig via a $TMPDIR file, deleted after read.
	cfgPath, err := writeConfigForSubproc(s.rpc.Config)
	if err != nil {
		return "", err
	}

	cmd := exec.Command(self, ConsoleModeFlag, cfgPath, sessionID)
	// Pre-populate terminal capability env vars so termenv/lipgloss don't
	// probe with OSC queries (\x1b]11;?\x07 etc). xterm.js answers those
	// truthfully but sliver reads the reply back as user input, which
	// then shows up on the prompt line as garbage like "11;rgb:0b0b/…".
	// COLORFGBG stops the bg-color probe; the rest keeps color detection
	// on without a round-trip.
	cmd.Env = append(os.Environ(),
		"TERM=xterm-256color",
		"COLORTERM=truecolor",
		"COLORFGBG=15;0",
	)

	master, err := pty.StartWithSize(cmd, &pty.Winsize{Cols: 100, Rows: 30})
	if err != nil {
		_ = os.Remove(cfgPath)
		return "", err
	}

	id := s.subproc.newJobID()
	job := &subprocJob{id: id, sessionID: sessionID, cmd: cmd, pty: master}
	s.subproc.add(job)
	// Drain any commands the GUI queued up for this session before the
	// subprocess was ready. Written after add() so a WriteConsole racing
	// against us finds the same job.
	if pending := s.subproc.takePending(sessionID); len(pending) > 0 {
		go func() {
			// A small delay lets sliver's readline finish drawing the
			// first prompt before we push lines at it. Without this the
			// first queued command sometimes lands before rc processing
			// completes and gets misparsed.
			time.Sleep(250 * time.Millisecond)
			for _, line := range pending {
				_, _ = master.Write([]byte(line + "\n"))
			}
		}()
	}
	go s.pumpSubproc(job)
	go func() {
		_ = cmd.Wait()
		s.subproc.remove(id)
		_ = master.Close()
		_ = os.Remove(cfgPath)
		s.emitConsoleExit(id, cmd.ProcessState)
	}()
	return id, nil
}

// SendToSessionConsole pushes a command line to the subprocess console
// bound to sessionID. If no console is running yet the line queues and
// is drained on the next StartConsole for the same session — callers
// don't have to synchronize with mount lifecycles.
func (s *Service) SendToSessionConsole(sessionID, line string) error {
	if line == "" {
		return nil
	}
	s.subproc.mu.Lock()
	jobID, ok := s.subproc.bySession[sessionID]
	s.subproc.mu.Unlock()
	if !ok {
		s.subproc.queuePending(sessionID, line)
		return nil
	}
	job := s.subproc.get(jobID)
	if job == nil {
		s.subproc.queuePending(sessionID, line)
		return nil
	}
	// Clear whatever the user has already typed at the prompt before
	// pushing our line: Ctrl+E (go to end of line) then Ctrl+U (kill
	// backward to start). Without this the injected command concatenates
	// onto their in-flight input and sliver parses the mash as one bad
	// command.
	_, err := job.pty.Write([]byte("\x05\x15" + line + "\n"))
	return err
}

func (s *Service) WriteConsole(jobID string, data []byte) error {
	job := s.subproc.get(jobID)
	if job == nil {
		return os.ErrClosed
	}
	_, err := job.pty.Write(data)
	return err
}

func (s *Service) ResizeConsole(jobID string, cols, rows int) error {
	job := s.subproc.get(jobID)
	if job == nil {
		return os.ErrClosed
	}
	if cols <= 0 || rows <= 0 {
		return nil
	}
	return pty.Setsize(job.pty, &pty.Winsize{Cols: uint16(cols), Rows: uint16(rows)})
}

func (s *Service) StopConsole(jobID string) error {
	job := s.subproc.get(jobID)
	if job == nil {
		return nil
	}
	if job.cmd.Process != nil {
		_ = job.cmd.Process.Signal(syscall.SIGTERM)
	}
	return nil
}

// CloseSubprocs kills every running console subprocess. Called at app
// shutdown so orphan sliver clients don't linger against the server.
func (s *Service) CloseSubprocs() {
	s.subproc.mu.Lock()
	jobs := s.subproc.jobs
	s.subproc.jobs = nil
	s.subproc.mu.Unlock()
	for _, j := range jobs {
		if j.cmd.Process != nil {
			_ = j.cmd.Process.Signal(syscall.SIGTERM)
		}
	}
}

func (s *Service) pumpSubproc(job *subprocJob) {
	buf := make([]byte, 32*1024)
	pending := make([]byte, 0, subprocOutputBatch)
	var controlCarry []byte
	ticker := time.NewTicker(subprocOutputInterval)
	defer ticker.Stop()
	chunks := make(chan []byte, 16)

	go func() {
		defer close(chunks)
		for {
			n, err := job.pty.Read(buf)
			if n > 0 {
				chunk := make([]byte, n)
				copy(chunk, buf[:n])
				chunks <- chunk
			}
			if err != nil {
				return
			}
		}
	}()

	flush := func() {
		if len(pending) == 0 {
			return
		}
		s.emitConsoleOutput(job.id, pending)
		pending = pending[:0]
	}
	for {
		select {
		case chunk, ok := <-chunks:
			if !ok {
				if len(controlCarry) > 0 {
					pending = append(pending, controlCarry...)
					controlCarry = nil
				}
				flush()
				return
			}
			visible, tails, carry := filterShellOpenFrames(controlCarry, chunk)
			controlCarry = carry
			for _, tail := range tails {
				s.emitConsoleOpenShell(job.id, job.sessionID, tail)
			}
			pending = append(pending, visible...)
			if len(pending) >= subprocOutputBatch {
				flush()
			}
		case <-ticker.C:
			flush()
		}
	}
}

func (s *Service) emitConsoleOutput(jobID string, data []byte) {
	if s.ctx == nil {
		return
	}
	wailsruntime.EventsEmit(s.ctx, "console-output", map[string]any{
		"jobID": jobID,
		"data":  base64.StdEncoding.EncodeToString(data),
	})
}

func (s *Service) emitConsoleExit(jobID string, state *os.ProcessState) {
	if s.ctx == nil {
		return
	}
	payload := map[string]any{"jobID": jobID}
	if state != nil {
		payload["exitCode"] = state.ExitCode()
	}
	wailsruntime.EventsEmit(s.ctx, "console-exit", payload)
}

func (s *Service) emitConsoleOpenShell(jobID, sessionID, tail string) {
	if s.ctx == nil {
		return
	}
	wailsruntime.EventsEmit(s.ctx, "console-open-shell", map[string]any{
		"jobID":     jobID,
		"sessionID": sessionID,
		"tail":      tail,
	})
}

func (m *subprocMgr) newJobID() string {
	return "console-" + strconv.FormatUint(m.next.Add(1), 10)
}

func (m *subprocMgr) add(j *subprocJob) {
	m.mu.Lock()
	if m.jobs == nil {
		m.jobs = make(map[string]*subprocJob)
	}
	if m.bySession == nil {
		m.bySession = make(map[string]string)
	}
	m.jobs[j.id] = j
	if j.sessionID != "" {
		m.bySession[j.sessionID] = j.id
	}
	m.mu.Unlock()
}

func (m *subprocMgr) remove(id string) {
	m.mu.Lock()
	if job, ok := m.jobs[id]; ok && job.sessionID != "" {
		if m.bySession[job.sessionID] == id {
			delete(m.bySession, job.sessionID)
		}
	}
	delete(m.jobs, id)
	m.mu.Unlock()
}

func (m *subprocMgr) queuePending(sessionID, line string) {
	m.mu.Lock()
	if m.pending == nil {
		m.pending = make(map[string][]string)
	}
	m.pending[sessionID] = append(m.pending[sessionID], line)
	m.mu.Unlock()
}

func (m *subprocMgr) takePending(sessionID string) []string {
	m.mu.Lock()
	lines := m.pending[sessionID]
	delete(m.pending, sessionID)
	m.mu.Unlock()
	return lines
}

func (m *subprocMgr) get(id string) *subprocJob {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.jobs[id]
}

// writeConfigForSubproc dumps the operator config to a private temp file
// so the child can re-establish the same mTLS connection. Deleted by the
// caller once the child has exited.
func writeConfigForSubproc(cfg *assets.ClientConfig) (string, error) {
	if cfg == nil {
		return "", os.ErrInvalid
	}
	f, err := os.CreateTemp("", "sliver-gui-console-*.json")
	if err != nil {
		return "", err
	}
	defer f.Close()
	if err := os.Chmod(f.Name(), 0o600); err != nil {
		return "", err
	}
	enc := json.NewEncoder(f)
	if err := enc.Encode(cfg); err != nil {
		_ = os.Remove(f.Name())
		return "", err
	}
	return f.Name(), nil
}

// RunConsoleSubprocess is the main entry for a --sliver-console re-exec.
// It reconnects to the server using the config the parent handed us,
// then hands control to sliver's own StartClient (which owns fd 0/1/2
// for readline). Session pinning is done via an rcScript containing
// `use <sessionID>` — this goes through sliver's real dispatch so menu
// switching, cwd, and any per-session bootstrap happen the same as if
// the user had typed it. Blocks until the user exits.
func RunConsoleSubprocess(configPath, sessionID string) error {
	cfg, err := loadConfigFromFile(configPath)
	if err != nil {
		return err
	}
	_ = os.Remove(configPath)

	// Sliver's tunnel/socks/log-stream internals dump into the standard
	// logger. Left visible those messages ("Set stream", "TunnelLoop
	// exited", etc.) mix with real console output — we don't want them
	// in the user's terminal.
	stdlog.SetOutput(io.Discard)

	rpcClient, grpcConn, err := transport.MTLSConnect(cfg)
	if err != nil {
		return err
	}
	defer grpcConn.Close()

	// rcScript is the script *body*, not a path (sliver reads it directly
	// via bufio.NewScanner(strings.NewReader(...))). Passing `use <id>`
	// makes the console drop into the implant menu on the first prompt.
	rcScript := ""
	if sessionID != "" {
		rcScript = "use " + sessionID + "\n"
	}

	con := sliverconsole.NewConsole(false)
	serverCmds := command.ServerCommands(con, nil)
	sliverCmds := command.SliverCommands(con)
	if sessionID != "" {
		serverCmds = pinServerTargetCommands(serverCmds, sessionID, con)
		sliverCmds = pinSliverTargetCommands(sliverCmds, sessionID, con)
	}
	details := &sliverconsole.ConnectionDetails{Config: cfg}
	return sliverconsole.StartClient(con, rpcClient, grpcConn, details, serverCmds, sliverCmds, true, rcScript)
}

func pinServerTargetCommands(base reefconsole.Commands, sessionID string, con *sliverconsole.SliverClient) reefconsole.Commands {
	return func() *cobra.Command {
		root := base()
		if useCmd := findTopLevelCommand(root, "use"); useCmd != nil {
			pinUseCommand(useCmd, sessionID, con)
		}
		if sessionsCmd := findTopLevelCommand(root, "sessions"); sessionsCmd != nil {
			pinSessionsCommand(sessionsCmd, sessionID, con)
		}
		return root
	}
}

func pinSliverTargetCommands(base reefconsole.Commands, sessionID string, con *sliverconsole.SliverClient) reefconsole.Commands {
	return func() *cobra.Command {
		root := base()
		if backgroundCmd := findTopLevelCommand(root, "background"); backgroundCmd != nil {
			wrapCommandRun(backgroundCmd, func(*cobra.Command, []string) bool {
				return false
			}, func() {
				printPinnedConsoleMessage(con, sessionID)
			})
		}
		if shellCmd := findTopLevelCommand(root, "shell"); shellCmd != nil {
			wrapCommandRun(shellCmd, func(command *cobra.Command, args []string) bool {
				emitShellOpenFrame(shellCommandTail(command, args))
				return false
			}, func() {})
		}
		return root
	}
}

func pinUseCommand(cmd *cobra.Command, sessionID string, con *sliverconsole.SliverClient) {
	wrapCommandRun(cmd, func(command *cobra.Command, args []string) bool {
		return command.Name() == "use" && len(args) == 1 && matchesPinnedTarget(args[0], sessionID)
	}, func() {
		printPinnedConsoleMessage(con, sessionID)
	})
	for _, child := range cmd.Commands() {
		pinUseSubcommands(child, sessionID, con)
	}
}

func pinUseSubcommands(cmd *cobra.Command, sessionID string, con *sliverconsole.SliverClient) {
	wrapCommandRun(cmd, func(*cobra.Command, []string) bool {
		return false
	}, func() {
		printPinnedConsoleMessage(con, sessionID)
	})
	for _, child := range cmd.Commands() {
		pinUseSubcommands(child, sessionID, con)
	}
}

func pinSessionsCommand(cmd *cobra.Command, sessionID string, con *sliverconsole.SliverClient) {
	wrapCommandRun(cmd, func(command *cobra.Command, _ []string) bool {
		interact, _ := command.Flags().GetString("interact")
		return interact == "" || matchesPinnedTarget(interact, sessionID)
	}, func() {
		printPinnedConsoleMessage(con, sessionID)
	})
}

func wrapCommandRun(cmd *cobra.Command, allow func(*cobra.Command, []string) bool, reject func()) {
	originalRun := cmd.Run
	originalRunE := cmd.RunE
	cmd.Run = func(command *cobra.Command, args []string) {
		if !allow(command, args) {
			reject()
			return
		}
		if originalRun != nil {
			originalRun(command, args)
			return
		}
		if originalRunE != nil {
			if err := originalRunE(command, args); err != nil {
				command.PrintErrln(err)
			}
		}
	}
	cmd.RunE = nil
}

func findTopLevelCommand(root *cobra.Command, name string) *cobra.Command {
	if root == nil {
		return nil
	}
	for _, command := range root.Commands() {
		if command.Name() == name {
			return command
		}
	}
	return nil
}

func matchesPinnedTarget(candidate, sessionID string) bool {
	candidate = strings.TrimSpace(candidate)
	if candidate == "" {
		return false
	}
	return candidate == sessionID || (len(candidate) >= 8 && strings.HasPrefix(sessionID, candidate))
}

func printPinnedConsoleMessage(con *sliverconsole.SliverClient, sessionID string) {
	shortID := sessionID
	if len(shortID) > 8 {
		shortID = shortID[:8]
	}
	con.PrintErrorf("This embedded console is pinned to %s. Open another agent's Console tab to switch targets.\n", shortID)
}

func shellCommandTail(cmd *cobra.Command, args []string) string {
	if cmd != nil {
		if shellPath, _ := cmd.Flags().GetString("shell-path"); shellPath != "" {
			return shellPath
		}
	}
	return strings.Join(args, " ")
}

func emitShellOpenFrame(tail string) {
	payload := base64.StdEncoding.EncodeToString([]byte(tail))
	_, _ = os.Stdout.WriteString(shellOpenFramePrefix + payload + shellOpenFrameSuffix)
}

func filterShellOpenFrames(carry, chunk []byte) ([]byte, []string, []byte) {
	buf := make([]byte, 0, len(carry)+len(chunk))
	buf = append(buf, carry...)
	buf = append(buf, chunk...)

	prefix := []byte(shellOpenFramePrefix)
	suffix := []byte(shellOpenFrameSuffix)
	visible := make([]byte, 0, len(buf))
	var tails []string

	for len(buf) > 0 {
		idx := bytes.Index(buf, prefix)
		if idx < 0 {
			keep := prefixSuffixLen(buf, prefix)
			visible = append(visible, buf[:len(buf)-keep]...)
			nextCarry := append([]byte(nil), buf[len(buf)-keep:]...)
			return visible, tails, nextCarry
		}

		visible = append(visible, buf[:idx]...)
		rest := buf[idx+len(prefix):]
		end := bytes.Index(rest, suffix)
		if end < 0 {
			nextCarry := append([]byte(nil), buf[idx:]...)
			return visible, tails, nextCarry
		}

		payload := rest[:end]
		if decoded, err := base64.StdEncoding.DecodeString(string(payload)); err == nil {
			tails = append(tails, string(decoded))
		}
		buf = rest[end+len(suffix):]
	}

	return visible, tails, nil
}

func prefixSuffixLen(buf, prefix []byte) int {
	max := len(prefix) - 1
	if len(buf) < max {
		max = len(buf)
	}
	for n := max; n > 0; n-- {
		if bytes.Equal(buf[len(buf)-n:], prefix[:n]) {
			return n
		}
	}
	return 0
}

func loadConfigFromFile(path string) (*assets.ClientConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg assets.ClientConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
