package tunneling

import (
	"context"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/bishopfox/sliver/client/core"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	sliverpb "github.com/bishopfox/sliver/protobuf/sliverpb"

	"sliver-gui/internal/rpc"
)

var nextID atomic.Uint64

type ProxyInfo struct {
	ID         uint64 `json:"id"`
	Kind       string `json:"kind"` // "socks", "portfwd", "rportfwd"
	SessionID  string `json:"sessionId"`
	BindAddr   string `json:"bindAddr"`
	RemoteAddr string `json:"remoteAddr,omitempty"`
	Username   string `json:"username,omitempty"`
	Password   string `json:"password,omitempty"`
	BytesIn    int64  `json:"bytesIn"`
	BytesOut   int64  `json:"bytesOut"`
	StartedAt  int64  `json:"startedAt"`
}

type socksProxy struct {
	info   ProxyInfo
	driver *socksDriver
}

type rportfwdKey struct {
	sessionID string
	id        uint64
}

type Service struct {
	rpc   *rpc.Client
	mu    sync.RWMutex
	socks map[uint64]*socksProxy
	pfwd  map[uint64]*portfwdProxy
	rpfwd map[rportfwdKey]*rportfwdInfo
}

func New(rpc *rpc.Client) *Service {
	return &Service{
		rpc:   rpc,
		socks: make(map[uint64]*socksProxy),
		pfwd:  make(map[uint64]*portfwdProxy),
		rpfwd: make(map[rportfwdKey]*rportfwdInfo),
	}
}

func (s *Service) StartSocks(sessionID, bindAddr, username, password string) (uint64, error) {
	if !s.rpc.Connected() {
		return 0, rpc.ErrNotConnected
	}
	listener, err := net.Listen("tcp", bindAddr)
	if err != nil {
		return 0, fmt.Errorf("listen %s: %w", bindAddr, err)
	}
	// We register with sliver's core registry so upstream code (the pivots
	// panel, `socks5` console command, etc.) still sees the proxy — but drive
	// it with our own loop instead of core.SocksProxies.Start, which imposes
	// a 10-op/sec rate limit that caps throughput at ~40 KB/s.
	wrapper := core.SocksProxies.Add(&core.TcpProxy{
		Rpc:             s.rpc.RPC,
		Session:         &clientpb.Session{ID: sessionID},
		Listener:        listener,
		BindAddr:        bindAddr,
		Username:        username,
		Password:        password,
		KeepAlivePeriod: 60 * time.Second,
		DialTimeout:     30 * time.Second,
	})
	driver := newSocksDriver(s.rpc.RPC, sessionID, username, password, listener)
	if err := driver.start(); err != nil {
		core.SocksProxies.Remove(wrapper.ID)
		listener.Close()
		return 0, fmt.Errorf("start socks proxy: %w", err)
	}
	s.mu.Lock()
	s.socks[wrapper.ID] = &socksProxy{
		info: ProxyInfo{
			ID: wrapper.ID, Kind: "socks", SessionID: sessionID,
			BindAddr: bindAddr, Username: username, Password: password,
			StartedAt: time.Now().UnixMilli(),
		},
		driver: driver,
	}
	s.mu.Unlock()
	return wrapper.ID, nil
}

func (s *Service) StopSocks(id uint64) error {
	s.mu.Lock()
	entry := s.socks[id]
	delete(s.socks, id)
	s.mu.Unlock()
	if entry != nil && entry.driver != nil {
		entry.driver.stop()
	}
	if !core.SocksProxies.Remove(id) && entry == nil {
		return fmt.Errorf("socks proxy %d not found", id)
	}
	return nil
}

// Close tears down every proxy this service is tracking. Meant to be called
// during app shutdown so live socks/portfwd/rportfwd tunnels don't get left
// mid-transfer when the gRPC connection drops — a state sliver's server has
// historically panicked on.
func (s *Service) Close() {
	s.mu.Lock()
	socks := make(map[uint64]*socksDriver, len(s.socks))
	for id, sp := range s.socks {
		socks[id] = sp.driver
	}
	pfwds := make([]*portfwdProxy, 0, len(s.pfwd))
	for _, pp := range s.pfwd {
		pfwds = append(pfwds, pp)
	}
	rpfwds := make([]rportfwdKey, 0, len(s.rpfwd))
	for key := range s.rpfwd {
		rpfwds = append(rpfwds, key)
	}
	s.socks = map[uint64]*socksProxy{}
	s.pfwd = map[uint64]*portfwdProxy{}
	s.rpfwd = map[rportfwdKey]*rportfwdInfo{}
	s.mu.Unlock()

	for id, driver := range socks {
		if driver != nil {
			driver.stop()
		}
		core.SocksProxies.Remove(id)
	}
	for _, pp := range pfwds {
		close(pp.quit)
		pp.listener.Close()
	}
	if s.rpc.Connected() {
		for _, key := range rpfwds {
			req := &sliverpb.RportFwdStopListenerReq{
				ID:      uint32(key.id),
				Request: &commonpb.Request{SessionID: key.sessionID},
			}
			_, _ = s.rpc.RPC.StopRportFwdListener(context.Background(), req)
		}
	}
}

// List returns every active SOCKS + port-forward known to the parent client.
// Routed console commands and GUI actions live in the local maps; the Sliver
// registries cover proxies created by other in-process integrations.
func (s *Service) List() []ProxyInfo {
	s.mu.RLock()
	localSocks := make(map[uint64]ProxyInfo, len(s.socks))
	for id, p := range s.socks {
		localSocks[id] = p.info
	}
	localPortfwds := make(map[uint64]ProxyInfo, len(s.pfwd))
	for id, p := range s.pfwd {
		localPortfwds[id] = p.info
	}
	s.mu.RUnlock()

	out := make([]ProxyInfo, 0, len(localSocks)+len(localPortfwds))
	for _, meta := range core.SocksProxies.List() {
		info := ProxyInfo{
			ID: meta.ID, Kind: "socks",
			SessionID: meta.SessionID,
			BindAddr:  meta.BindAddr,
			Username:  meta.Username,
			Password:  meta.Password,
		}
		if prev, ok := localSocks[meta.ID]; ok {
			info.StartedAt = prev.StartedAt
		}
		out = append(out, info)
	}
	for _, meta := range core.Portfwds.List() {
		info := ProxyInfo{
			ID: uint64(meta.ID), Kind: "portfwd",
			SessionID:  meta.SessionID,
			BindAddr:   meta.BindAddr,
			RemoteAddr: meta.RemoteAddr,
		}
		if prev, ok := localPortfwds[uint64(meta.ID)]; ok {
			info.StartedAt = prev.StartedAt
		}
		out = append(out, info)
	}
	// GUI-created portfwds live in s.pfwd (we hand-rolled portfwd instead of
	// using core.Portfwds), so append those too.
	for _, info := range localPortfwds {
		out = append(out, info)
	}
	return out
}
