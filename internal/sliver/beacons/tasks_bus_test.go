package beacons

import (
	"context"
	"errors"
	"testing"

	automationevents "siren/internal/automation/events"
	"siren/internal/bus"
	"siren/internal/execctx"
)

type recordingBus struct {
	events []bus.Event
}

func (b *recordingBus) Publish(ev bus.Event) {
	b.events = append(b.events, ev)
}

func (b *recordingBus) Subscribe(_ []string, _ bus.Handler) func() {
	return func() {}
}

func TestPublishBeaconTaskResultNilSafe(t *testing.T) {
	s := &Service{}
	s.publishBeaconTaskResult(context.Background(), "b1", nil)
}

func TestPublishBeaconTaskResultPublishesStatus(t *testing.T) {
	b := &recordingBus{}
	s := &Service{bus: b}
	ctx := execctx.WithTarget(context.Background(), "sess-1", "session", "host-1")

	s.publishBeaconTaskResult(ctx, "b1", nil)
	s.publishBeaconTaskResult(ctx, "b2", errors.New("boom"))

	if len(b.events) != 2 {
		t.Fatalf("events: %d", len(b.events))
	}
	if b.events[0].Type != "beacon.task-result" {
		t.Fatalf("type: %q", b.events[0].Type)
	}
	first, isResult := b.events[0].Payload.(automationevents.TaskResult)
	if !isResult {
		t.Fatalf("payload: %T", b.events[0].Payload)
	}
	if first.Verb != "BeaconTaskResult" || first.TargetID != "b1" || first.TargetKind != "beacon" {
		t.Fatalf("result: %+v", first)
	}
	if first.Status != "ok" || first.Error != "" {
		t.Fatalf("result: %+v", first)
	}
	if first.Hostname != "host-1" {
		t.Fatalf("hostname: %q", first.Hostname)
	}
	failed, _ := b.events[1].Payload.(automationevents.TaskResult)
	if failed.TargetID != "b2" || failed.Status != "error" || failed.Error != "boom" {
		t.Fatalf("result: %+v", failed)
	}
}
