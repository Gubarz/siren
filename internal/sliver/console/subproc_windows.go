//go:build windows

package console

import (
	"fmt"
	"os"
	"strings"
)

// subprocCommandTerminator submits a line to the subprocess console.
// Console input under ConPTY delivers Enter as a carriage return.
const subprocCommandTerminator = "\r"

// StartConsole spawns a subprocess that runs a real sliver client
// console (readline + all commands) attached to a fresh ConPTY pseudo
// console. The subprocess gets the pseudoconsole as its console at
// creation, so sliver's readline (CONIN$/console API) works exactly as
// it does in Windows Terminal. Returns a jobID the frontend uses for I/O.
func (s *Service) StartConsole(sessionID string) (string, error) {
	id, _, err := s.AcquireConsole(sessionID)
	return id, err
}

// AcquireConsole starts or attaches to the session console and reports whether
// the returned job was already running. Attached terminals use this to avoid
// answering terminal queries found in replayed history.
func (s *Service) AcquireConsole(sessionID string) (string, bool, error) {
	if !validSessionID(sessionID) {
		return "", false, fmt.Errorf("refusing to open a console for a malformed session id")
	}
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
	// serialized ClientConfig via a %TEMP% file, deleted after read.
	cfgPath, err := writeConfigForSubproc(s.rpc.Config)
	if err != nil {
		return "", false, err
	}

	cpty, err := startConPTY(winQuoteArgs(append([]string{self}, ConsoleModeFlag, cfgPath, sessionID)), 100, 30)
	if err != nil {
		_ = os.Remove(cfgPath)
		return "", false, err
	}

	id := s.subproc.newJobID()
	job := &subprocJob{id: id, sessionID: sessionID, proc: cpty, pty: cpty}
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
	cpty, ok := job.pty.(*winConPTY)
	if !ok {
		return os.ErrInvalid
	}
	return cpty.Resize(int16(cols), int16(rows))
}

// winQuoteArgs joins arguments into a CreateProcess command line using
// CommandLineToArgvW-compatible quoting.
func winQuoteArgs(args []string) string {
	quoted := make([]string, len(args))
	for i, arg := range args {
		quoted[i] = winQuoteArg(arg)
	}
	return strings.Join(quoted, " ")
}

func winQuoteArg(s string) string {
	if s != "" && !strings.ContainsAny(s, " \t\"") {
		return s
	}
	var b strings.Builder
	b.WriteByte('"')
	slashes := 0
	for _, r := range s {
		switch r {
		case '\\':
			slashes++
			continue
		case '"':
			b.WriteString(strings.Repeat(`\`, slashes*2+1))
			b.WriteByte('"')
			slashes = 0
			continue
		}
		b.WriteString(strings.Repeat(`\`, slashes))
		slashes = 0
		b.WriteRune(r)
	}
	b.WriteString(strings.Repeat(`\`, slashes*2))
	b.WriteByte('"')
	return b.String()
}
