package agents

import (
	"context"
	"fmt"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"

	"sliver-gui/internal/sliver/console"
	"sliver-gui/internal/sliver/rpc"
)

type Service struct {
	rpc     *rpc.Client
	console *console.Service
}

func New(rpc *rpc.Client, con *console.Service) *Service {
	return &Service{rpc: rpc, console: con}
}

func (s *Service) Sessions() (*clientpb.Sessions, error) {
	if !s.rpc.Connected() {
		return nil, rpc.ErrNotConnected
	}
	result, err := s.rpc.RPC.GetSessions(context.Background(), &commonpb.Empty{})
	if err == nil {
		s.rpc.PopulateSessions(result)
	}
	return result, err
}

func (s *Service) Beacons() (*clientpb.Beacons, error) {
	if !s.rpc.Connected() {
		return nil, rpc.ErrNotConnected
	}
	result, err := s.rpc.RPC.GetBeacons(context.Background(), &commonpb.Empty{})
	if err == nil {
		s.rpc.PopulateBeacons(result)
	}
	return result, err
}

func (s *Service) Kill(id string) error {
	if !s.rpc.Connected() {
		return rpc.ErrNotConnected
	}
	sess, beacon, err := s.console.FindTarget(id)
	if err != nil {
		return err
	}
	req := &commonpb.Request{}
	if sess != nil {
		req.SessionID = sess.ID
	} else {
		req.BeaconID = beacon.ID
	}
	_, err = s.rpc.RPC.Kill(context.Background(), &sliverpb.KillReq{Request: req, Force: true})
	return err
}

func (s *Service) Rename(id, name string) error {
	if !s.rpc.Connected() {
		return rpc.ErrNotConnected
	}
	sess, beacon, err := s.console.FindTarget(id)
	if err != nil {
		return err
	}
	req := &clientpb.RenameReq{Name: name}
	if sess != nil {
		req.SessionID = sess.ID
	} else {
		req.BeaconID = beacon.ID
	}
	_, err = s.rpc.RPC.Rename(context.Background(), req)
	return err
}

func (s *Service) RemoveBeacon(id string) error {
	if !s.rpc.Connected() {
		return rpc.ErrNotConnected
	}
	if id == "" {
		return fmt.Errorf("beacon ID is required")
	}
	_, err := s.rpc.RPC.RmBeacon(context.Background(), &clientpb.Beacon{ID: id})
	return err
}

func (s *Service) Version() (*clientpb.Version, error) {
	if !s.rpc.Connected() {
		return nil, rpc.ErrNotConnected
	}
	return s.rpc.RPC.GetVersion(context.Background(), &commonpb.Empty{})
}
