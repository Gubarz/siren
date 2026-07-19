package builders

import (
	"context"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"

	"sliver-gui/internal/rpc"
)

type Service struct {
	rpc *rpc.Client
}

func New(rpc *rpc.Client) *Service {
	return &Service{rpc: rpc}
}

func (s *Service) Close() {}

func (s *Service) Builders() (*clientpb.Builders, error) {
	if !s.rpc.Connected() {
		return nil, rpc.ErrNotConnected
	}
	return s.rpc.RPC.Builders(context.Background(), &commonpb.Empty{})
}

func (s *Service) GenerateExternal(req *clientpb.ExternalGenerateReq) (*clientpb.ExternalImplantConfig, error) {
	if !s.rpc.Connected() {
		return nil, rpc.ErrNotConnected
	}
	return s.rpc.RPC.GenerateExternal(context.Background(), req)
}

func (s *Service) GetExternalBuildConfig(build *clientpb.ImplantBuild) (*clientpb.ExternalImplantConfig, error) {
	if !s.rpc.Connected() {
		return nil, rpc.ErrNotConnected
	}
	return s.rpc.RPC.GenerateExternalGetBuildConfig(context.Background(), build)
}

func (s *Service) SaveExternalBuild(binary *clientpb.ExternalImplantBinary) error {
	if !s.rpc.Connected() {
		return rpc.ErrNotConnected
	}
	_, err := s.rpc.RPC.GenerateExternalSaveBuild(context.Background(), binary)
	return err
}

func (s *Service) Trigger(ev *clientpb.Event) error {
	if !s.rpc.Connected() {
		return rpc.ErrNotConnected
	}
	_, err := s.rpc.RPC.BuilderTrigger(context.Background(), ev)
	return err
}
