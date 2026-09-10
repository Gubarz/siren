package gui

import (
	"reflect"
	"testing"
	"time"

	consts "github.com/bishopfox/sliver/client/constants"

	"siren/internal/bus"
	knownagents "siren/internal/localstate/agents"
)

func TestKnownAgentFromEventPayload(t *testing.T) {
	payload := map[string]interface{}{
		"sessionID":     "sess-a",
		"name":          "name-a",
		"hostname":      "host-a",
		"username":      "user-a",
		"os":            "linux",
		"arch":          "amd64",
		"remoteAddress": "10.0.0.1:443",
		"transport":     "mtls",
	}
	got, ok := knownAgentFromEventPayload(payload)
	if !ok {
		t.Fatal("knownAgentFromEventPayload() ok = false, want true")
	}
	want := knownagents.Record{
		ID:            "sess-a",
		Kind:          "session",
		Name:          "name-a",
		Hostname:      "host-a",
		Username:      "user-a",
		OS:            "linux",
		Arch:          "amd64",
		RemoteAddress: "10.0.0.1:443",
		Transport:     "mtls",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("knownAgentFromEventPayload() = %+v, want %+v", got, want)
	}
}

func TestKnownAgentFromEventPayloadRequiresSessionID(t *testing.T) {
	for name, payload := range map[string]map[string]interface{}{
		"nil payload":          nil,
		"missing sessionID":    {"name": "name-a"},
		"empty sessionID":      {"sessionID": ""},
		"non-string sessionID": {"sessionID": 42},
	} {
		t.Run(name, func(t *testing.T) {
			if _, ok := knownAgentFromEventPayload(payload); ok {
				t.Fatalf("knownAgentFromEventPayload(%v) ok = true, want false", payload)
			}
		})
	}
}

// emission captures the known-agent state at the moment the frontend event
// is emitted, so the test can prove bookkeeping happened first.
type emission struct {
	payload map[string]interface{}
	status  string
}

func recvEmission(t *testing.T, ch <-chan emission) emission {
	t.Helper()
	select {
	case e := <-ch:
		return e
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for frontend emission")
		return emission{}
	}
}

func TestHandleSliverEventTracksSessionLifecycleBeforeEmitting(t *testing.T) {
	known := knownagents.New(t.TempDir())
	emissions := make(chan emission, 4)
	deps := sliverEventDeps{
		known: known,
		emit: func(_ string, payload any) {
			p, _ := payload.(map[string]interface{})
			list := known.List()
			status := ""
			if len(list) > 0 {
				status = list[0].Status
			}
			emissions <- emission{payload: p, status: status}
		},
	}

	b := bus.New()
	unsub := b.Subscribe(nil, func(ev bus.Event) { handleSliverEvent(deps, ev) })
	defer unsub()

	b.Publish(bus.Event{
		Type: "sliver." + consts.SessionOpenedEvent,
		Payload: map[string]interface{}{
			"sessionID":     "sess-a",
			"name":          "name-a",
			"hostname":      "host-a",
			"username":      "user-a",
			"os":            "linux",
			"arch":          "amd64",
			"remoteAddress": "10.0.0.1:443",
			"transport":     "mtls",
		},
	})
	opened := recvEmission(t, emissions)
	if opened.payload["type"] != "session-connected" {
		t.Fatalf("opened payload type = %v, want session-connected", opened.payload["type"])
	}
	if opened.status != "active" {
		t.Fatalf("status at emit = %q, want active (observed before emit)", opened.status)
	}

	b.Publish(bus.Event{
		Type:    "sliver." + consts.SessionClosedEvent,
		Payload: map[string]interface{}{"sessionID": "sess-a"},
	})
	closed := recvEmission(t, emissions)
	if closed.payload["type"] != "session-disconnected" {
		t.Fatalf("closed payload type = %v, want session-disconnected", closed.payload["type"])
	}
	if closed.status != "lost" {
		t.Fatalf("status at emit = %q, want lost (marked before emit)", closed.status)
	}

	list := known.List()
	if len(list) != 1 || list[0].ID != "sess-a" || list[0].Status != "lost" {
		t.Fatalf("List() = %+v, want sess-a lost", list)
	}
}

func TestEventSessionID(t *testing.T) {
	if got := eventSessionID(map[string]interface{}{"sessionID": "sess-a"}); got != "sess-a" {
		t.Fatalf("eventSessionID() = %q, want sess-a", got)
	}
	for name, payload := range map[string]map[string]interface{}{
		"nil":     nil,
		"missing": {},
		"blank":   {"sessionID": ""},
	} {
		t.Run(name, func(t *testing.T) {
			if got := eventSessionID(payload); got != "" {
				t.Fatalf("eventSessionID() = %q, want empty", got)
			}
		})
	}
}
