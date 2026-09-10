package agents

import (
	"context"
	"fmt"
	"log"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"

	knownagents "siren/internal/localstate/agents"
	"siren/internal/sliver/console"
	"siren/internal/sliver/rpc"
	"siren/internal/sliver/rpcwrap"
)

type Service struct {
	rpc     *rpc.Client
	console *console.Service
	known   *knownagents.Service
}

func New(rpc *rpc.Client, con *console.Service) *Service {
	return &Service{rpc: rpc, console: con}
}

func (s *Service) SetKnownAgents(k *knownagents.Service) {
	s.known = k
}

func (s *Service) Sessions() (*clientpb.Sessions, error) {
	result, err := rpcwrap.Call(s.rpc, rpcpb.SliverRPCClient.GetSessions, &commonpb.Empty{})
	if err == nil {
		s.rpc.PopulateSessions(result)
		s.observe(sessionRecords(result))
	}
	return result, err
}

func (s *Service) Beacons() (*clientpb.Beacons, error) {
	result, err := rpcwrap.Call(s.rpc, rpcpb.SliverRPCClient.GetBeacons, &commonpb.Empty{})
	if err == nil {
		// Beacons are never persisted as known agents: the teamserver keeps
		// beacon records and the UI renders DEAD beacons from live data, so
		// known-agent state tracks vanished sessions only.
		s.rpc.PopulateBeacons(result)
	}
	return result, err
}

// observe records the live list as best-effort history; a failure to persist
// must never fail the list fetch. Beacon records are dropped here as a
// defensive invariant: the teamserver keeps beacon records and the UI renders
// DEAD beacons from live data, so known-agent state tracks vanished sessions
// only. Persisting beacons would leave every beacon ever seen permanently
// active (sliver emits no beacon-lost event).
func (s *Service) observe(records []knownagents.Record) {
	if s.known == nil || len(records) == 0 {
		return
	}
	filtered := make([]knownagents.Record, 0, len(records))
	for i := range records {
		if records[i].Kind == "beacon" {
			continue
		}
		filtered = append(filtered, records[i])
	}
	if len(filtered) == 0 {
		return
	}
	if err := s.known.Observe(filtered); err != nil {
		log.Printf("agents: observe live agents: %v", err)
	}
}

func sessionRecords(sessions *clientpb.Sessions) []knownagents.Record {
	records := []knownagents.Record{}
	if sessions == nil {
		return records
	}
	for _, sess := range sessions.Sessions {
		if sess == nil {
			continue
		}
		records = append(records, knownagents.Record{
			ID:            sess.ID,
			Kind:          "session",
			Name:          sess.Name,
			Hostname:      sess.Hostname,
			Username:      sess.Username,
			OS:            sess.OS,
			Arch:          sess.Arch,
			RemoteAddress: sess.RemoteAddress,
			Transport:     sess.Transport,
		})
	}
	return records
}

func (s *Service) Kill(id string) error {
	c, err := rpcwrap.Client(s.rpc)
	if err != nil {
		return err
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
	_, err = c.RPC.Kill(context.Background(), &sliverpb.KillReq{Request: req, Force: true})
	return err
}

func (s *Service) Rename(id, name string) error {
	c, err := rpcwrap.Client(s.rpc)
	if err != nil {
		return err
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
	_, err = c.RPC.Rename(context.Background(), req)
	return err
}

func (s *Service) RemoveBeacon(id string) error {
	c, err := rpcwrap.Client(s.rpc)
	if err != nil {
		return err
	}
	if id == "" {
		return fmt.Errorf("beacon ID is required")
	}
	_, err = c.RPC.RmBeacon(context.Background(), &clientpb.Beacon{ID: id})
	return err
}

func (s *Service) Version() (*clientpb.Version, error) {
	return rpcwrap.Call(s.rpc, rpcpb.SliverRPCClient.GetVersion, &commonpb.Empty{})
}
