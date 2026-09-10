package wireguard

import (
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"

	"siren/internal/sliver/rpc"
	"siren/internal/sliver/rpcwrap"
)

type Service struct {
	rpc *rpc.Client
}

func New(rpc *rpc.Client) *Service {
	return &Service{rpc: rpc}
}

func (s *Service) Close() {}

func (s *Service) StartListener(host string, port, nport, keyPort uint32, tunIP string) (*clientpb.ListenerJob, error) {
	return rpcwrap.Call(s.rpc, rpcpb.SliverRPCClient.StartWGListener, &clientpb.WGListenerReq{
		Host:    host,
		Port:    port,
		NPort:   nport,
		KeyPort: keyPort,
		TunIP:   tunIP,
	})
}

func (s *Service) GenerateClientConfig() (*clientpb.WGClientConfig, error) {
	return rpcwrap.Call(s.rpc, rpcpb.SliverRPCClient.GenerateWGClientConfig, &commonpb.Empty{})
}

func (s *Service) GenerateUniqueIP() (*clientpb.UniqueWGIP, error) {
	return rpcwrap.Call(s.rpc, rpcpb.SliverRPCClient.GenerateUniqueIP, &commonpb.Empty{})
}

func (s *Service) StartSocks(sessionID string, port int32) (*sliverpb.WGSocks, error) {
	return rpcwrap.CallTarget(s.rpc, rpcpb.SliverRPCClient.WGStartSocks, &sliverpb.WGSocksStartReq{Port: port}, sessionID)
}

func (s *Service) StopSocks(sessionID string, id int32) (*sliverpb.WGSocks, error) {
	return rpcwrap.CallTarget(s.rpc, rpcpb.SliverRPCClient.WGStopSocks, &sliverpb.WGSocksStopReq{ID: id}, sessionID)
}

func (s *Service) ListSocksServers(sessionID string) (*sliverpb.WGSocksServers, error) {
	return rpcwrap.CallTarget(s.rpc, rpcpb.SliverRPCClient.WGListSocksServers, &sliverpb.WGSocksServersReq{}, sessionID)
}

func (s *Service) StartPortForward(sessionID string, localPort int32, remoteAddr string) (*sliverpb.WGPortForward, error) {
	return rpcwrap.CallTarget(s.rpc, rpcpb.SliverRPCClient.WGStartPortForward, &sliverpb.WGPortForwardStartReq{
		LocalPort:     localPort,
		RemoteAddress: remoteAddr,
	}, sessionID)
}

func (s *Service) StopPortForward(sessionID string, id int32) (*sliverpb.WGPortForward, error) {
	return rpcwrap.CallTarget(s.rpc, rpcpb.SliverRPCClient.WGStopPortForward, &sliverpb.WGPortForwardStopReq{ID: id}, sessionID)
}

func (s *Service) ListForwarders(sessionID string) (*sliverpb.WGTCPForwarders, error) {
	return rpcwrap.CallTarget(s.rpc, rpcpb.SliverRPCClient.WGListForwarders, &sliverpb.WGTCPForwardersReq{}, sessionID)
}
