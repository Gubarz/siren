package loot

import (
	"testing"
	"time"

	"siren/internal/bus"
	"siren/internal/sliver/rpc"
)

func TestServicePublishPayload(t *testing.T) {
	b := bus.New()
	captured := make(chan bus.Event, 1)
	b.Subscribe([]string{"gui.loot-added"}, func(e bus.Event) {
		select {
		case captured <- e:
		default:
		}
	})

	s := &Service{rpc: rpc.NewClient(), bus: b}
	s.publish("gui.loot-added", map[string]any{
		"type":     "loot-added",
		"lootID":   "test-id",
		"name":     "test",
		"fileType": int32(0),
		"size":     int64(4),
	})

	select {
	case ev := <-captured:
		if ev.Type != "gui.loot-added" {
			t.Fatalf("expected event type gui.loot-added, got %s", ev.Type)
		}
		payload, ok := ev.Payload.(map[string]any)
		if !ok {
			t.Fatal("payload is not map[string]any")
		}
		if payload["type"] != "loot-added" {
			t.Fatalf("expected payload type loot-added, got %v", payload["type"])
		}
		if payload["lootID"] != "test-id" {
			t.Fatalf("expected lootID test-id, got %v", payload["lootID"])
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for bus event")
	}
}
