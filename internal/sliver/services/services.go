package services

import (
	"context"
	"fmt"
	"time"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"google.golang.org/protobuf/proto"

	"sliver-gui/internal/sliver/rpc"
)

type Service struct {
	rpc *rpc.Client
}

func New(rpc *rpc.Client) *Service {
	return &Service{rpc: rpc}
}

func (s *Service) GetServices(sessionID string) (*sliverpb.Services, error) {
	if !s.rpc.Connected() {
		return nil, rpc.ErrNotConnected
	}

	req := &commonpb.Request{
		SessionID: sessionID,
		Timeout:   int64((120 * time.Second) / time.Second),
	}
	if sess := s.rpc.LookupSession(sessionID); sess == nil {
		if beacon := s.rpc.LookupBeacon(sessionID); beacon != nil {
			req.BeaconID = sessionID
			req.Async = true
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	resp, err := s.rpc.RPC.Services(ctx, &sliverpb.ServicesReq{
		Request: req,
	})
	if err != nil {
		return nil, err
	}
	if resp.Response != nil && resp.Response.Err != "" {
		return nil, fmt.Errorf("implant: %s", resp.Response.Err)
	}

	if resp.Response != nil && resp.Response.Async && resp.Response.TaskID != "" {
		return s.awaitTask(ctx, resp)
	}

	return resp, nil
}

func (s *Service) awaitTask(ctx context.Context, services *sliverpb.Services) (*sliverpb.Services, error) {
	taskID := services.Response.TaskID
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		task, err := s.rpc.RPC.GetBeaconTaskContent(ctx, &clientpb.BeaconTask{ID: taskID})
		if err != nil {
			return nil, err
		}
		switch task.State {
		case "completed":
			if len(task.Response) > 0 {
				if err := proto.Unmarshal(task.Response, services); err != nil {
					return nil, fmt.Errorf("decode: %w", err)
				}
			}
			return services, nil
		case "failed", "canceled":
			return nil, fmt.Errorf("task %s", task.State)
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
		}
	}
}

func (s *Service) StartService(sessionID, name string) error {
	return s.runServiceAction(sessionID, func(req *commonpb.Request) (*sliverpb.ServiceInfo, error) {
		return s.rpc.RPC.StartServiceByName(context.Background(), &sliverpb.StartServiceByNameReq{
			ServiceInfo: &sliverpb.ServiceInfoReq{ServiceName: name},
			Request:     req,
		})
	})
}

func (s *Service) StopService(sessionID, name string) error {
	return s.runServiceAction(sessionID, func(req *commonpb.Request) (*sliverpb.ServiceInfo, error) {
		return s.rpc.RPC.StopService(context.Background(), &sliverpb.StopServiceReq{
			ServiceInfo: &sliverpb.ServiceInfoReq{ServiceName: name},
			Request:     req,
		})
	})
}

func (s *Service) RemoveService(sessionID, name string) error {
	return s.runServiceAction(sessionID, func(req *commonpb.Request) (*sliverpb.ServiceInfo, error) {
		return s.rpc.RPC.RemoveService(context.Background(), &sliverpb.RemoveServiceReq{
			ServiceInfo: &sliverpb.ServiceInfoReq{ServiceName: name},
			Request:     req,
		})
	})
}

func (s *Service) runServiceAction(sessionID string, action func(*commonpb.Request) (*sliverpb.ServiceInfo, error)) error {
	if !s.rpc.Connected() {
		return rpc.ErrNotConnected
	}
	req := &commonpb.Request{
		SessionID: sessionID,
		Timeout:   int64((60 * time.Second) / time.Second),
	}
	if s.rpc.LookupSession(sessionID) == nil {
		if beacon := s.rpc.LookupBeacon(sessionID); beacon != nil {
			req.BeaconID = sessionID
			req.Async = true
		}
	}
	resp, err := action(req)
	if err != nil {
		return err
	}
	if resp.Response != nil && resp.Response.Err != "" {
		return fmt.Errorf("implant: %s", resp.Response.Err)
	}
	return nil
}
