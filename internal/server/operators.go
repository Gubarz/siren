package server

import (
	"context"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"

	"sliver-gui/internal/rpc"
)

type Service struct {
	rpc serverRPC
}

func New(rpc *rpc.Client) *Service {
	return &Service{rpc: liveServerRPC{client: rpc}}
}

func (s *Service) GetOperators() (*clientpb.Operators, error) {
	if !s.rpc.Connected() {
		return nil, rpc.ErrNotConnected
	}
	return s.rpc.GetOperators(context.Background(), &commonpb.Empty{})
}
