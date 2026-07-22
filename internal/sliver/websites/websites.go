// Package websites wraps sliver's Website RPCs — the teamserver keeps a
// per-name key/value blob store used by the http-c2 handler to serve
// arbitrary content (droppers, decoy pages, license-check payloads, etc.).
package websites

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/bishopfox/sliver/protobuf/clientpb"

	"sliver-gui/internal/sliver/rpc"
)

const rpcTimeout = 60 * time.Second

// ctxWithTimeout is the shared timeout helper for the whole package — every
// wrapper here talks to the same server, so hardcoding one deadline is
// simpler than piping context deadlines through the frontend.
func ctxWithTimeout() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), rpcTimeout)
}

type Service struct {
	rpc *rpc.Client
}

func New(client *rpc.Client) *Service {
	return &Service{rpc: client}
}

func (s *Service) Close() {}

// GetWebsite returns the full record for one site — includes the WebContent
// map so the editor can render every registered path.
func (s *Service) GetWebsite(name string) (*clientpb.Website, error) {
	if !s.rpc.Connected() {
		return nil, rpc.ErrNotConnected
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("website name is required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), rpcTimeout)
	defer cancel()
	return s.rpc.RPC.Website(ctx, &clientpb.Website{Name: name})
}

// RemoveWebsite drops the whole site — every registered path with it.
func (s *Service) RemoveWebsite(name string) error {
	if !s.rpc.Connected() {
		return rpc.ErrNotConnected
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("website name is required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), rpcTimeout)
	defer cancel()
	_, err := s.rpc.RPC.WebsiteRemove(ctx, &clientpb.Website{Name: name})
	return err
}
