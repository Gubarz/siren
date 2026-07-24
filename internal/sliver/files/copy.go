package files

import (
	"context"

	"github.com/bishopfox/sliver/protobuf/sliverpb"

	"sliver-gui/internal/sliver/rpc"
)

func (s *Service) CopyPath(sessionID, src, dst string) (int64, error) {
	if !s.rpc.Connected() {
		return 0, rpc.ErrNotConnected
	}
	req, err := s.rpc.TargetRequest(sessionID, defaultRPCTimeout)
	if err != nil {
		return 0, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), defaultRPCTimeout)
	defer cancel()
	resp, err := s.rpc.RPC.Cp(ctx, &sliverpb.CpReq{
		Request: req,
		Src:     src,
		Dst:     dst,
	})
	if err != nil {
		return 0, err
	}
	if err := s.rpc.AwaitAsyncResponse(ctx, resp, resp); err != nil {
		return 0, err
	}
	return resp.BytesWritten, nil
}
