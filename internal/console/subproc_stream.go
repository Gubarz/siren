package console

import (
	"encoding/base64"
	"os"
	"time"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

const (
	subprocOutputInterval = 16 * time.Millisecond
	subprocOutputBatch    = 16 * 1024
)

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
			_, _ = job.pty.Write([]byte(line + subprocCommandTerminator))
		}
	}()
}

// watchConsole reaps the subprocess and releases its resources on exit.
func (s *Service) watchConsole(job *subprocJob, cfgPath string) {
	_ = job.proc.Wait()
	s.subproc.remove(job.id)
	_ = job.pty.Close()
	_ = os.Remove(cfgPath)
	s.emitConsoleExit(job.id, job.proc.ExitCode())
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
	_, err := job.pty.Write([]byte("\x05\x15" + line + subprocCommandTerminator))
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

func (s *Service) StopConsole(jobID string) error {
	job := s.subproc.get(jobID)
	if job == nil {
		return nil
	}
	return job.proc.Kill()
}

// CloseSubprocs kills every running console subprocess. Called at app
// shutdown so orphan sliver clients don't linger against the server.
func (s *Service) CloseSubprocs() {
	s.subproc.mu.Lock()
	jobs := s.subproc.jobs
	s.subproc.jobs = nil
	s.subproc.mu.Unlock()
	for _, j := range jobs {
		_ = j.proc.Kill()
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
func readPTYChunks(ptyFile consolePTY, chunks chan<- []byte) {
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

func (s *Service) emitConsoleExit(jobID string, exitCode int) {
	if s.ctx == nil {
		return
	}
	wailsruntime.EventsEmit(s.ctx, "console-exit", map[string]any{
		"jobID":    jobID,
		"exitCode": exitCode,
	})
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
