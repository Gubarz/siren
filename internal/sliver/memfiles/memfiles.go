package memfiles

import (
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"

	"siren/internal/sliver/rpc"
	"siren/internal/sliver/rpcwrap"
)

type Service struct {
	rpc *rpc.Client
}

func New(rpc *rpc.Client) *Service {
	return &Service{rpc: rpc}
}

func (s *Service) Close() {}

func (s *Service) List(sessionID string) (*sliverpb.Ls, error) {
	return rpcwrap.CallTarget(s.rpc, rpcpb.SliverRPCClient.MemfilesList, &sliverpb.MemfilesListReq{}, sessionID)
}

func (s *Service) Add(sessionID string) (*sliverpb.MemfilesAdd, error) {
	return rpcwrap.CallTarget(s.rpc, rpcpb.SliverRPCClient.MemfilesAdd, &sliverpb.MemfilesAddReq{}, sessionID)
}

func (s *Service) Remove(sessionID string, fd int64) (*sliverpb.MemfilesRm, error) {
	return rpcwrap.CallTarget(s.rpc, rpcpb.SliverRPCClient.MemfilesRm, &sliverpb.MemfilesRmReq{Fd: fd}, sessionID)
}
