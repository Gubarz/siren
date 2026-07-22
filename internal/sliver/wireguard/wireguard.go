package wireguard

import (
	"context"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"

	"sliver-gui/internal/sliver/rpc"
)

type Service struct {
	rpc *rpc.Client
}

func New(rpc *rpc.Client) *Service {
	return &Service{rpc: rpc}
}

func (s *Service) Close() {}

func (s *Service) StartListener(host string, port, nport, keyPort uint32, tunIP string) (*clientpb.ListenerJob, error) {
	if !s.rpc.Connected() {
		return nil, rpc.ErrNotConnected
	}
	return s.rpc.RPC.StartWGListener(context.Background(), &clientpb.WGListenerReq{
		Host:    host,
		Port:    port,
		NPort:   nport,
		KeyPort: keyPort,
		TunIP:   tunIP,
	})
}

func (s *Service) GenerateClientConfig() (*clientpb.WGClientConfig, error) {
	if !s.rpc.Connected() {
		return nil, rpc.ErrNotConnected
	}
	return s.rpc.RPC.GenerateWGClientConfig(context.Background(), &commonpb.Empty{})
}

func (s *Service) GenerateUniqueIP() (*clientpb.UniqueWGIP, error) {
	if !s.rpc.Connected() {
		return nil, rpc.ErrNotConnected
	}
	return s.rpc.RPC.GenerateUniqueIP(context.Background(), &commonpb.Empty{})
}

func (s *Service) StartSocks(sessionID string, port int32) (*sliverpb.WGSocks, error) {
	if !s.rpc.Connected() {
		return nil, rpc.ErrNotConnected
	}
	return s.rpc.RPC.WGStartSocks(context.Background(), &sliverpb.WGSocksStartReq{
		Port:    port,
		Request: &commonpb.Request{SessionID: sessionID},
	})
}

func (s *Service) StopSocks(sessionID string, id int32) (*sliverpb.WGSocks, error) {
	if !s.rpc.Connected() {
		return nil, rpc.ErrNotConnected
	}
	return s.rpc.RPC.WGStopSocks(context.Background(), &sliverpb.WGSocksStopReq{
		ID:      id,
		Request: &commonpb.Request{SessionID: sessionID},
	})
}

func (s *Service) ListSocksServers(sessionID string) (*sliverpb.WGSocksServers, error) {
	if !s.rpc.Connected() {
		return nil, rpc.ErrNotConnected
	}
	return s.rpc.RPC.WGListSocksServers(context.Background(), &sliverpb.WGSocksServersReq{
		Request: &commonpb.Request{SessionID: sessionID},
	})
}

func (s *Service) StartPortForward(sessionID string, localPort int32, remoteAddr string) (*sliverpb.WGPortForward, error) {
	if !s.rpc.Connected() {
		return nil, rpc.ErrNotConnected
	}
	return s.rpc.RPC.WGStartPortForward(context.Background(), &sliverpb.WGPortForwardStartReq{
		LocalPort:     localPort,
		RemoteAddress: remoteAddr,
		Request:       &commonpb.Request{SessionID: sessionID},
	})
}

func (s *Service) StopPortForward(sessionID string, id int32) (*sliverpb.WGPortForward, error) {
	if !s.rpc.Connected() {
		return nil, rpc.ErrNotConnected
	}
	return s.rpc.RPC.WGStopPortForward(context.Background(), &sliverpb.WGPortForwardStopReq{
		ID:      id,
		Request: &commonpb.Request{SessionID: sessionID},
	})
}

func (s *Service) ListForwarders(sessionID string) (*sliverpb.WGTCPForwarders, error) {
	if !s.rpc.Connected() {
		return nil, rpc.ErrNotConnected
	}
	return s.rpc.RPC.WGListForwarders(context.Background(), &sliverpb.WGTCPForwardersReq{
		Request: &commonpb.Request{SessionID: sessionID},
	})
}
