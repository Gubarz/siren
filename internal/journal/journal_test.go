package journal

import (
	"context"
	"testing"
)

func TestOverlayRoundTrip(t *testing.T) {
	ctx := WithContext(context.Background(), Overlay{
		ActorKind: "operator", Panel: "console", CommandLine: "ps",
		TargetID: "abc", CorrelationID: "corr-1",
	})
	o, ok := OverlayFrom(ctx)
	if !ok {
		t.Fatal("overlay missing")
	}
	if o.CommandLine != "ps" || o.CorrelationID != "corr-1" || o.ActorKind != "operator" {
		t.Fatalf("unexpected overlay: %+v", o)
	}
}

func TestWithContextMergesOverExisting(t *testing.T) {
	ctx := WithContext(context.Background(), Overlay{ActorKind: "automation", RuleID: "r1"})
	ctx = WithContext(ctx, Overlay{CommandLine: "whoami"})
	o, _ := OverlayFrom(ctx)
	if o.ActorKind != "automation" || o.RuleID != "r1" || o.CommandLine != "whoami" {
		t.Fatalf("merge lost fields: %+v", o)
	}
}

func TestOverlayFromMissing(t *testing.T) {
	if _, ok := OverlayFrom(context.Background()); ok {
		t.Fatal("expected no overlay")
	}
}

func TestApplyOverlayDefaultsActor(t *testing.T) {
	var e Entry
	e.ApplyOverlay(Overlay{CommandLine: "ls", CorrelationID: "c"})
	if e.ActorKind != "operator" || e.CommandLine != "ls" || e.CorrelationID != "c" {
		t.Fatalf("bad entry: %+v", e)
	}
}

func TestApplyOverlayNeverOverwritesSetFields(t *testing.T) {
	e := Entry{ActorKind: "integration"}
	e.ApplyOverlay(Overlay{ActorKind: "automation", RuleID: "r"})
	if e.ActorKind != "integration" || e.RuleID != "r" {
		t.Fatalf("bad entry: %+v", e)
	}
}

func TestClassifyVerbDropsPolls(t *testing.T) {
	for _, verb := range []string{
		"GetSessions", "GetBeacons", "GetJobs", "GetVersion",
		"GetBeaconTask", "GetBeaconTasks", "Events", "ClientLog", "TunnelData",
	} {
		if ClassifyVerb(verb) != VerbDrop {
			t.Fatalf("%s must be dropped", verb)
		}
	}
}

func TestClassifyVerbRecordsActions(t *testing.T) {
	for _, verb := range []string{"Ps", "Download", "Execute", "Kill", "Screenshot", "Generate"} {
		if ClassifyVerb(verb) != VerbRecord {
			t.Fatalf("%s must be recorded", verb)
		}
	}
}
