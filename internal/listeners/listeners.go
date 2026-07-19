package listeners

import (
	"context"
	"fmt"
	"strings"

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

func (s *Service) GetJobs() (*clientpb.Jobs, error) {
	if !s.rpc.Connected() {
		return nil, rpc.ErrNotConnected
	}
	return s.rpc.RPC.GetJobs(context.Background(), &commonpb.Empty{})
}

func (s *Service) KillJob(id uint32) error {
	if !s.rpc.Connected() {
		return rpc.ErrNotConnected
	}
	_, err := s.rpc.RPC.KillJob(context.Background(), &clientpb.KillJobReq{ID: id})
	return err
}

func (s *Service) StartListener(protocol, host string, port uint32, domains string) error {
	if !s.rpc.Connected() {
		return rpc.ErrNotConnected
	}
	ctx := context.Background()

	switch strings.ToLower(protocol) {
	case "mtls":
		_, err := s.rpc.RPC.StartMTLSListener(ctx, &clientpb.MTLSListenerReq{Host: host, Port: port})
		return err
	case "http":
		_, err := s.rpc.RPC.StartHTTPListener(ctx, &clientpb.HTTPListenerReq{Host: host, Port: port, Secure: false})
		return err
	case "https":
		_, err := s.rpc.RPC.StartHTTPSListener(ctx, &clientpb.HTTPListenerReq{Host: host, Port: port, Secure: true})
		return err
	case "dns":
		var doms []string
		for _, d := range strings.Split(domains, ",") {
			if d = strings.TrimSpace(d); d != "" {
				doms = append(doms, d)
			}
		}
		_, err := s.rpc.RPC.StartDNSListener(ctx, &clientpb.DNSListenerReq{Domains: doms, Host: host, Port: port})
		return err
	default:
		return fmt.Errorf("unknown listener protocol: %s", protocol)
	}
}
