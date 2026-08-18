package shells

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/bishopfox/sliver/client/core"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"

	"siren/internal/journal"
	"siren/internal/sliver/console"
	"siren/internal/sliver/rpc"
	"siren/internal/wailsadapter"
)

type ShellInfo struct {
	ID        string `json:"id"`
	SessionID string `json:"sessionID"`
	Path      string `json:"path"`
	PID       uint32 `json:"pid"`
	PTY       bool   `json:"pty"`
}

type guiShell struct {
	info      ShellInfo
	tunnel    *core.TunnelIO
	mu        sync.RWMutex
	output    []byte
	inputTail string
}

type Service struct {
	rpc       *rpc.Client
	console   *console.Service
	journal   *journal.Service
	ui        *wailsadapter.Bridge
	shellMu   sync.RWMutex
	shells    map[string]*guiShell
	nextShell atomic.Uint64
}

func New(rpc *rpc.Client, con *console.Service) *Service {
	return &Service{
		rpc:     rpc,
		console: con,
		shells:  make(map[string]*guiShell),
	}
}

func (s *Service) SetJournal(j *journal.Service) {
	s.journal = j
}

func (s *Service) SetUI(ui *wailsadapter.Bridge) {
	s.ui = ui
}

func prepareShellParams(sessionOS string, enablePTY bool, rows, cols uint32) (bool, uint32, uint32) {
	osName := strings.ToLower(sessionOS)
	enablePTY = enablePTY && (osName == "linux" || osName == "darwin")
	if rows == 0 {
		rows = 24
	}
	if cols == 0 {
		cols = 80
	}
	return enablePTY, rows, cols
}

func (s *Service) StartShell(sessionID, path string, enablePTY bool, rows, cols uint32) (*ShellInfo, error) {
	if !s.rpc.Connected() {
		return nil, rpc.ErrNotConnected
	}

	session, beacon, err := s.console.FindTarget(sessionID)
	if err != nil {
		return nil, err
	}
	if session == nil || beacon != nil {
		return nil, fmt.Errorf("interactive shells require a live session")
	}

	enablePTY, rows, cols = prepareShellParams(session.OS, enablePTY, rows, cols)

	tunnel, shellPath, pid, err := s.openShellTunnel(sessionID, path, enablePTY, rows, cols)
	if err != nil {
		return nil, err
	}

	return s.registerShell(sessionID, shellPath, pid, enablePTY, tunnel), nil
}

func (s *Service) registerShell(sessionID, shellPath string, pid uint32, enablePTY bool, tunnel *core.TunnelIO) *ShellInfo {
	id := fmt.Sprintf("shell-%d", s.nextShell.Add(1))
	info := ShellInfo{
		ID: id, SessionID: sessionID, Path: shellPath,
		PID: pid, PTY: enablePTY,
	}

	s.shellMu.Lock()
	s.shells[id] = &guiShell{info: info, tunnel: tunnel}
	s.shellMu.Unlock()

	go s.readShell(id, tunnel)
	return &info
}

func (s *Service) WriteShell(id, data string) error {
	shell, err := s.getShell(id)
	if err != nil {
		return err
	}
	if !shell.info.PTY {
		data = strings.ReplaceAll(data, "\x03", "")
		if data == "" {
			return nil
		}
	}
	_, err = shell.tunnel.Write([]byte(data))
	if err != nil {
		return err
	}
	if !shell.info.PTY {
		s.journalShellInput(shell, data)
	}
	return nil
}

func (s *Service) journalShellInput(shell *guiShell, data string) {
	if s.journal == nil {
		return
	}
	shell.mu.Lock()
	combined := shell.inputTail + data
	lastNewline := strings.LastIndex(combined, "\n")
	if lastNewline < 0 {
		shell.inputTail = combined
		shell.mu.Unlock()
		return
	}
	complete := combined[:lastNewline]
	shell.inputTail = combined[lastNewline+1:]
	shell.mu.Unlock()

	for _, line := range strings.Split(complete, "\n") {
		line = strings.TrimRight(line, "\r")
		if strings.TrimSpace(line) == "" {
			continue
		}
		s.journal.Record(journal.Entry{
			Verb:        "ShellInput",
			CommandLine: line,
			TargetID:    shell.info.SessionID,
			TargetKind:  "session",
			ActorKind:   "operator",
			Panel:       "shell",
			Status:      "ok",
		})
	}
}

func (s *Service) InterruptShell(id string) (bool, error) {
	shell, err := s.getShell(id)
	if err != nil {
		return false, err
	}
	if shell.info.PTY {
		_, err = shell.tunnel.Write([]byte{3})
		return false, err
	}

	s.shellMu.Lock()
	delete(s.shells, id)
	s.shellMu.Unlock()
	return true, s.closeTunnel(shell.tunnel.ID, shell.info.SessionID)
}

func (s *Service) GetShellOutput(id string) (string, error) {
	s.shellMu.RLock()
	shell := s.shells[id]
	s.shellMu.RUnlock()
	if shell == nil {
		return "", fmt.Errorf("shell %q was not found", id)
	}
	shell.mu.RLock()
	defer shell.mu.RUnlock()
	return string(shell.output), nil
}

func (s *Service) ResizeShell(id string, rows, cols uint32) error {
	shell, err := s.getShell(id)
	if err != nil {
		return err
	}
	if !shell.info.PTY || rows == 0 || cols == 0 {
		return nil
	}
	_, err = s.rpc.RPC.ShellResize(context.Background(), &sliverpb.ShellResizeReq{
		Request:  &commonpb.Request{SessionID: shell.info.SessionID, Timeout: int64(9 * time.Second)},
		Rows:     rows,
		Cols:     cols,
		TunnelID: shell.tunnel.ID,
	})
	return err
}

func (s *Service) CloseShell(id string) error {
	s.shellMu.Lock()
	shell := s.shells[id]
	delete(s.shells, id)
	s.shellMu.Unlock()
	if shell == nil {
		return nil
	}

	_, _ = shell.tunnel.Write([]byte("exit\n"))
	return s.closeTunnel(shell.tunnel.ID, shell.info.SessionID)
}

// Close terminates every open GUI shell tunnel. Called during app shutdown so
// active shells don't get orphaned on the sliver server when the client's
// gRPC connection drops.
func (s *Service) Close() {
	s.shellMu.Lock()
	shells := s.shells
	s.shells = map[string]*guiShell{}
	s.shellMu.Unlock()
	for _, shell := range shells {
		if shell == nil {
			continue
		}
		_ = s.closeTunnel(shell.tunnel.ID, shell.info.SessionID)
	}
}

func (s *Service) getShell(id string) (*guiShell, error) {
	s.shellMu.RLock()
	shell := s.shells[id]
	s.shellMu.RUnlock()
	if shell == nil || !shell.tunnel.IsOpen() {
		return nil, fmt.Errorf("shell %q is closed", id)
	}
	return shell, nil
}
