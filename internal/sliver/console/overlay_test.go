package console

import (
	"context"
	"testing"

	"siren/internal/journal"
)

func TestWithCommandOverlayDefaultsOperator(t *testing.T) {
	ctx := withCommandOverlay(context.Background(), "t1", "session", "host-1", "ps")
	o, ok := journal.OverlayFrom(ctx)
	if !ok {
		t.Fatal("no overlay")
	}
	if o.ActorKind != "operator" || o.Panel != "console" || o.CommandLine != "ps" {
		t.Fatalf("overlay: %+v", o)
	}
	if o.TargetID != "t1" || o.TargetKind != "session" || o.CorrelationID == "" {
		t.Fatalf("overlay: %+v", o)
	}
}

func TestWithCommandOverlayPreservesAutomationActor(t *testing.T) {
	parent := journal.WithContext(context.Background(), journal.Overlay{
		ActorKind: "automation", RuleID: "r1", CorrelationID: "run-1",
	})
	ctx := withCommandOverlay(parent, "t2", "beacon", "host-2", "ls")
	o, _ := journal.OverlayFrom(ctx)
	if o.ActorKind != "automation" || o.RuleID != "r1" || o.CorrelationID != "run-1" {
		t.Fatalf("automation overlay clobbered: %+v", o)
	}
	if o.CommandLine != "ls" || o.TargetID != "t2" {
		t.Fatalf("command fields missing: %+v", o)
	}
}
