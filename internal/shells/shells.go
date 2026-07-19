package shells

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/bishopfox/sliver/client/core"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/wailsapp/wails/v2/pkg/runtime"

	"sliver-gui/internal/console"
	"sliver-gui/internal/rpc"
)

const (
	maxShellOutputBytes = 4 * 1024 * 1024
	shellTrimTarget     = 3 * 1024 * 1024
	shellEventBatchSize = 512 * 1024
	shellEventInterval  = 100 * time.Millisecond
)

type ShellInfo struct {
	ID        string `json:"id"`
	SessionID string `json:"sessionID"`
	Path      string `json:"path"`
	PID       uint32 `json:"pid"`
	PTY       bool   `json:"pty"`
}

type guiShell struct {
	info   ShellInfo
	tunnel *core.TunnelIO
	mu     sync.RWMutex
	output []byte
}

type Service struct {
	rpc       *rpc.Client
	console   *console.Service
	ctx       context.Context
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

func (s *Service) SetCtx(ctx context.Context) {
	s.ctx = ctx
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

func (s *Service) cleanUpTunnelOnError(tunnelID uint64, sessionID string, contextStr string) {
	if closeErr := s.closeTunnel(tunnelID, sessionID); closeErr != nil {
		log.Printf("shell: failed to close tunnel %s: %v", contextStr, closeErr)
	}
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

	rpcTunnel, err := s.rpc.RPC.CreateTunnel(context.Background(), &sliverpb.Tunnel{
		SessionID: sessionID,
	})
	if err != nil {
		return nil, err
	}
	tunnel := core.GetTunnels().Start(rpcTunnel.TunnelID, rpcTunnel.SessionID)

	response, err := s.rpc.RPC.Shell(context.Background(), &sliverpb.ShellReq{
		Request: &commonpb.Request{
			SessionID: sessionID,
			Timeout:   int64(59 * time.Second),
		},
		Path:      strings.TrimSpace(path),
		EnablePTY: enablePTY,
		Rows:      rows,
		Cols:      cols,
		TunnelID:  tunnel.ID,
	})
	if err != nil {
		s.cleanUpTunnelOnError(tunnel.ID, sessionID, "after shell error")
		return nil, err
	}
	if err := rpc.CheckResponse(response); err != nil {
		s.cleanUpTunnelOnError(tunnel.ID, sessionID, "after remote shell error")
		return nil, err
	}

	id := fmt.Sprintf("shell-%d", s.nextShell.Add(1))
	shellPath := strings.TrimSpace(path)
	if shellPath == "" {
		shellPath = response.Path
	}
	info := ShellInfo{
		ID: id, SessionID: sessionID, Path: shellPath,
		PID: response.Pid, PTY: enablePTY,
	}

	s.shellMu.Lock()
	s.shells[id] = &guiShell{info: info, tunnel: tunnel}
	s.shellMu.Unlock()

	go s.readShell(id, tunnel)
	return &info, nil
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
	return err
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

func (s *Service) readShell(id string, tunnel *core.TunnelIO) {
	buffer := make([]byte, 64*1024)
	outputEvents := make(chan []byte, 64)
	eventsDone := make(chan struct{})
	go s.emitShellOutput(id, outputEvents, eventsDone)

	var readErr error
	for {
		n, err := tunnel.Read(buffer)
		if n > 0 {
			chunk := append([]byte(nil), buffer[:n]...)
			s.shellMu.RLock()
			shell := s.shells[id]
			s.shellMu.RUnlock()
			if shell != nil {
				shell.mu.Lock()
				shell.output = appendBoundedShellOutput(shell.output, chunk)
				shell.mu.Unlock()
			}
			outputEvents <- chunk
		}
		if err != nil {
			if err != io.EOF {
				readErr = err
			}
			break
		}
	}

	close(outputEvents)
	<-eventsDone
	if readErr != nil {
		runtime.EventsEmit(s.ctx, "shell-output", map[string]interface{}{
			"id": id, "error": readErr.Error(),
		})
	}
	runtime.EventsEmit(s.ctx, "shell-output", map[string]interface{}{"id": id, "closed": true})

	s.shellMu.Lock()
	shell := s.shells[id]
	delete(s.shells, id)
	s.shellMu.Unlock()

	if shell != nil {
		if closeErr := s.closeTunnel(tunnel.ID, shell.info.SessionID); closeErr != nil {
			log.Printf("shell: failed to close tunnel on shell exit: %v", closeErr)
		}
	}
}

func (s *Service) emitShellOutput(id string, chunks <-chan []byte, done chan<- struct{}) {
	defer close(done)
	ticker := time.NewTicker(shellEventInterval)
	defer ticker.Stop()

	pending := make([]byte, 0, shellEventBatchSize)
	flush := func() {
		if len(pending) == 0 {
			return
		}
		runtime.EventsEmit(s.ctx, "shell-output", map[string]interface{}{
			"id": id, "data": string(pending),
		})
		pending = pending[:0]
	}

	for {
		select {
		case chunk, ok := <-chunks:
			if !ok {
				flush()
				return
			}
			pending = append(pending, chunk...)
			if len(pending) >= shellEventBatchSize {
				flush()
			}
		case <-ticker.C:
			flush()
		}
	}
}

func (s *Service) closeTunnel(tunnelID uint64, sessionID string) error {
	core.GetTunnels().Close(tunnelID)
	_, err := s.rpc.RPC.CloseTunnel(context.Background(), &sliverpb.Tunnel{
		TunnelID:  tunnelID,
		SessionID: sessionID,
	})
	return err
}

func appendBoundedShellOutput(output, chunk []byte) []byte {
	output = append(output, chunk...)
	if len(output) <= maxShellOutputBytes {
		return output
	}

	start := len(output) - shellTrimTarget
	if newline := bytes.IndexByte(output[start:], '\n'); newline >= 0 {
		start += newline + 1
	}
	trimmed := make([]byte, len(output)-start)
	copy(trimmed, output[start:])
	return trimmed
}
