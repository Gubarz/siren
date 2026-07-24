package console

import (
	"context"
	"fmt"
	"time"

	commandflags "github.com/bishopfox/sliver/client/command/flags"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
)

func (s *Service) GetTokenPrivs(sessionID string) (*sliverpb.GetPrivs, error) {
	sess, beacon, err := s.FindTarget(sessionID)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	privs, err := s.rpc.RPC.GetPrivs(ctx, &sliverpb.GetPrivsReq{
		Request: targetRequestFromConsole(sess, beacon),
	})
	if err != nil {
		return nil, fmt.Errorf("getprivs: %w", err)
	}
	if err := s.rpc.AwaitAsyncResponse(ctx, privs, privs); err != nil {
		return nil, fmt.Errorf("getprivs: %w", err)
	}
	return privs, nil
}

func (s *Service) RevToSelfToken(sessionID string) error {
	sess, beacon, err := s.FindTarget(sessionID)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	revert, err := s.rpc.RPC.RevToSelf(ctx, &sliverpb.RevToSelfReq{
		Request: targetRequestFromConsole(sess, beacon),
	})
	if err != nil {
		return fmt.Errorf("rev2self: %w", err)
	}
	return s.rpc.AwaitAsyncResponse(ctx, revert, revert)
}

func targetRequestFromConsole(sess *clientpb.Session, beacon *clientpb.Beacon) *commonpb.Request {
	if sess != nil {
		return &commonpb.Request{SessionID: sess.ID, Timeout: commandflags.DefaultTimeout}
	}
	if beacon != nil {
		return &commonpb.Request{Async: true, BeaconID: beacon.ID, Timeout: commandflags.DefaultTimeout}
	}
	return &commonpb.Request{Timeout: commandflags.DefaultTimeout}
}
