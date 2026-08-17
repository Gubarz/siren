package console

import (
	"encoding/json"
	"io"
	"os"
	"strconv"
	"sync"
	"sync/atomic"

	"github.com/bishopfox/sliver/client/assets"
)

// ConsoleModeFlag names the CLI flag that puts a re-exec of this binary
// into "run a sliver client console" mode. main.go checks for it before
// spawning the wails app.
const ConsoleModeFlag = "--sliver-console"

// consolePTY is the GUI-side end of the console subprocess's terminal:
// unix PTY master on unix, ConPTY pipes on Windows. Writes land on the
// subprocess's stdin (readline input), reads carry its rendered output.
type consolePTY interface {
	io.Reader
	io.Writer
	Close() error
}

// consoleProc is the running console subprocess itself. The platform
// files provide the concrete implementation (exec.Cmd on unix, a raw
// process handle on Windows where ConPTY owns process creation).
type consoleProc interface {
	// Wait blocks until the subprocess exits.
	Wait() error
	// Kill terminates the subprocess.
	Kill() error
	// ExitCode reports the subprocess exit code after Wait returns.
	ExitCode() int
}

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
	id         string
	sessionID  string
	proc       consoleProc
	pty        consolePTY
	refs       int
	stopping   bool
	outputMu   sync.RWMutex
	output     []byte
	promptMu   sync.Mutex
	promptLine string
}

const consoleOutputReplayMaxBytes = 8 * 1024 * 1024

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
	if j.refs == 0 {
		j.refs = 1
	}
	if j.sessionID != "" {
		m.bySession[j.sessionID] = j.id
	}
	m.mu.Unlock()
}

func (m *subprocMgr) acquireSession(sessionID string) string {
	m.mu.Lock()
	defer m.mu.Unlock()
	id := m.bySession[sessionID]
	job := m.jobs[id]
	if job == nil {
		return ""
	}
	job.refs++
	return id
}

func (m *subprocMgr) release(id string) (*subprocJob, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	job := m.jobs[id]
	if job == nil {
		return nil, false
	}
	if job.refs > 0 {
		job.refs--
	}
	if job.refs != 0 || job.stopping {
		return job, false
	}
	job.stopping = true
	return job, true
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

func (j *subprocJob) appendOutput(data []byte) {
	if len(data) == 0 {
		return
	}
	j.outputMu.Lock()
	j.output = append(j.output, data...)
	if len(j.output) > consoleOutputReplayMaxBytes {
		trim := len(j.output) - consoleOutputReplayMaxBytes
		copy(j.output, j.output[trim:])
		j.output = j.output[:consoleOutputReplayMaxBytes]
	}
	j.outputMu.Unlock()
}

func (j *subprocJob) outputSnapshot() []byte {
	j.outputMu.RLock()
	defer j.outputMu.RUnlock()
	return append([]byte(nil), j.output...)
}

// writeConfigForSubproc dumps the operator config to a private temp file
// so the child can re-establish the same mTLS connection. Deleted by the
// caller once the child has exited.
func writeConfigForSubproc(cfg *assets.ClientConfig) (string, error) {
	if cfg == nil {
		return "", os.ErrInvalid
	}
	f, err := os.CreateTemp("", "siren-console-*.json")
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()
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
