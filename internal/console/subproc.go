//go:build unix

package console

import (
	"encoding/base64"
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/creack/pty"
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
	s.drainPendingCommands(job)
	go s.pumpSubproc(job)
	go s.watchConsole(job, cfgPath)
	return id, nil
}

// drainPendingCommands flushes any commands the GUI queued for this session
// before the subprocess was ready. Called after add() so a WriteConsole
// racing against us finds the same job.
func (s *Service) drainPendingCommands(job *subprocJob) {
	pending := s.subproc.takePending(job.sessionID)
	if len(pending) == 0 {
		return
	}
	go func() {
		// A small delay lets sliver's readline finish drawing the first
		// prompt before we push lines at it. Without this the first queued
		// command sometimes lands before rc processing completes and gets
		// misparsed.
		time.Sleep(250 * time.Millisecond)
		for _, line := range pending {
			_, _ = job.pty.Write([]byte(line + "\n"))
		}
	}()
}

// watchConsole reaps the subprocess and releases its resources on exit.
func (s *Service) watchConsole(job *subprocJob, cfgPath string) {
	_ = job.cmd.Wait()
	s.subproc.remove(job.id)
	_ = job.pty.Close()
	_ = os.Remove(cfgPath)
	s.emitConsoleExit(job.id, job.cmd.ProcessState)
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
	chunks := make(chan []byte, 16)
	go readPTYChunks(job.pty, chunks)

	pending := make([]byte, 0, subprocOutputBatch)
	var controlCarry []byte
	ticker := time.NewTicker(subprocOutputInterval)
	defer ticker.Stop()

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

// readPTYChunks copies PTY reads onto chunks until the PTY errors or closes.
func readPTYChunks(ptyFile *os.File, chunks chan<- []byte) {
	defer close(chunks)
	buf := make([]byte, 32*1024)
	for {
		n, err := ptyFile.Read(buf)
		if n > 0 {
			chunk := make([]byte, n)
			copy(chunk, buf[:n])
			chunks <- chunk
		}
		if err != nil {
			return
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
