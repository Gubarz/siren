package triggers

import (
	"context"
	"testing"
	"time"

	"siren/internal/automation"
	automationevents "siren/internal/automation/events"
	"siren/internal/bus"
)

func TestTaskFinishFiresOnBeaconTaskResult(t *testing.T) {
	b := bus.New()
	tr := TaskFinish(b)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	fired := make(chan automation.FireEvent, 8)
	go func() {
		_ = tr.Arm(ctx, map[string]any{"verb": "BeaconTaskResult"}, func(fe automation.FireEvent) { fired <- fe })
	}()

	ev := bus.Event{Type: "beacon.task-result", Payload: automationevents.TaskResult{
		Verb: "BeaconTaskResult", TargetID: "b1", TargetKind: "beacon", Hostname: "host-1", Status: "ok",
	}}
	deadline := time.Now().Add(2 * time.Second)
	for {
		b.Publish(ev)
		select {
		case fe := <-fired:
			if fe.Target == nil || fe.Target.ID != "b1" || fe.Target.Kind != "beacon" || fe.Target.Hostname != "host-1" {
				t.Fatalf("target: %+v", fe.Target)
			}
			entry, ok := fe.Data["entry"].(automationevents.TaskResult)
			if !ok || entry.Status != "ok" {
				t.Fatalf("data: %+v", fe.Data)
			}
			return
		case <-time.After(5 * time.Millisecond):
		}
		if time.Now().After(deadline) {
			t.Fatal("task-finish did not fire")
		}
	}
}

func TestTaskFinishFiltersVerbAndTarget(t *testing.T) {
	b := bus.New()
	tr := TaskFinish(b)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	fired := make(chan automation.FireEvent, 16)
	go func() {
		_ = tr.Arm(ctx, map[string]any{"verb": "Wanted", "targetID": "b2"}, func(fe automation.FireEvent) { fired <- fe })
	}()

	matching := func(hostname string) bus.Event {
		return bus.Event{Type: "beacon.task-result", Payload: automationevents.TaskResult{
			Verb: "Wanted", TargetID: "b2", TargetKind: "beacon", Hostname: hostname,
		}}
	}
	next := func() automationevents.TaskResult {
		t.Helper()
		select {
		case fe := <-fired:
			entry, ok := fe.Data["entry"].(automationevents.TaskResult)
			if !ok {
				t.Fatalf("data: %+v", fe.Data)
			}
			return entry
		case <-time.After(2 * time.Second):
			t.Fatal("task-finish did not fire")
			return automationevents.TaskResult{}
		}
	}

	deadline := time.Now().Add(2 * time.Second)
	for len(fired) == 0 {
		b.Publish(matching("warmup"))
		time.Sleep(2 * time.Millisecond)
		if time.Now().After(deadline) {
			t.Fatal("task-finish did not fire")
		}
	}

	// FIFO delivery makes the sync event a barrier: once observed, every
	// earlier publish has passed through the handler.
	b.Publish(matching("sync"))
	for next().Hostname != "sync" {
	}

	// Non-matching publishes precede the sentinel, so an event that wrongly
	// passed the filter would be observed before it.
	b.Publish(bus.Event{Type: "beacon.task-result", Payload: automationevents.TaskResult{Verb: "Other", TargetID: "b2"}})
	b.Publish(bus.Event{Type: "beacon.task-result", Payload: automationevents.TaskResult{Verb: "Wanted", TargetID: "b9"}})
	b.Publish(bus.Event{Type: "beacon.task-result", Payload: "not a task result"})
	b.Publish(matching("sentinel"))
	entry := next()
	if entry.Hostname != "sentinel" {
		t.Fatalf("filtered event fired: %+v", entry)
	}
	if entry.Verb != "Wanted" || entry.TargetID != "b2" || entry.TargetKind != "beacon" {
		t.Fatalf("matching payload = %+v", entry)
	}
}
