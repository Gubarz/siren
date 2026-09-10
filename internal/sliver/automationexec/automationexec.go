package automationexec

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/gubarz/revils/store"

	"siren/internal/automation"
	"siren/internal/execctx"
	"siren/internal/sliver/beacons"
	"siren/internal/sliver/console"
	"siren/internal/sliver/rpc"
)

type Executor struct {
	console *console.Service
	beacons *beacons.Service
	store   *store.Store
}

func NewExecutor(con *console.Service, beac *beacons.Service) *Executor {
	return &Executor{console: con, beacons: beac}
}

func (e *Executor) SetStore(st *store.Store) {
	e.store = st
}

func (e *Executor) Execute(ctx context.Context, targetID string, targetKind string, command string) (string, error) {
	// The beacon-await path publishes task results from this context; hostname
	// is unknown here, so target attribution stays id/kind only.
	ctx = execctx.WithTarget(ctx, targetID, targetKind, "")
	result, taskID, err := e.console.RunAutomationLineContext(ctx, targetID, command)
	if err == nil && targetKind == "beacon" {
		result, _, err = e.beacons.AwaitBeaconTask(ctx, targetID, result, taskID)
	}
	writeEvidence(e.store, ctx, targetID, targetKind, result, err)
	return result, err
}

// writeEvidence records the outcome of an automation step. It is a no-op
// outside an automation run (no run id in context) or without a store.
func writeEvidence(st *store.Store, ctx context.Context, targetID, targetKind string, result string, err error) {
	if st == nil {
		return
	}
	runID, stageID := execctx.Run(ctx)
	if runID == "" {
		return
	}
	status := store.StatusVerified
	errText := ""
	if err != nil {
		status = store.StatusFailed
		errText = err.Error()
	}
	body, merr := json.Marshal(struct {
		Step        string `json:"step"`
		Error       string `json:"error"`
		OutputBytes int    `json:"output_bytes"`
	}{Step: "command", Error: errText, OutputBytes: len(result)})
	if merr != nil {
		body = []byte(`{"step":"command"}`)
	}
	rec := store.Record{
		Kind: store.KindEvidence, Status: status,
		RunID: runID, StageID: stageID, Method: "automation.execute",
		JSON: string(body),
	}
	if targetKind == "beacon" {
		rec.BeaconID = targetID
	} else {
		rec.SessionID = targetID
	}
	if _, werr := st.Write(rec); werr != nil {
		slog.Error("automation: evidence write failed", "run", runID, "error", werr)
	}
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
	return p.loadTargets(ctx, false)
}

func (p *TargetProvider) GetBeacons(ctx context.Context) ([]automation.Target, error) {
	return p.loadTargets(ctx, true)
}

func (p *TargetProvider) loadTargets(ctx context.Context, beacons bool) ([]automation.Target, error) {
	if beacons {
		resp, err := p.rpc.RPC.GetBeacons(ctx, &commonpb.Empty{})
		if err != nil {
			return nil, err
		}
		p.rpc.PopulateBeacons(resp)
		return collectTargets(resp.Beacons, targetFromBeacon), nil
	}
	resp, err := p.rpc.RPC.GetSessions(ctx, &commonpb.Empty{})
	if err != nil {
		return nil, err
	}
	p.rpc.PopulateSessions(resp)
	return collectTargets(resp.Sessions, targetFromSession), nil
}

// collectTargets preserves the non-nil empty slice callers expect when the
// server reports no agents.
func collectTargets[T any](items []T, convert func(T) automation.Target) []automation.Target {
	targets := make([]automation.Target, 0, len(items))
	for _, item := range items {
		targets = append(targets, convert(item))
	}
	return targets
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
