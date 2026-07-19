package hosts

import (
	"context"
	"fmt"
	"strings"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"

	"sliver-gui/internal/rpc"
)

type hostRPC interface {
	Connected() bool
	Hosts(context.Context, *commonpb.Empty) (*clientpb.AllHosts, error)
	Host(context.Context, *clientpb.Host) (*clientpb.Host, error)
	HostRm(context.Context, *clientpb.Host) (*commonpb.Empty, error)
	HostIOCRm(context.Context, *clientpb.IOC) (*commonpb.Empty, error)
}

type Service struct {
	rpc hostRPC
}

func New(rpc *rpc.Client) *Service {
	return &Service{rpc: liveHostRPC{client: rpc}}
}

func (s *Service) Close() {}

func (s *Service) List() (*clientpb.AllHosts, error) {
	if !s.rpc.Connected() {
		return nil, rpc.ErrNotConnected
	}
	return s.rpc.Hosts(context.Background(), &commonpb.Empty{})
}

func (s *Service) Get(hostUUID string) (*clientpb.Host, error) {
	if !s.rpc.Connected() {
		return nil, rpc.ErrNotConnected
	}
	hostUUID = strings.TrimSpace(hostUUID)
	if hostUUID == "" {
		return nil, fmt.Errorf("host UUID is required")
	}
	return s.rpc.Host(context.Background(), &clientpb.Host{HostUUID: hostUUID})
}

func (s *Service) Remove(hostUUID string) error {
	if !s.rpc.Connected() {
		return rpc.ErrNotConnected
	}
	hostUUID = strings.TrimSpace(hostUUID)
	if hostUUID == "" {
		return fmt.Errorf("host UUID is required")
	}
	_, err := s.rpc.HostRm(context.Background(), &clientpb.Host{HostUUID: hostUUID})
	return err
}

func (s *Service) RemoveIOC(iocID string) error {
	if !s.rpc.Connected() {
		return rpc.ErrNotConnected
	}
	iocID = strings.TrimSpace(iocID)
	if iocID == "" {
		return fmt.Errorf("IOC ID is required")
	}
	_, err := s.rpc.HostIOCRm(context.Background(), &clientpb.IOC{ID: iocID})
	return err
}

type liveHostRPC struct {
	client *rpc.Client
}

func (r liveHostRPC) Connected() bool {
	return r.client != nil && r.client.Connected()
}

func (r liveHostRPC) Hosts(ctx context.Context, req *commonpb.Empty) (*clientpb.AllHosts, error) {
	return r.client.RPC.Hosts(ctx, req)
}

func (r liveHostRPC) Host(ctx context.Context, req *clientpb.Host) (*clientpb.Host, error) {
	return r.client.RPC.Host(ctx, req)
}

func (r liveHostRPC) HostRm(ctx context.Context, req *clientpb.Host) (*commonpb.Empty, error) {
	return r.client.RPC.HostRm(ctx, req)
}

func (r liveHostRPC) HostIOCRm(ctx context.Context, req *clientpb.IOC) (*commonpb.Empty, error) {
	return r.client.RPC.HostIOCRm(ctx, req)
}
