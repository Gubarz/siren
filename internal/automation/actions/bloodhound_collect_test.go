package actions

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"siren/internal/automation"
)

type fakeCollector struct {
	stages   []automation.CollectorProgress
	idx      int
	startErr error
	req      automation.CollectorRequest
	agentID  string
}

func (f *fakeCollector) StartCollection(ctx context.Context, agentID, agentKind, agentOS string, req automation.CollectorRequest) (string, error) {
	if f.startErr != nil {
		return "", f.startErr
	}
	f.req = req
	f.agentID = agentID
	return "run-1", nil
}

func (f *fakeCollector) CollectionState(ctx context.Context, id string) (automation.CollectorProgress, bool) {
	if f.idx >= len(f.stages) {
		if len(f.stages) == 0 {
			return automation.CollectorProgress{Stage: "collecting"}, true // stuck forever
		}
		return automation.CollectorProgress{Stage: "done"}, true
	}
	stage := f.stages[f.idx]
	f.idx++
	return stage, true
}

func runContext(t *testing.T, ctx context.Context, starter *fakeCollector) (*automation.RunContext, *strings.Builder) {
	t.Helper()
	var output strings.Builder
	rc := &automation.RunContext{
		Ctx:    ctx,
		Target: automation.Target{ID: "sess-1", Kind: "session", OS: "windows"},
		Rule:   automation.AutomationRule{},
		Action: automation.ActionSpec{Config: map[string]any{
			"collector":      "sharphound",
			"methods":        "Default,Session",
			"flags":          "--Stealth",
			"domain":         "corp.local",
			"timeoutMinutes": float64(15),
			"autoIngest":     true,
			"archiveLoot":    true,
		}},
		Deps: automation.ActionDeps{Collector: starter},
	}
	rc.Log = func(args ...any) {
		output.WriteString(fmt.Sprint(args...))
	}
	return rc, &output
}

func TestBloodHoundCollectCompletes(t *testing.T) {
	starter := &fakeCollector{stages: []automation.CollectorProgress{
		{Stage: "collecting"}, {Stage: "downloading"}, {Stage: "ingesting"},
	}}
	rc, output := runContext(t, context.Background(), starter)

	if err := BloodHoundCollect().Execute(rc); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if starter.agentID != "sess-1" {
		t.Fatalf("agentID = %q, want sess-1", starter.agentID)
	}
	if starter.req.Collector != "sharphound" || len(starter.req.Methods) != 2 || starter.req.Domain != "corp.local" {
		t.Fatalf("request = %+v", starter.req)
	}
	if !starter.req.Ingest || !starter.req.Loot || starter.req.TimeoutSeconds != 900 {
		t.Fatalf("request = %+v", starter.req)
	}
	if !strings.Contains(output.String(), "stage: ingesting") || !strings.Contains(output.String(), "collection started") {
		t.Fatalf("output = %q", output.String())
	}
}

func TestBloodHoundCollectSurfacesFailure(t *testing.T) {
	starter := &fakeCollector{stages: []automation.CollectorProgress{
		{Stage: "failed", Error: "collector exit status 1"},
	}}
	rc, _ := runContext(t, context.Background(), starter)

	err := BloodHoundCollect().Execute(rc)
	if err == nil || !strings.Contains(err.Error(), "exit status 1") {
		t.Fatalf("Execute = %v, want stage failure", err)
	}
}

func TestBloodHoundCollectStartError(t *testing.T) {
	starter := &fakeCollector{startErr: errors.New("not connected")}
	rc, _ := runContext(t, context.Background(), starter)

	if err := BloodHoundCollect().Execute(rc); !strings.Contains(err.Error(), "not connected") {
		t.Fatalf("Execute = %v, want start error", err)
	}
}

func TestBloodHoundCollectHonorsContextCancel(t *testing.T) {
	starter := &fakeCollector{} // empty stages: stuck in "collecting" until cancelled
	ctx, cancel := context.WithCancel(context.Background())
	rc, _ := runContext(t, ctx, starter)

	done := make(chan error, 1)
	go func() { done <- BloodHoundCollect().Execute(rc) }()
	time.Sleep(80 * time.Millisecond)
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("Execute = %v, want context.Canceled", err)
	}
}

func TestBloodHoundCollectWithoutCollector(t *testing.T) {
	rc, _ := runContext(t, context.Background(), &fakeCollector{})
	rc.Deps.Collector = nil
	if err := BloodHoundCollect().Execute(rc); err == nil {
		t.Fatal("Execute with nil collector should fail")
	}
}
