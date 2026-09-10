package console

import (
	"context"
	"testing"

	"siren/internal/execctx"
)

func TestWithCommandOverlaySetsTarget(t *testing.T) {
	ctx := withCommandOverlay(context.Background(), "t1", "session", "host-1", "ps")
	id, kind, hostname := execctx.Target(ctx)
	if id != "t1" || kind != "session" || hostname != "host-1" {
		t.Fatalf("target: %q %q %q", id, kind, hostname)
	}
}

func TestWithCommandOverlayPreservesRunContext(t *testing.T) {
	parent := execctx.WithRun(context.Background(), "run-1", "commands#0")
	ctx := withCommandOverlay(parent, "t2", "beacon", "host-2", "ls")
	if runID, stageID := execctx.Run(ctx); runID != "run-1" || stageID != "commands#0" {
		t.Fatalf("run context clobbered: %q %q", runID, stageID)
	}
	id, kind, hostname := execctx.Target(ctx)
	if id != "t2" || kind != "beacon" || hostname != "host-2" {
		t.Fatalf("target: %q %q %q", id, kind, hostname)
	}
}
