package services

import (
	"time"

	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"

	"siren/internal/sliver/rpc"
	"siren/internal/sliver/rpcwrap"
)

type Service struct {
	rpc *rpc.Client
}

const requestTimeout = 5 * time.Minute

func New(rpc *rpc.Client) *Service {
	return &Service{rpc: rpc}
}

func (s *Service) GetServices(sessionID string) (*sliverpb.Services, error) {
	return rpcwrap.TargetCall(s.rpc, sessionID, requestTimeout, rpcpb.SliverRPCClient.Services, &sliverpb.ServicesReq{})
}

func (s *Service) StartService(sessionID, name string) error {
	return rpcwrap.TargetAction(s.rpc, sessionID, requestTimeout, rpcpb.SliverRPCClient.StartServiceByName, &sliverpb.StartServiceByNameReq{
		ServiceInfo: &sliverpb.ServiceInfoReq{ServiceName: name},
	})
}

func (s *Service) StopService(sessionID, name string) error {
	return rpcwrap.TargetAction(s.rpc, sessionID, requestTimeout, rpcpb.SliverRPCClient.StopService, &sliverpb.StopServiceReq{
		ServiceInfo: &sliverpb.ServiceInfoReq{ServiceName: name},
	})
}

func (s *Service) RemoveService(sessionID, name string) error {
	return rpcwrap.TargetAction(s.rpc, sessionID, requestTimeout, rpcpb.SliverRPCClient.RemoveService, &sliverpb.RemoveServiceReq{
		ServiceInfo: &sliverpb.ServiceInfoReq{ServiceName: name},
	})
}
