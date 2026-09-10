package builders

import (
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"

	"siren/internal/sliver/rpc"
	"siren/internal/sliver/rpcwrap"
)

type Service struct {
	rpc.Emitter
	rpc *rpc.Client
}

func New(rpcClient *rpc.Client) *Service {
	return &Service{
		rpc:     rpcClient,
		Emitter: rpc.NewEmitter(rpcClient),
	}
}

func (s *Service) Close() {}

func (s *Service) Builders() (*clientpb.Builders, error) {
	return rpcwrap.Call(s.rpc, rpcpb.SliverRPCClient.Builders, &commonpb.Empty{})
}

func (s *Service) GenerateExternal(req *clientpb.ExternalGenerateReq) (*clientpb.ExternalImplantConfig, error) {
	return rpcwrap.Call(s.rpc, rpcpb.SliverRPCClient.GenerateExternal, req)
}

func (s *Service) GetExternalBuildConfig(build *clientpb.ImplantBuild) (*clientpb.ExternalImplantConfig, error) {
	return rpcwrap.Call(s.rpc, rpcpb.SliverRPCClient.GenerateExternalGetBuildConfig, build)
}

func (s *Service) SaveExternalBuild(binary *clientpb.ExternalImplantBinary) error {
	if _, err := rpcwrap.Call(s.rpc, rpcpb.SliverRPCClient.GenerateExternalSaveBuild, binary); err != nil {
		return err
	}
	s.Publish("gui.payload-built", map[string]any{"name": binary.Name, "builder": "external"})
	return nil
}

func (s *Service) Trigger(ev *clientpb.Event) error {
	_, err := rpcwrap.Call(s.rpc, rpcpb.SliverRPCClient.BuilderTrigger, ev)
	return err
}
