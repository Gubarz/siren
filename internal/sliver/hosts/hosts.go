package hosts

import (
	"context"
	"fmt"
	"strings"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"

	"siren/internal/sliver/rpc"
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

func (s *Service) client() (hostRPC, error) {
	if !s.rpc.Connected() {
		return nil, rpc.ErrNotConnected
	}
	return s.rpc, nil
}

func (s *Service) List() (*clientpb.AllHosts, error) {
	c, err := s.client()
	if err != nil {
		return nil, err
	}
	return c.Hosts(context.Background(), &commonpb.Empty{})
}

func (s *Service) Get(hostUUID string) (*clientpb.Host, error) {
	c, err := s.client()
	if err != nil {
		return nil, err
	}
	hostUUID = strings.TrimSpace(hostUUID)
	if hostUUID == "" {
		return nil, fmt.Errorf("host UUID is required")
	}
	return c.Host(context.Background(), &clientpb.Host{HostUUID: hostUUID})
}

func (s *Service) Remove(hostUUID string) error {
	return removeTarget(s, hostUUID, "host UUID", hostRPC.HostRm, &clientpb.Host{HostUUID: strings.TrimSpace(hostUUID)})
}

func (s *Service) RemoveIOC(iocID string) error {
	return removeTarget(s, iocID, "IOC ID", hostRPC.HostIOCRm, &clientpb.IOC{ID: strings.TrimSpace(iocID)})
}

func removeTarget[Req any](
	s *Service,
	id, label string,
	method func(hostRPC, context.Context, Req) (*commonpb.Empty, error),
	req Req,
) error {
	c, err := s.client()
	if err != nil {
		return err
	}
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("%s is required", label)
	}
	_, err = method(c, context.Background(), req)
	return err
}

type liveHostRPC struct {
	client *rpc.Client
}

func (r liveHostRPC) Connected() bool {
	return r.client != nil && r.client.Connected()
}

func (r liveHostRPC) Hosts(ctx context.Context, req *commonpb.Empty) (*clientpb.AllHosts, error) {
	return r.client.RPC().Hosts(ctx, req)
}

func (r liveHostRPC) Host(ctx context.Context, req *clientpb.Host) (*clientpb.Host, error) {
	return r.client.RPC().Host(ctx, req)
}

func (r liveHostRPC) HostRm(ctx context.Context, req *clientpb.Host) (*commonpb.Empty, error) {
	return r.client.RPC().HostRm(ctx, req)
}

func (r liveHostRPC) HostIOCRm(ctx context.Context, req *clientpb.IOC) (*commonpb.Empty, error) {
	return r.client.RPC().HostIOCRm(ctx, req)
}
