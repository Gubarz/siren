package execctx

import (
	"context"
	"testing"
)

func TestRunRoundtrip(t *testing.T) {
	ctx := WithRun(context.Background(), "run-1", "commands#0")
	runID, stageID := Run(ctx)
	if runID != "run-1" || stageID != "commands#0" {
		t.Fatalf("got %q/%q", runID, stageID)
	}
	if runID, stageID := Run(context.Background()); runID != "" || stageID != "" {
		t.Fatalf("empty ctx returned %q/%q", runID, stageID)
	}
}

func TestTargetRoundtrip(t *testing.T) {
	ctx := WithTarget(context.Background(), "sess-1", "session", "host-a")
	id, kind, host := Target(ctx)
	if id != "sess-1" || kind != "session" || host != "host-a" {
		t.Fatalf("got %q/%q/%q", id, kind, host)
	}
}
