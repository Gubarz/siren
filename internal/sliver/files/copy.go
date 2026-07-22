package files

import (
	"context"

	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"

	"sliver-gui/internal/sliver/rpc"
)

func (s *Service) CopyPath(sessionID, src, dst string) (int64, error) {
	if !s.rpc.Connected() {
		return 0, rpc.ErrNotConnected
	}
	resp, err := s.rpc.RPC.Cp(context.Background(), &sliverpb.CpReq{
		Request: &commonpb.Request{SessionID: sessionID},
		Src:     src,
		Dst:     dst,
	})
	if err != nil {
		return 0, err
	}
	if err := rpc.CheckResponse(resp); err != nil {
		return 0, err
	}
	return resp.BytesWritten, nil
}
