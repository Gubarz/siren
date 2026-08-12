package monitor

import (
	"context"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"

	"siren/internal/sliver/rpc"
)

type Service struct {
	rpc *rpc.Client
}

func New(rpc *rpc.Client) *Service {
	return &Service{rpc: rpc}
}

func (s *Service) Close() {}

func (s *Service) MonitorStart() (*commonpb.Response, error) {
	if !s.rpc.Connected() {
		return nil, rpc.ErrNotConnected
	}
	return s.rpc.RPC.MonitorStart(context.Background(), &commonpb.Empty{})
}

func (s *Service) MonitorStop() error {
	if !s.rpc.Connected() {
		return rpc.ErrNotConnected
	}
	_, err := s.rpc.RPC.MonitorStop(context.Background(), &commonpb.Empty{})
	return err
}

func (s *Service) ListConfig() (*clientpb.MonitoringProviders, error) {
	if !s.rpc.Connected() {
		return nil, rpc.ErrNotConnected
	}
	return s.rpc.RPC.MonitorListConfig(context.Background(), &commonpb.Empty{})
}

func (s *Service) AddConfig(provider *clientpb.MonitoringProvider) (*commonpb.Response, error) {
	if !s.rpc.Connected() {
		return nil, rpc.ErrNotConnected
	}
	return s.rpc.RPC.MonitorAddConfig(context.Background(), provider)
}

func (s *Service) DelConfig(provider *clientpb.MonitoringProvider) (*commonpb.Response, error) {
	if !s.rpc.Connected() {
		return nil, rpc.ErrNotConnected
	}
	return s.rpc.RPC.MonitorDelConfig(context.Background(), provider)
}
