package files

import (
	"context"
	"time"

	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"

	"siren/internal/sliver/rpc"
	"siren/internal/sliver/rpcwrap"
	"siren/internal/wailsadapter"
)

const defaultRPCTimeout = 5 * time.Minute

type Service struct {
	rpc.Emitter
	rpc     *rpc.Client
	ui      *wailsadapter.Bridge
	history *HistoryStore
	dl      func(ctx context.Context, in *sliverpb.DownloadReq) (*sliverpb.Download, error)
}

func New(rpcClient *rpc.Client) *Service {
	return &Service{
		rpc:     rpcClient,
		Emitter: rpc.NewEmitter(rpcClient),
		history: NewHistoryStore(),
	}
}

func (s *Service) SetUI(ui *wailsadapter.Bridge) {
	s.ui = ui
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

func runPathCommand[Req proto.Message, Resp PathResponse](
	s *Service,
	sessionID string,
	method func(rpcpb.SliverRPCClient, context.Context, Req, ...grpc.CallOption) (Resp, error),
	req Req,
) (string, error) {
	resp, err := rpcwrap.TargetCall(s.rpc, sessionID, defaultRPCTimeout, method, req)
	if err != nil {
		return "", err
	}
	return resp.GetPath(), nil
}

func runVoidCommand[Req proto.Message, Resp rpcwrap.TargetResponse](
	s *Service,
	sessionID string,
	method func(rpcpb.SliverRPCClient, context.Context, Req, ...grpc.CallOption) (Resp, error),
	req Req,
) error {
	_, err := rpcwrap.TargetCall(s.rpc, sessionID, defaultRPCTimeout, method, req)
	return err
}

func (s *Service) MakeDir(sessionID, path string) error {
	return runVoidCommand(s, sessionID, rpcpb.SliverRPCClient.Mkdir, &sliverpb.MkdirReq{Path: path})
}

func (s *Service) RemovePath(sessionID, path string, recursive bool) error {
	return runVoidCommand(s, sessionID, rpcpb.SliverRPCClient.Rm, &sliverpb.RmReq{
		Path:      path,
		Recursive: recursive,
		Force:     true,
	})
}

func (s *Service) RenamePath(sessionID, src, dst string) error {
	return runVoidCommand(s, sessionID, rpcpb.SliverRPCClient.Mv, &sliverpb.MvReq{
		Src: src,
		Dst: dst,
	})
}

func (s *Service) Cd(sessionID, path string) (string, error) {
	return runPathCommand(s, sessionID, rpcpb.SliverRPCClient.Cd, &sliverpb.CdReq{Path: path})
}

func (s *Service) Pwd(sessionID string) (string, error) {
	return runPathCommand(s, sessionID, rpcpb.SliverRPCClient.Pwd, &sliverpb.PwdReq{})
}

func (s *Service) GetFileList(sessionID string, path string) (*sliverpb.Ls, error) {
	if path == "" {
		path = "."
	}
	return rpcwrap.TargetCall(s.rpc, sessionID, defaultRPCTimeout, rpcpb.SliverRPCClient.Ls, &sliverpb.LsReq{
		Path: path,
	})
}
