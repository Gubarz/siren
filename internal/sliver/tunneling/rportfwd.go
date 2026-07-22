package tunneling

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/bishopfox/sliver/protobuf/commonpb"
	sliverpb "github.com/bishopfox/sliver/protobuf/sliverpb"

	"sliver-gui/internal/sliver/rpc"
)

// rportfwdInfo tracks a reverse port forward we started, so we can list/stop
// it with the correct SessionID (the RPC won't route without one).
type rportfwdInfo struct {
	info      ProxyInfo
	sessionID string
}

func (s *Service) StartRportfwd(sessionID, bindAddr string, bindPort int, forwardAddr string, forwardPort int) (uint64, error) {
	// The implant handler ignores BindPort/ForwardPort and parses the port out
	// of BindAddress/ForwardAddress, so we must send them as full "host:port".
	fullBind := net.JoinHostPort(bindAddr, strconv.Itoa(bindPort))
	fullForward := net.JoinHostPort(forwardAddr, strconv.Itoa(forwardPort))
	return s.startRportfwd(sessionID, fullBind, fullForward)
}

func (s *Service) startRportfwd(sessionID, fullBind, fullForward string) (uint64, error) {
	if !s.rpc.Connected() {
		return 0, rpc.ErrNotConnected
	}

	resp, err := s.rpc.RPC.StartRportFwdListener(context.Background(), &sliverpb.RportFwdStartListenerReq{
		Request:        &commonpb.Request{SessionID: sessionID},
		BindAddress:    fullBind,
		ForwardAddress: fullForward,
	})
	if err != nil {
		return 0, err
	}
	if resp == nil {
		return 0, fmt.Errorf("start rportfwd: empty response")
	}
	if resp.Response != nil && resp.Response.Err != "" {
		return 0, fmt.Errorf("implant: %s", resp.Response.Err)
	}
	id := uint64(resp.ID)
	s.mu.Lock()
	s.rpfwd[rportfwdKey{sessionID: sessionID, id: id}] = &rportfwdInfo{
		sessionID: sessionID,
		info: ProxyInfo{
			ID: id, Kind: "rportfwd", SessionID: sessionID,
			BindAddr: fullBind, RemoteAddr: fullForward,
			StartedAt: time.Now().UnixMilli(),
		},
	}
	s.mu.Unlock()
	return id, nil
}

func (s *Service) StopRportfwd(id uint64, sessionID string) error {
	if !s.rpc.Connected() {
		return rpc.ErrNotConnected
	}
	sid := sessionID
	s.mu.RLock()
	if sid != "" {
		if entry := s.rpfwd[rportfwdKey{sessionID: sid, id: id}]; entry != nil && entry.sessionID != "" {
			sid = entry.sessionID
		}
	} else {
		for key, entry := range s.rpfwd {
			if key.id == id && entry.sessionID != "" {
				sid = entry.sessionID
				break
			}
		}
	}
	s.mu.RUnlock()
	if sid == "" {
		return fmt.Errorf("rportfwd %d: unknown session", id)
	}
	req := &sliverpb.RportFwdStopListenerReq{
		ID:      uint32(id),
		Request: &commonpb.Request{SessionID: sid},
	}
	resp, err := s.rpc.RPC.StopRportFwdListener(context.Background(), req)
	if err != nil {
		return err
	}
	if resp != nil && resp.Response != nil && resp.Response.Err != "" {
		return fmt.Errorf("implant: %s", resp.Response.Err)
	}
	s.mu.Lock()
	delete(s.rpfwd, rportfwdKey{sessionID: sid, id: id})
	s.mu.Unlock()
	return nil
}

// ListRportfwds returns every reverse port-forward listener the client knows
// about: routed console and GUI entries from the local map, plus a sweep of
// every live session for listeners created by other clients. Server-side
// listeners are per implant, so there is no global list RPC.
func (s *Service) ListRportfwds() ([]ProxyInfo, error) {
	if !s.rpc.Connected() {
		return nil, rpc.ErrNotConnected
	}

	seen := s.snapshotLocalRportfwds()

	sessions, err := s.rpc.RPC.GetSessions(context.Background(), &commonpb.Empty{})
	if err != nil {
		// Best-effort: return what we have locally instead of failing the panel.
		return rportfwdSlice(seen), nil
	}

	for _, sess := range sessions.GetSessions() {
		s.mergeSessionRportfwds(seen, sess.ID)
	}
	return rportfwdSlice(seen), nil
}

func (s *Service) snapshotLocalRportfwds() map[rportfwdKey]ProxyInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()
	seen := make(map[rportfwdKey]ProxyInfo, len(s.rpfwd))
	for key, e := range s.rpfwd {
		seen[key] = e.info
	}
	return seen
}

func (s *Service) mergeSessionRportfwds(seen map[rportfwdKey]ProxyInfo, sessionID string) {
	resp, err := s.rpc.RPC.GetRportFwdListeners(context.Background(), &sliverpb.RportFwdListenersReq{
		Request: &commonpb.Request{SessionID: sessionID},
	})
	if err != nil || resp == nil {
		return
	}
	for _, l := range resp.Listeners {
		id := uint64(l.ID)
		key := rportfwdKey{sessionID: sessionID, id: id}
		if _, ok := seen[key]; ok {
			continue // GUI-started — local metadata (StartedAt) wins
		}
		seen[key] = ProxyInfo{
			Kind:       "rportfwd",
			ID:         id,
			SessionID:  sessionID,
			BindAddr:   l.BindAddress,
			RemoteAddr: l.ForwardAddress,
		}
	}
}

func rportfwdSlice(seen map[rportfwdKey]ProxyInfo) []ProxyInfo {
	out := make([]ProxyInfo, 0, len(seen))
	for _, v := range seen {
		out = append(out, v)
	}
	return out
}
