package tunneling

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/bishopfox/sliver/protobuf/commonpb"
	sliverpb "github.com/bishopfox/sliver/protobuf/sliverpb"

	"sliver-gui/internal/rpc"
)

// rportfwdInfo tracks a reverse port forward we started, so we can list/stop
// it with the correct SessionID (the RPC won't route without one).
type rportfwdInfo struct {
	info      ProxyInfo
	sessionID string
}

func (s *Service) StartRportfwd(sessionID, bindAddr string, bindPort int, forwardAddr string, forwardPort int) (uint64, error) {
	if !s.rpc.Connected() {
		return 0, rpc.ErrNotConnected
	}

	// The implant handler ignores BindPort/ForwardPort and parses the port out
	// of BindAddress/ForwardAddress, so we must send them as full "host:port".
	fullBind := net.JoinHostPort(bindAddr, strconv.Itoa(bindPort))
	fullForward := net.JoinHostPort(forwardAddr, strconv.Itoa(forwardPort))

	resp, err := s.rpc.RPC.StartRportFwdListener(context.Background(), &sliverpb.RportFwdStartListenerReq{
		Request:        &commonpb.Request{SessionID: sessionID},
		BindAddress:    fullBind,
		ForwardAddress: fullForward,
	})
	if err != nil {
		return 0, err
	}
	if resp.Response != nil && resp.Response.Err != "" {
		return 0, fmt.Errorf("implant: %s", resp.Response.Err)
	}
	id := uint64(resp.ID)
	s.mu.Lock()
	s.rpfwd[id] = &rportfwdInfo{
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
	s.mu.RLock()
	entry := s.rpfwd[id]
	s.mu.RUnlock()

	// Local map only has GUI-started rportfwds; console-created ones live only
	// on the implant, so the UI passes the SessionID from the listing row.
	sid := sessionID
	if entry != nil && entry.sessionID != "" {
		sid = entry.sessionID
	}
	if sid == "" {
		return fmt.Errorf("rportfwd %d: unknown session", id)
	}
	req := &sliverpb.RportFwdStopListenerReq{
		ID:      uint32(id),
		Request: &commonpb.Request{SessionID: sid},
	}
	_, err := s.rpc.RPC.StopRportFwdListener(context.Background(), req)
	if err != nil {
		return err
	}
	s.mu.Lock()
	delete(s.rpfwd, id)
	s.mu.Unlock()
	return nil
}

// ListRportfwds returns every reverse port-forward listener the client knows
// about — anything we started via the GUI (from the local map) plus a sweep
// of every live session's listeners so RMC-created / console-created
// rportfwds show up too. Server-side listeners are per-implant, so we ask
// each session individually — there's no global list RPC.
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

func (s *Service) snapshotLocalRportfwds() map[uint64]ProxyInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()
	seen := make(map[uint64]ProxyInfo, len(s.rpfwd))
	for id, e := range s.rpfwd {
		seen[id] = e.info
	}
	return seen
}

func (s *Service) mergeSessionRportfwds(seen map[uint64]ProxyInfo, sessionID string) {
	resp, err := s.rpc.RPC.GetRportFwdListeners(context.Background(), &sliverpb.RportFwdListenersReq{
		Request: &commonpb.Request{SessionID: sessionID},
	})
	if err != nil || resp == nil {
		return
	}
	for _, l := range resp.Listeners {
		id := uint64(l.ID)
		if _, ok := seen[id]; ok {
			continue // GUI-started — local metadata (StartedAt) wins
		}
		seen[id] = ProxyInfo{
			Kind:       "rportfwd",
			ID:         id,
			SessionID:  sessionID,
			BindAddr:   l.BindAddress,
			RemoteAddr: l.ForwardAddress,
		}
	}
}

func rportfwdSlice(seen map[uint64]ProxyInfo) []ProxyInfo {
	out := make([]ProxyInfo, 0, len(seen))
	for _, v := range seen {
		out = append(out, v)
	}
	return out
}
