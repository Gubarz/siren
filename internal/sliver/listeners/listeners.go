package listeners

import (
	"context"
	"fmt"
	"strings"

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

func (s *Service) GetJobs() (*clientpb.Jobs, error) {
	return rpcwrap.Call(s.rpc, rpcpb.SliverRPCClient.GetJobs, &commonpb.Empty{})
}

func (s *Service) KillJob(id uint32) error {
	_, err := rpcwrap.Call(s.rpc, rpcpb.SliverRPCClient.KillJob, &clientpb.KillJobReq{ID: id})
	return err
}

func (s *Service) StartListener(protocol, host string, port uint32, domains string) error {
	c, err := rpcwrap.Client(s.rpc)
	if err != nil {
		return err
	}
	ctx := context.Background()

	switch strings.ToLower(protocol) {
	case "mtls":
		_, err := c.RPC.StartMTLSListener(ctx, &clientpb.MTLSListenerReq{Host: host, Port: port})
		return err
	case "http":
		_, err := c.RPC.StartHTTPListener(ctx, &clientpb.HTTPListenerReq{Host: host, Port: port, Secure: false})
		return err
	case "https":
		_, err := c.RPC.StartHTTPSListener(ctx, &clientpb.HTTPListenerReq{Host: host, Port: port, Secure: true})
		return err
	case "dns":
		var doms []string
		for _, d := range strings.Split(domains, ",") {
			if d = strings.TrimSpace(d); d != "" {
				doms = append(doms, d)
			}
		}
		_, err := c.RPC.StartDNSListener(ctx, &clientpb.DNSListenerReq{Domains: doms, Host: host, Port: port})
		return err
	default:
		return fmt.Errorf("unknown listener protocol: %s", protocol)
	}
}
