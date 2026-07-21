package files

import (
	"context"
	"time"

	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"

	"sliver-gui/internal/rpc"
)

const defaultRPCTimeout = 60 * time.Second

type Service struct {
	rpc     *rpc.Client
	ctx     context.Context
	history *HistoryStore
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
	GetResponse() *commonpb.Response
	GetPath() string
}

func (s *Service) runPathCommand(execute func() (PathResponse, error)) (string, error) {
	if !s.rpc.Connected() {
		return "", rpc.ErrNotConnected
	}
	resp, err := execute()
	if err != nil {
		return "", err
	}
	if err := rpc.CheckResponse(resp); err != nil {
		return "", err
	}
	return resp.GetPath(), nil
}

func (s *Service) runVoidCommand(execute func() (rpc.ResponseWithError, error)) error {
	if !s.rpc.Connected() {
		return rpc.ErrNotConnected
	}
	resp, err := execute()
	if err != nil {
		return err
	}
	return rpc.CheckResponse(resp)
}

func (s *Service) MakeDir(sessionID, path string) error {
	return s.runVoidCommand(func() (rpc.ResponseWithError, error) {
		return s.rpc.RPC.Mkdir(context.Background(), &sliverpb.MkdirReq{
			Request: &commonpb.Request{SessionID: sessionID}, Path: path,
		})
	})
}

func (s *Service) RemovePath(sessionID, path string, recursive bool) error {
	return s.runVoidCommand(func() (rpc.ResponseWithError, error) {
		return s.rpc.RPC.Rm(context.Background(), &sliverpb.RmReq{
			Request: &commonpb.Request{SessionID: sessionID}, Path: path, Recursive: recursive, Force: true,
		})
	})
}

func (s *Service) RenamePath(sessionID, src, dst string) error {
	return s.runVoidCommand(func() (rpc.ResponseWithError, error) {
		return s.rpc.RPC.Mv(context.Background(), &sliverpb.MvReq{
			Request: &commonpb.Request{SessionID: sessionID}, Src: src, Dst: dst,
		})
	})
}

func (s *Service) Cd(sessionID, path string) (string, error) {
	return s.runPathCommand(func() (PathResponse, error) {
		return s.rpc.RPC.Cd(context.Background(), &sliverpb.CdReq{
			Request: &commonpb.Request{SessionID: sessionID},
			Path:    path,
		})
	})
}

func (s *Service) Pwd(sessionID string) (string, error) {
	return s.runPathCommand(func() (PathResponse, error) {
		return s.rpc.RPC.Pwd(context.Background(), &sliverpb.PwdReq{
			Request: &commonpb.Request{SessionID: sessionID},
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

	req := &sliverpb.LsReq{
		Request: &commonpb.Request{
			SessionID: sessionID,
			Timeout:   int64(defaultRPCTimeout / time.Second),
		},
		Path: path,
	}

	return s.rpc.RPC.Ls(ctx, req)
}
