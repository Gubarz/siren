package procs

import (
	"context"
	"encoding/base64"
	"time"

	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"

	"sliver-gui/internal/sliver/rpc"
)

const defaultRPCTimeout = 60 * time.Second

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

	req := &sliverpb.PsReq{
		FullInfo: fullInfo,
		Request: &commonpb.Request{
			SessionID: sessionID,
			Timeout:   int64(defaultRPCTimeout - time.Nanosecond),
		},
	}

	return s.rpc.RPC.Ps(ctx, req)
}

func (s *Service) KillProcess(sessionID string, pid int32) error {
	if !s.rpc.Connected() {
		return rpc.ErrNotConnected
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultRPCTimeout)
	defer cancel()

	req := &sliverpb.TerminateReq{
		Request: &commonpb.Request{
			SessionID: sessionID,
			Timeout:   int64(defaultRPCTimeout - time.Nanosecond),
		},
		Pid:   pid,
		Force: true,
	}

	_, err := s.rpc.RPC.Terminate(ctx, req)
	return err
}

func (s *Service) TakeScreenshot(sessionID string) (string, error) {
	if !s.rpc.Connected() {
		return "", rpc.ErrNotConnected
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultRPCTimeout)
	defer cancel()

	req := &sliverpb.ScreenshotReq{
		Request: &commonpb.Request{
			SessionID: sessionID,
			Timeout:   int64(defaultRPCTimeout - time.Nanosecond),
		},
	}

	resp, err := s.rpc.RPC.Screenshot(ctx, req)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(resp.Data), nil
}
