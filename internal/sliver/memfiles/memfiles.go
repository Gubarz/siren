package memfiles

import (
	"context"

	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"

	"sliver-gui/internal/sliver/rpc"
)

type Service struct {
	rpc *rpc.Client
}

func New(rpc *rpc.Client) *Service {
	return &Service{rpc: rpc}
}

func (s *Service) Close() {}

func (s *Service) List(sessionID string) (*sliverpb.Ls, error) {
	if !s.rpc.Connected() {
		return nil, rpc.ErrNotConnected
	}
	return s.rpc.RPC.MemfilesList(context.Background(), &sliverpb.MemfilesListReq{
		Request: &commonpb.Request{SessionID: sessionID},
	})
}

func (s *Service) Add(sessionID string) (*sliverpb.MemfilesAdd, error) {
	if !s.rpc.Connected() {
		return nil, rpc.ErrNotConnected
	}
	return s.rpc.RPC.MemfilesAdd(context.Background(), &sliverpb.MemfilesAddReq{
		Request: &commonpb.Request{SessionID: sessionID},
	})
}

func (s *Service) Remove(sessionID string, fd int64) (*sliverpb.MemfilesRm, error) {
	if !s.rpc.Connected() {
		return nil, rpc.ErrNotConnected
	}
	return s.rpc.RPC.MemfilesRm(context.Background(), &sliverpb.MemfilesRmReq{
		Fd:      fd,
		Request: &commonpb.Request{SessionID: sessionID},
	})
}
