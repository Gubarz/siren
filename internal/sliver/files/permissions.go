package files

import (
	"context"

	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
)

func (s *Service) Chmod(sessionID, path, mode string, recursive bool) error {
	return s.runVoidCommand(sessionID, func(ctx context.Context, req *commonpb.Request) (protobufResponse, error) {
		return s.rpc.RPC.Chmod(ctx, &sliverpb.ChmodReq{
			Request:   req,
			Path:      path,
			FileMode:  mode,
			Recursive: recursive,
		})
	})
}

func (s *Service) Chown(sessionID, path, uid, gid string, recursive bool) error {
	return s.runVoidCommand(sessionID, func(ctx context.Context, req *commonpb.Request) (protobufResponse, error) {
		return s.rpc.RPC.Chown(ctx, &sliverpb.ChownReq{
			Request:   req,
			Path:      path,
			Uid:       uid,
			Gid:       gid,
			Recursive: recursive,
		})
	})
}

func (s *Service) Chtimes(sessionID, path string, atimeUnix, mtimeUnix int64) error {
	return s.runVoidCommand(sessionID, func(ctx context.Context, req *commonpb.Request) (protobufResponse, error) {
		return s.rpc.RPC.Chtimes(ctx, &sliverpb.ChtimesReq{
			Request: req,
			Path:    path,
			ATime:   atimeUnix,
			MTime:   mtimeUnix,
		})
	})
}
