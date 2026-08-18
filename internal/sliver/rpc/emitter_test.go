package rpc

import (
	"testing"

	"siren/internal/bus"
)

type mockBus struct {
	events []bus.Event
}

func (m *mockBus) Publish(e bus.Event) {
	m.events = append(m.events, e)
}

func (m *mockBus) Subscribe(_ []string, _ bus.Handler) func() {
	return func() {}
}

func TestEmitter_Publish(t *testing.T) {
	// Nil bus should not panic
	emitter := NewEmitter(nil)
	emitter.Publish("test.event", map[string]any{"foo": "bar"})

	b := &mockBus{}
	emitter.SetBus(b)

	emitter.Publish("gui.test-event", map[string]any{"count": 1})
	if len(b.events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(b.events))
	}
	ev := b.events[0]
	if ev.Type != "gui.test-event" || ev.Source != "gui" {
		t.Fatalf("unexpected event: %+v", ev)
	}
	payload, ok := ev.Payload.(map[string]any)
	if !ok || payload["count"] != 1 {
		t.Fatalf("unexpected payload: %+v", ev.Payload)
	}
}
