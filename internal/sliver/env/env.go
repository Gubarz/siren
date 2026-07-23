package env

import (
	"context"
	"fmt"
	"time"

	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"

	"sliver-gui/internal/sliver/rpc"
)

type Service struct {
	rpc *rpc.Client
}

func New(rpc *rpc.Client) *Service {
	return &Service{rpc: rpc}
}

func (s *Service) GetEnv(sessionID string) (*sliverpb.EnvInfo, error) {
	if !s.rpc.Connected() {
		return nil, rpc.ErrNotConnected
	}

	req := &sliverpb.EnvReq{
		Request: s.requestFor(sessionID),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	resp, err := s.rpc.RPC.GetEnv(ctx, req)
	if err != nil {
		return nil, err
	}
	if resp.Response != nil && resp.Response.Err != "" {
		return nil, fmt.Errorf("implant: %s", resp.Response.Err)
	}
	return resp, nil
}

func (s *Service) SetEnv(sessionID, key, value string) error {
	if !s.rpc.Connected() {
		return rpc.ErrNotConnected
	}

	req := &sliverpb.SetEnvReq{
		Variable: &commonpb.EnvVar{Key: key, Value: value},
		Request:  s.requestFor(sessionID),
	}

	resp, err := s.rpc.RPC.SetEnv(context.Background(), req)
	if err != nil {
		return err
	}
	if resp.Response != nil && resp.Response.Err != "" {
		return fmt.Errorf("implant: %s", resp.Response.Err)
	}
	return nil
}

func (s *Service) UnsetEnv(sessionID, name string) error {
	if !s.rpc.Connected() {
		return rpc.ErrNotConnected
	}

	req := &sliverpb.UnsetEnvReq{
		Name:    name,
		Request: s.requestFor(sessionID),
	}

	resp, err := s.rpc.RPC.UnsetEnv(context.Background(), req)
	if err != nil {
		return err
	}
	if resp.Response != nil && resp.Response.Err != "" {
		return fmt.Errorf("implant: %s", resp.Response.Err)
	}
	return nil
}

func (s *Service) requestFor(sessionID string) *commonpb.Request {
	req := &commonpb.Request{
		SessionID: sessionID,
		Timeout:   int64((60 * time.Second) / time.Second),
	}
	if sess := s.rpc.LookupSession(sessionID); sess == nil {
		if beacon := s.rpc.LookupBeacon(sessionID); beacon != nil {
			req.BeaconID = sessionID
			req.Async = true
		}
	}
	return req
}
