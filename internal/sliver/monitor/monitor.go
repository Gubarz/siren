package monitor

import (
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"

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

func (s *Service) MonitorStart() (*commonpb.Response, error) {
	return rpcwrap.Call(s.rpc, rpcpb.SliverRPCClient.MonitorStart, &commonpb.Empty{})
}

func (s *Service) MonitorStop() error {
	_, err := rpcwrap.Call(s.rpc, rpcpb.SliverRPCClient.MonitorStop, &commonpb.Empty{})
	return err
}

func (s *Service) ListConfig() (*clientpb.MonitoringProviders, error) {
	return rpcwrap.Call(s.rpc, rpcpb.SliverRPCClient.MonitorListConfig, &commonpb.Empty{})
}

func (s *Service) AddConfig(provider *clientpb.MonitoringProvider) (*commonpb.Response, error) {
	return rpcwrap.Call(s.rpc, rpcpb.SliverRPCClient.MonitorAddConfig, provider)
}

func (s *Service) DelConfig(provider *clientpb.MonitoringProvider) (*commonpb.Response, error) {
	return rpcwrap.Call(s.rpc, rpcpb.SliverRPCClient.MonitorDelConfig, provider)
}
