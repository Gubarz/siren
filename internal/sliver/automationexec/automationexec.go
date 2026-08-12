package automationexec

import (
	"context"
	"fmt"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"

	"siren/internal/automation"
	"siren/internal/sliver/beacons"
	"siren/internal/sliver/console"
	"siren/internal/sliver/rpc"
)

type Executor struct {
	console *console.Service
	beacons *beacons.Service
}

func NewExecutor(con *console.Service, beac *beacons.Service) *Executor {
	return &Executor{console: con, beacons: beac}
}

func (e *Executor) Execute(ctx context.Context, targetID string, targetKind string, command string) (string, error) {
	result, taskID, err := e.console.RunAutomationLineContext(ctx, targetID, command)
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
