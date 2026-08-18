package shells

import (
	"context"
	"io"
	"log"
	"strings"
	"time"

	"github.com/bishopfox/sliver/client/core"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"

	"siren/internal/sliver/rpc"
)

func (s *Service) cleanUpTunnelOnError(tunnelID uint64, sessionID string, contextStr string) {
	if closeErr := s.closeTunnel(tunnelID, sessionID); closeErr != nil {
		log.Printf("shell: failed to close tunnel %s: %v", contextStr, closeErr)
	}
}

// openShellTunnel creates the tunnel and starts the remote shell over it,
// tearing the tunnel down if the shell request fails. Returns the resolved
// shell path (the requested one, or whatever the target picked).
func (s *Service) openShellTunnel(sessionID, path string, enablePTY bool, rows, cols uint32) (*core.TunnelIO, string, uint32, error) {
	rpcTunnel, err := s.rpc.RPC.CreateTunnel(context.Background(), &sliverpb.Tunnel{
		SessionID: sessionID,
	})
	if err != nil {
		return nil, "", 0, err
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
		return nil, "", 0, err
	}
	if err := rpc.CheckResponse(response); err != nil {
		s.cleanUpTunnelOnError(tunnel.ID, sessionID, "after remote shell error")
		return nil, "", 0, err
	}

	shellPath := strings.TrimSpace(path)
	if shellPath == "" {
		shellPath = response.Path
	}
	return tunnel, shellPath, response.Pid, nil
}

func (s *Service) readShell(id string, tunnel *core.TunnelIO) {
	buffer := make([]byte, 64*1024)
	outputEvents := make(chan []byte, 64)
	eventsDone := make(chan struct{})
	go s.emitShellOutput(id, outputEvents, eventsDone)

	readErr := s.pumpShellOutput(id, tunnel, buffer, outputEvents)

	close(outputEvents)
	<-eventsDone
	s.finishShell(id, tunnel, readErr)
}

// pumpShellOutput forwards tunnel reads into the shell's bounded buffer and
// the event channel until the tunnel errors or hits EOF.
func (s *Service) pumpShellOutput(id string, tunnel *core.TunnelIO, buffer []byte, outputEvents chan<- []byte) error {
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
			return readErr
		}
	}
}

func (s *Service) finishShell(id string, tunnel *core.TunnelIO, readErr error) {
	if readErr != nil {
		s.ui.Emit("shell-output", map[string]interface{}{
			"id": id, "error": readErr.Error(),
		})
	}
	s.ui.Emit("shell-output", map[string]interface{}{"id": id, "closed": true})

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

func (s *Service) closeTunnel(tunnelID uint64, sessionID string) error {
	core.GetTunnels().Close(tunnelID)
	_, err := s.rpc.RPC.CloseTunnel(context.Background(), &sliverpb.Tunnel{
		TunnelID:  tunnelID,
		SessionID: sessionID,
	})
	return err
}
