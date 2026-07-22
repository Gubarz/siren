package automationexec

import (
	"context"
	"fmt"

	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"google.golang.org/protobuf/proto"

	"sliver-gui/internal/automation"
	"sliver-gui/internal/sliver/beacons"
	"sliver-gui/internal/sliver/console"
	"sliver-gui/internal/sliver/rpc"
)

type Executor struct {
	console *console.Service
	beacons *beacons.Service
}

func NewExecutor(con *console.Service, beac *beacons.Service) *Executor {
	return &Executor{console: con, beacons: beac}
}

func (e *Executor) Execute(ctx context.Context, targetID string, targetKind string, command string) (string, error) {
	result, taskID, err := e.console.RunAutomationLine(targetID, command)
	if err != nil || targetKind != "beacon" {
		return result, err
	}
	result, _, err = e.beacons.AwaitBeaconTask(ctx, targetID, result, taskID)
	return result, err
}

type TargetProvider struct {
	rpc *rpc.Client
}

func NewTargetProvider(rpcClient *rpc.Client) *TargetProvider {
	return &TargetProvider{rpc: rpcClient}
}

func (p *TargetProvider) Connected() bool {
	return p.rpc.Connected()
}

func (p *TargetProvider) GetSessions(ctx context.Context) ([]automation.Target, error) {
	sessions, err := p.rpc.RPC.GetSessions(ctx, &commonpb.Empty{})
	if err != nil {
		return nil, err
	}
	p.rpc.PopulateSessions(sessions)
	targets := make([]automation.Target, 0, len(sessions.Sessions))
	for _, s := range sessions.Sessions {
		targets = append(targets, targetFromSession(s))
	}
	return targets, nil
}

func (p *TargetProvider) GetBeacons(ctx context.Context) ([]automation.Target, error) {
	beaconsResp, err := p.rpc.RPC.GetBeacons(ctx, &commonpb.Empty{})
	if err != nil {
		return nil, err
	}
	p.rpc.PopulateBeacons(beaconsResp)
	targets := make([]automation.Target, 0, len(beaconsResp.Beacons))
	for _, b := range beaconsResp.Beacons {
		targets = append(targets, targetFromBeacon(b))
	}
	return targets, nil
}

func (p *TargetProvider) FindTarget(ctx context.Context, targetID string) (automation.Target, error) {
	if sess := p.rpc.LookupSession(targetID); sess != nil {
		return targetFromSession(sess), nil
	}
	if beacon := p.rpc.LookupBeacon(targetID); beacon != nil {
		return targetFromBeacon(beacon), nil
	}
	sessions, sessionErr := p.rpc.RPC.GetSessions(ctx, &commonpb.Empty{})
	if sessionErr == nil {
		p.rpc.PopulateSessions(sessions)
		for _, s := range sessions.Sessions {
			if s.ID == targetID {
				return targetFromSession(s), nil
			}
		}
	}
	beaconsResp, beaconErr := p.rpc.RPC.GetBeacons(ctx, &commonpb.Empty{})
	if beaconErr == nil {
		p.rpc.PopulateBeacons(beaconsResp)
		for _, b := range beaconsResp.Beacons {
			if b.ID == targetID {
				return targetFromBeacon(b), nil
			}
		}
	}
	return automation.Target{}, fmt.Errorf("agent not found: %s", targetID)
}

type EventSource struct {
	rpc     *rpc.Client
	handler automation.EventHandler
	cancel  context.CancelFunc
}

func NewEventSource(rpcClient *rpc.Client) *EventSource {
	return &EventSource{rpc: rpcClient}
}

func (s *EventSource) Start(ctx context.Context, handler automation.EventHandler) {
	s.handler = handler
	if s.rpc != nil {
		s.stop()
		streamCtx, cancel := context.WithCancel(ctx)
		s.cancel = cancel
		s.rpc.StartEventStream(streamCtx, func(ev *clientpb.Event) {
			s.HandleSliverEvent(ev)
		})
	}
}

func (s *EventSource) Stop() {
	s.handler = nil
	s.stop()
}

func (s *EventSource) stop() {
	if s.cancel != nil {
		s.cancel()
		s.cancel = nil
	}
}

func (s *EventSource) HandleSliverEvent(ev *clientpb.Event) {
	if s.handler == nil {
		return
	}
	switch ev.EventType {
	case consts.SessionOpenedEvent:
		if ev.Session != nil {
			s.handler("session-connected", targetFromSession(ev.Session))
		}
	case consts.BeaconRegisteredEvent:
		beacon := &clientpb.Beacon{}
		if len(ev.Data) > 0 && proto.Unmarshal(ev.Data, beacon) == nil && beacon.ID != "" {
			s.handler("beacon-registered", targetFromBeacon(beacon))
		}
	}
}

func targetFromSession(session *clientpb.Session) automation.Target {
	return automation.Target{
		ID: session.ID, Name: session.Name, Hostname: session.Hostname,
		Username: session.Username, OS: session.OS, Arch: session.Arch, Kind: "session",
	}
}

func targetFromBeacon(beacon *clientpb.Beacon) automation.Target {
	return automation.Target{
		ID: beacon.ID, Name: beacon.Name, Hostname: beacon.Hostname,
		Username: beacon.Username, OS: beacon.OS, Arch: beacon.Arch, Kind: "beacon",
		LastCheckin: beacon.LastCheckin,
	}
}
