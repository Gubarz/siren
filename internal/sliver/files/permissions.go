package files

import (
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
)

func (s *Service) Chmod(sessionID, path, mode string, recursive bool) error {
	return runVoidCommand(s, sessionID, rpcpb.SliverRPCClient.Chmod, &sliverpb.ChmodReq{
		Path:      path,
		FileMode:  mode,
		Recursive: recursive,
	})
}

func (s *Service) Chown(sessionID, path, uid, gid string, recursive bool) error {
	return runVoidCommand(s, sessionID, rpcpb.SliverRPCClient.Chown, &sliverpb.ChownReq{
		Path:      path,
		Uid:       uid,
		Gid:       gid,
		Recursive: recursive,
	})
}

func (s *Service) Chtimes(sessionID, path string, atimeUnix, mtimeUnix int64) error {
	return runVoidCommand(s, sessionID, rpcpb.SliverRPCClient.Chtimes, &sliverpb.ChtimesReq{
		Path:  path,
		ATime: atimeUnix,
		MTime: mtimeUnix,
	})
}
