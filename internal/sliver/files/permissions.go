package files

import (
	"context"

	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"

	"sliver-gui/internal/sliver/rpc"
)

func (s *Service) Chmod(sessionID, path, mode string, recursive bool) error {
	return s.runVoidCommand(func() (rpc.ResponseWithError, error) {
		return s.rpc.RPC.Chmod(context.Background(), &sliverpb.ChmodReq{
			Request:   &commonpb.Request{SessionID: sessionID},
			Path:      path,
			FileMode:  mode,
			Recursive: recursive,
		})
	})
}

func (s *Service) Chown(sessionID, path, uid, gid string, recursive bool) error {
	return s.runVoidCommand(func() (rpc.ResponseWithError, error) {
		return s.rpc.RPC.Chown(context.Background(), &sliverpb.ChownReq{
			Request:   &commonpb.Request{SessionID: sessionID},
			Path:      path,
			Uid:       uid,
			Gid:       gid,
			Recursive: recursive,
		})
	})
}

func (s *Service) Chtimes(sessionID, path string, atimeUnix, mtimeUnix int64) error {
	return s.runVoidCommand(func() (rpc.ResponseWithError, error) {
		return s.rpc.RPC.Chtimes(context.Background(), &sliverpb.ChtimesReq{
			Request: &commonpb.Request{SessionID: sessionID},
			Path:    path,
			ATime:   atimeUnix,
			MTime:   mtimeUnix,
		})
	})
}
