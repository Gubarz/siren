package console

import (
	"context"
	"fmt"

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

	privs, err := s.rpc.RPC.GetPrivs(context.Background(), &sliverpb.GetPrivsReq{
		Request: targetRequestFromConsole(sess, beacon),
	})
	if err != nil {
		return nil, fmt.Errorf("getprivs: %w", err)
	}
	if privs.Response != nil && privs.Response.GetErr() != "" {
		return nil, fmt.Errorf("%s", privs.Response.GetErr())
	}
	return privs, nil
}

func (s *Service) RevToSelfToken(sessionID string) error {
	sess, beacon, err := s.FindTarget(sessionID)
	if err != nil {
		return err
	}

	revert, err := s.rpc.RPC.RevToSelf(context.Background(), &sliverpb.RevToSelfReq{
		Request: targetRequestFromConsole(sess, beacon),
	})
	if err != nil {
		return fmt.Errorf("rev2self: %w", err)
	}
	if revert.Response != nil && revert.Response.GetErr() != "" {
		return fmt.Errorf("%s", revert.Response.GetErr())
	}
	return nil
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
