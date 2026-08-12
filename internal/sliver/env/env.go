package env

import (
	"context"
	"fmt"
	"time"

	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"

	"siren/internal/sliver/rpc"
)

type Service struct {
	rpc *rpc.Client
}

const requestTimeout = 5 * time.Minute

func New(rpc *rpc.Client) *Service {
	return &Service{rpc: rpc}
}

func (s *Service) GetEnv(sessionID string) (*sliverpb.EnvInfo, error) {
	if !s.rpc.Connected() {
		return nil, rpc.ErrNotConnected
	}

	request, err := s.rpc.TargetRequest(sessionID, requestTimeout)
	if err != nil {
		return nil, err
	}
	req := &sliverpb.EnvReq{Request: request}

	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()

	resp, err := s.rpc.RPC.GetEnv(ctx, req)
	if err != nil {
		return nil, err
	}
	if err := s.rpc.AwaitAsyncResponse(ctx, resp, resp); err != nil {
		return nil, fmt.Errorf("getenv: %w", err)
	}
	return resp, nil
}

func (s *Service) SetEnv(sessionID, key, value string) error {
	if !s.rpc.Connected() {
		return rpc.ErrNotConnected
	}

	request, err := s.rpc.TargetRequest(sessionID, requestTimeout)
	if err != nil {
		return err
	}
	req := &sliverpb.SetEnvReq{
		Variable: &commonpb.EnvVar{Key: key, Value: value},
		Request:  request,
	}

	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()
	resp, err := s.rpc.RPC.SetEnv(ctx, req)
	if err != nil {
		return err
	}
	return s.rpc.AwaitAsyncResponse(ctx, resp, resp)
}

func (s *Service) UnsetEnv(sessionID, name string) error {
	if !s.rpc.Connected() {
		return rpc.ErrNotConnected
	}

	request, err := s.rpc.TargetRequest(sessionID, requestTimeout)
	if err != nil {
		return err
	}
	req := &sliverpb.UnsetEnvReq{
		Name:    name,
		Request: request,
	}

	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()
	resp, err := s.rpc.RPC.UnsetEnv(ctx, req)
	if err != nil {
		return err
	}
	return s.rpc.AwaitAsyncResponse(ctx, resp, resp)
}
