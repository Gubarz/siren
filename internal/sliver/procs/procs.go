package procs

import (
	"context"
	"encoding/base64"
	"time"

	"github.com/bishopfox/sliver/protobuf/sliverpb"

	"sliver-gui/internal/sliver/rpc"
)

const defaultRPCTimeout = 5 * time.Minute

type Service struct {
	rpc *rpc.Client
}

func New(rpc *rpc.Client) *Service {
	return &Service{rpc: rpc}
}

func (s *Service) GetProcessList(sessionID string, fullInfo bool) (*sliverpb.Ps, error) {
	if !s.rpc.Connected() {
		return nil, rpc.ErrNotConnected
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultRPCTimeout)
	defer cancel()

	request, err := s.rpc.TargetRequest(sessionID, defaultRPCTimeout)
	if err != nil {
		return nil, err
	}
	req := &sliverpb.PsReq{
		FullInfo: fullInfo,
		Request:  request,
	}
	resp, err := s.rpc.RPC.Ps(ctx, req)
	if err != nil {
		return nil, err
	}
	if err := s.rpc.AwaitAsyncResponse(ctx, resp, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

func (s *Service) KillProcess(sessionID string, pid int32) error {
	if !s.rpc.Connected() {
		return rpc.ErrNotConnected
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultRPCTimeout)
	defer cancel()

	request, err := s.rpc.TargetRequest(sessionID, defaultRPCTimeout)
	if err != nil {
		return err
	}
	req := &sliverpb.TerminateReq{
		Request: request,
		Pid:     pid,
		Force:   true,
	}

	resp, err := s.rpc.RPC.Terminate(ctx, req)
	if err != nil {
		return err
	}
	return s.rpc.AwaitAsyncResponse(ctx, resp, resp)
}

func (s *Service) TakeScreenshot(sessionID string) (string, error) {
	if !s.rpc.Connected() {
		return "", rpc.ErrNotConnected
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultRPCTimeout)
	defer cancel()

	request, err := s.rpc.TargetRequest(sessionID, defaultRPCTimeout)
	if err != nil {
		return "", err
	}
	req := &sliverpb.ScreenshotReq{
		Request: request,
	}

	resp, err := s.rpc.RPC.Screenshot(ctx, req)
	if err != nil {
		return "", err
	}
	if err := s.rpc.AwaitAsyncResponse(ctx, resp, resp); err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(resp.Data), nil
}
