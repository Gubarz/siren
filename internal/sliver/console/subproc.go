//go:build unix

package console

import (
	"os"
	"os/exec"
	"syscall"

	"siren/internal/envvars"

	"github.com/creack/pty"
)

// subprocCommandTerminator submits a line to the subprocess console.
// On a unix PTY the Enter key arrives as a newline.
const subprocCommandTerminator = "\n"

// StartConsole spawns a subprocess that runs a real sliver client
// console (readline + all commands) with its stdio wired to a fresh PTY.
// Interactive prompts work because the subprocess owns fd 0/1/2 outright
// — no in-process hijack. Returns a jobID the frontend uses for I/O.
func (s *Service) StartConsole(sessionID string) (string, error) {
	id, _, err := s.AcquireConsole(sessionID)
	return id, err
}

// AcquireConsole starts or attaches to the session console and reports whether
// the returned job was already running. A newly started console still needs a
// terminal to answer its startup capability queries; an attached terminal must
// not answer queries found in replayed history.
func (s *Service) AcquireConsole(sessionID string) (string, bool, error) {
	s.subprocStart.Lock()
	defer s.subprocStart.Unlock()
	if id := s.subproc.acquireSession(sessionID); id != "" {
		return id, true, nil
	}

	self, err := os.Executable()
	if err != nil {
		return "", false, err
	}

	// The subprocess needs the operator config to reconnect. We pass the
	// serialized ClientConfig via a $TMPDIR file, deleted after read.
	cfgPath, err := writeConfigForSubproc(s.rpc.Config)
	if err != nil {
		return "", false, err
	}

	cmd := exec.Command(self, ConsoleModeFlag, cfgPath, sessionID)
	// Pre-populate terminal capability env vars so termenv/lipgloss don't
	// probe with OSC queries (\x1b]11;?\x07 etc). xterm.js answers those
	// truthfully but sliver reads the reply back as user input, which
	// then shows up on the prompt line as garbage like "11;rgb:0b0b/…".
	// COLORFGBG stops the bg-color probe; the rest keeps color detection
	// on without a round-trip.
	cmd.Env = envvars.BuildPassthroughEnv(
		"TERM=xterm-256color",
		"COLORTERM=truecolor",
		"COLORFGBG=15;0",
	)

	master, err := pty.StartWithSize(cmd, &pty.Winsize{Cols: 100, Rows: 30})
	if err != nil {
		_ = os.Remove(cfgPath)
		return "", false, err
	}

	id := s.subproc.newJobID()
	job := &subprocJob{id: id, sessionID: sessionID, proc: &unixProc{cmd: cmd}, pty: master}
	s.subproc.add(job)
	s.drainPendingCommands(job)
	go s.pumpSubproc(job)
	go s.watchConsole(job, cfgPath)
	return id, false, nil
}

func (s *Service) ResizeConsole(jobID string, cols, rows int) error {
	job := s.subproc.get(jobID)
	if job == nil {
		return os.ErrClosed
	}
	if cols <= 0 || rows <= 0 {
		return nil
	}
	ptyFile, ok := job.pty.(*os.File)
	if !ok {
		return os.ErrInvalid
	}
	return pty.Setsize(ptyFile, &pty.Winsize{Cols: uint16(cols), Rows: uint16(rows)})
}

// unixProc adapts exec.Cmd to consoleProc.
type unixProc struct {
	cmd *exec.Cmd
}

func (p *unixProc) Wait() error { return p.cmd.Wait() }

func (p *unixProc) Kill() error {
	if p.cmd.Process == nil {
		return nil
	}
	return p.cmd.Process.Signal(syscall.SIGTERM)
}

func (p *unixProc) ExitCode() int {
	if p.cmd.ProcessState == nil {
		return -1
	}
	return p.cmd.ProcessState.ExitCode()
}
