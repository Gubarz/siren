package services

import (
	"context"
	"time"

	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"

	"sliver-gui/internal/sliver/rpc"
)

type Service struct {
	rpc *rpc.Client
}

const requestTimeout = 5 * time.Minute

func New(rpc *rpc.Client) *Service {
	return &Service{rpc: rpc}
}

func (s *Service) GetServices(sessionID string) (*sliverpb.Services, error) {
	if !s.rpc.Connected() {
		return nil, rpc.ErrNotConnected
	}

	req, err := s.rpc.TargetRequest(sessionID, requestTimeout)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()

	resp, err := s.rpc.RPC.Services(ctx, &sliverpb.ServicesReq{
		Request: req,
	})
	if err != nil {
		return nil, err
	}
	if err := s.rpc.AwaitAsyncResponse(ctx, resp, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

func (s *Service) StartService(sessionID, name string) error {
	return s.runServiceAction(sessionID, func(ctx context.Context, req *commonpb.Request) (*sliverpb.ServiceInfo, error) {
		return s.rpc.RPC.StartServiceByName(ctx, &sliverpb.StartServiceByNameReq{
			ServiceInfo: &sliverpb.ServiceInfoReq{ServiceName: name},
			Request:     req,
		})
	})
}

func (s *Service) StopService(sessionID, name string) error {
	return s.runServiceAction(sessionID, func(ctx context.Context, req *commonpb.Request) (*sliverpb.ServiceInfo, error) {
		return s.rpc.RPC.StopService(ctx, &sliverpb.StopServiceReq{
			ServiceInfo: &sliverpb.ServiceInfoReq{ServiceName: name},
			Request:     req,
		})
	})
}

func (s *Service) RemoveService(sessionID, name string) error {
	return s.runServiceAction(sessionID, func(ctx context.Context, req *commonpb.Request) (*sliverpb.ServiceInfo, error) {
		return s.rpc.RPC.RemoveService(ctx, &sliverpb.RemoveServiceReq{
			ServiceInfo: &sliverpb.ServiceInfoReq{ServiceName: name},
			Request:     req,
		})
	})
}

func (s *Service) runServiceAction(
	sessionID string,
	action func(context.Context, *commonpb.Request) (*sliverpb.ServiceInfo, error),
) error {
	if !s.rpc.Connected() {
		return rpc.ErrNotConnected
	}
	req, err := s.rpc.TargetRequest(sessionID, requestTimeout)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()
	resp, err := action(ctx, req)
	if err != nil {
		return err
	}
	return s.rpc.AwaitAsyncResponse(ctx, resp, resp)
}
