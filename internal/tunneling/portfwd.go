package tunneling

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"strconv"
	"time"

	"github.com/bishopfox/sliver/client/core"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	sliverpb "github.com/bishopfox/sliver/protobuf/sliverpb"

	"sliver-gui/internal/rpc"
)

type portfwdProxy struct {
	info     ProxyInfo
	rpc      *rpc.Client
	listener net.Listener
	quit     chan struct{}
}

func (s *Service) StartPortfwd(sessionID, bindAddr, remoteAddr string) (uint64, error) {
	if !s.rpc.Connected() {
		return 0, rpc.ErrNotConnected
	}

	listener, err := net.Listen("tcp", bindAddr)
	if err != nil {
		return 0, fmt.Errorf("listen %s: %w", bindAddr, err)
	}

	id := nextID.Add(1)
	pp := &portfwdProxy{
		info: ProxyInfo{
			ID: id, Kind: "portfwd", SessionID: sessionID,
			BindAddr: bindAddr, RemoteAddr: remoteAddr,
			StartedAt: time.Now().UnixMilli(),
		},
		rpc:      s.rpc,
		listener: listener,
		quit:     make(chan struct{}),
	}

	s.mu.Lock()
	s.pfwd[id] = pp
	s.mu.Unlock()

	go pp.acceptLoop(sessionID, remoteAddr)
	return id, nil
}

func (s *Service) StopPortfwd(id uint64) error {
	s.mu.Lock()
	pp, ok := s.pfwd[id]
	if ok {
		delete(s.pfwd, id)
		s.mu.Unlock()
		close(pp.quit)
		pp.listener.Close()
		return nil
	}
	s.mu.Unlock()
	// Not one of ours — try Sliver's core registry (portfwds started via the
	// console/right-click).
	if core.Portfwds.Remove(int(id)) {
		return nil
	}
	return fmt.Errorf("portfwd %d not found", id)
}

func (pp *portfwdProxy) acceptLoop(sessionID, remoteAddr string) {
	defer pp.listener.Close()
	for {
		conn, err := pp.listener.Accept()
		if err != nil {
			return
		}
		select {
		case <-pp.quit:
			conn.Close()
			return
		default:
		}
		go pp.handleConn(conn, sessionID, remoteAddr)
	}
}

func (pp *portfwdProxy) handleConn(conn net.Conn, sessionID, remoteAddr string) {
	defer conn.Close()

	host, portStr, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		log.Printf("portfwd: bad remote addr %s: %v", remoteAddr, err)
		return
	}
	portNum, err := strconv.Atoi(portStr)
	if err != nil || portNum < 1 || portNum > 65535 {
		log.Printf("portfwd: bad port %s", portStr)
		return
	}

	rpcTunnel, err := pp.rpc.RPC.CreateTunnel(context.Background(), &sliverpb.Tunnel{
		SessionID: sessionID,
	})
	if err != nil {
		return
	}
	tunnel := core.GetTunnels().Start(rpcTunnel.TunnelID, rpcTunnel.SessionID)
	defer core.GetTunnels().Close(tunnel.ID)

	_, err = pp.rpc.RPC.Portfwd(context.Background(), &sliverpb.PortfwdReq{
		Request:  &commonpb.Request{SessionID: sessionID},
		Host:     host,
		Port:     uint32(portNum),
		Protocol: sliverpb.PortFwdProtoTCP,
		TunnelID: tunnel.ID,
	})
	if err != nil {
		return
	}

	errs := make(chan error, 2)
	go func() { _, err := io.Copy(tunnel, conn); errs <- err }()
	go func() { _, err := io.Copy(conn, tunnel); errs <- err }()
	<-errs
}
