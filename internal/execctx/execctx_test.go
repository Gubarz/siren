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

func TestSetCurrentRestoresPrevious(t *testing.T) {
	if got := Current(); got != (Snapshot{}) {
		t.Fatalf("initial current = %+v", got)
	}
	restore := SetCurrent(Snapshot{
		RunID: "run-1", StageID: "commands#0",
		TargetID: "sess-1", TargetKind: "session", Hostname: "host-a",
	})
	got := Current()
	if got.RunID != "run-1" || got.StageID != "commands#0" || got.TargetID != "sess-1" ||
		got.TargetKind != "session" || got.Hostname != "host-a" {
		t.Fatalf("current = %+v", got)
	}
	restore()
	if got := Current(); got != (Snapshot{}) {
		t.Fatalf("after restore = %+v", got)
	}
}
