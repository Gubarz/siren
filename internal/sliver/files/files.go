package files

import (
	"context"
	"time"

	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"google.golang.org/protobuf/proto"

	"sliver-gui/internal/sliver/rpc"
)

const defaultRPCTimeout = 5 * time.Minute

type Service struct {
	rpc     *rpc.Client
	ctx     context.Context
	history *HistoryStore
	dl      func(ctx context.Context, in *sliverpb.DownloadReq) (*sliverpb.Download, error)
}

func New(rpc *rpc.Client) *Service {
	return &Service{
		rpc:     rpc,
		history: NewHistoryStore(),
	}
}

func (s *Service) SetCtx(ctx context.Context) {
	s.ctx = ctx
}

func (s *Service) SetHistoryStore(h *HistoryStore) {
	s.history = h
}

func (s *Service) GetDownloadHistory(sessionID, remotePath string) ([]DownloadRecord, error) {
	if s.history == nil {
		return []DownloadRecord{}, nil
	}
	return s.history.GetHistory(sessionID, remotePath), nil
}

func (s *Service) GetAllDownloadHistory() ([]DownloadRecord, error) {
	if s.history == nil {
		return []DownloadRecord{}, nil
	}
	return s.history.GetAllHistory(), nil
}

func (s *Service) ClearDownloadHistory(sessionID, remotePath string) error {
	if s.history != nil {
		s.history.ClearHistory(sessionID, remotePath)
	}
	return nil
}

type PathResponse interface {
	rpc.ResponseWithError
	proto.Message
	GetPath() string
}

type protobufResponse interface {
	rpc.ResponseWithError
	proto.Message
}

func (s *Service) runPathCommand(
	sessionID string,
	execute func(context.Context, *commonpb.Request) (PathResponse, error),
) (string, error) {
	if !s.rpc.Connected() {
		return "", rpc.ErrNotConnected
	}
	req, err := s.rpc.TargetRequest(sessionID, defaultRPCTimeout)
	if err != nil {
		return "", err
	}
	ctx, cancel := context.WithTimeout(context.Background(), defaultRPCTimeout)
	defer cancel()
	resp, err := execute(ctx, req)
	if err != nil {
		return "", err
	}
	if err := s.rpc.AwaitAsyncResponse(ctx, resp, resp); err != nil {
		return "", err
	}
	return resp.GetPath(), nil
}

func (s *Service) runVoidCommand(
	sessionID string,
	execute func(context.Context, *commonpb.Request) (protobufResponse, error),
) error {
	if !s.rpc.Connected() {
		return rpc.ErrNotConnected
	}
	req, err := s.rpc.TargetRequest(sessionID, defaultRPCTimeout)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), defaultRPCTimeout)
	defer cancel()
	resp, err := execute(ctx, req)
	if err != nil {
		return err
	}
	return s.rpc.AwaitAsyncResponse(ctx, resp, resp)
}

func (s *Service) MakeDir(sessionID, path string) error {
	return s.runVoidCommand(sessionID, func(ctx context.Context, req *commonpb.Request) (protobufResponse, error) {
		return s.rpc.RPC.Mkdir(ctx, &sliverpb.MkdirReq{
			Request: req, Path: path,
		})
	})
}

func (s *Service) RemovePath(sessionID, path string, recursive bool) error {
	return s.runVoidCommand(sessionID, func(ctx context.Context, req *commonpb.Request) (protobufResponse, error) {
		return s.rpc.RPC.Rm(ctx, &sliverpb.RmReq{
			Request: req, Path: path, Recursive: recursive, Force: true,
		})
	})
}

func (s *Service) RenamePath(sessionID, src, dst string) error {
	return s.runVoidCommand(sessionID, func(ctx context.Context, req *commonpb.Request) (protobufResponse, error) {
		return s.rpc.RPC.Mv(ctx, &sliverpb.MvReq{
			Request: req, Src: src, Dst: dst,
		})
	})
}

func (s *Service) Cd(sessionID, path string) (string, error) {
	return s.runPathCommand(sessionID, func(ctx context.Context, req *commonpb.Request) (PathResponse, error) {
		return s.rpc.RPC.Cd(ctx, &sliverpb.CdReq{
			Request: req,
			Path:    path,
		})
	})
}

func (s *Service) Pwd(sessionID string) (string, error) {
	return s.runPathCommand(sessionID, func(ctx context.Context, req *commonpb.Request) (PathResponse, error) {
		return s.rpc.RPC.Pwd(ctx, &sliverpb.PwdReq{
			Request: req,
		})
	})
}

func (s *Service) GetFileList(sessionID string, path string) (*sliverpb.Ls, error) {
	if !s.rpc.Connected() {
		return nil, rpc.ErrNotConnected
	}

	if path == "" {
		path = "."
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultRPCTimeout)
	defer cancel()

	request, err := s.rpc.TargetRequest(sessionID, defaultRPCTimeout)
	if err != nil {
		return nil, err
	}
	req := &sliverpb.LsReq{
		Request: request,
		Path: path,
	}

	resp, err := s.rpc.RPC.Ls(ctx, req)
	if err != nil {
		return nil, err
	}
	if err := s.rpc.AwaitAsyncResponse(ctx, resp, resp); err != nil {
		return nil, err
	}
	return resp, nil
}
