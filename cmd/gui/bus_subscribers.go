package gui

import (
	"encoding/json"
	"log"
	"strings"
	"time"

	consts "github.com/bishopfox/sliver/client/constants"

	"siren/internal/bus"
	knownagents "siren/internal/localstate/agents"
	"siren/internal/localstate/events"
)

func (a *App) startBusSubscribers() {
	a.Bus.Subscribe(nil, a.frontendBusSubscriber)
	a.Bus.Subscribe(nil, a.eventsStoreBusSubscriber)
}

func (a *App) frontendBusSubscriber(ev bus.Event) {
	switch {
	case ev.Type == "sliver.stream-closed":
		a.RPC.InvalidateAgentCache()
		a.Console.ResetConsole()
		a.bridge.Emit("sliver-event", map[string]interface{}{"type": "stream-closed"})
	case strings.HasPrefix(ev.Type, "sliver."):
		handleSliverEvent(sliverEventDeps{
			known:      a.KnownAgents,
			invalidate: a.RPC.InvalidateAgentCache,
			emit:       a.bridge.Emit,
		}, ev)
	case strings.HasPrefix(ev.Type, "gui."):
		if payload, ok := ev.Payload.(map[string]interface{}); ok {
			p := copyPayload(payload)
			p["type"] = ev.Type
			a.bridge.Emit("gui-event", p)
		}
	case strings.HasPrefix(ev.Type, "bloodhound."):
		raw, err := json.Marshal(ev.Payload)
		if err != nil {
			return
		}
		var payload any
		if err := json.Unmarshal(raw, &payload); err != nil {
			return
		}
		a.bridge.Emit("bloodhound-event", map[string]interface{}{
			"type":    ev.Type,
			"payload": payload,
		})
	}
}

// sliverEventDeps bundles the side effects a sliver bus event triggers.
// Extracted so tests can drive the handler with a real known-agent service
// and a recording emitter instead of a full App.
type sliverEventDeps struct {
	known      *knownagents.Service
	invalidate func()
	emit       func(name string, payload any)
}

// handleSliverEvent applies known-agent bookkeeping before notifying the
// frontend, so a refresh triggered by the emitted event cannot read
// pre-update state.
func handleSliverEvent(deps sliverEventDeps, ev bus.Event) {
	payload, _ := ev.Payload.(map[string]interface{})
	invalidate := deps.invalidate
	if invalidate == nil {
		invalidate = func() {}
	}
	switch ev.Type {
	case "sliver." + consts.SessionOpenedEvent:
		if record, ok := knownAgentFromEventPayload(payload); ok && deps.known != nil {
			if err := deps.known.Observe([]knownagents.Record{record}); err != nil {
				log.Printf("bus: observe opened session: %v", err)
			}
		}
		invalidate()
	case "sliver." + consts.SessionClosedEvent:
		if id := eventSessionID(payload); id != "" && deps.known != nil {
			if err := deps.known.MarkLost(id); err != nil {
				log.Printf("bus: mark session lost: %v", err)
			}
		}
		invalidate()
	case "sliver." + consts.BeaconRegisteredEvent:
		invalidate()
	}
	if payload == nil || deps.emit == nil {
		return
	}
	p := copyPayload(payload)
	p["type"] = strings.TrimPrefix(ev.Type, "sliver.")
	deps.emit("sliver-event", p)
}

func copyPayload(src map[string]interface{}) map[string]interface{} {
	dst := make(map[string]interface{}, len(src)+1)
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func (a *App) eventsStoreBusSubscriber(ev bus.Event) {
	if ev.Type == "sliver.stream-closed" {
		return
	}
	payload, ok := ev.Payload.(map[string]interface{})
	if !ok {
		return
	}
	se := events.StoredEvent{
		Type:      strings.TrimPrefix(ev.Type, "sliver."),
		Time:      time.Now().UnixMilli(),
		SessionID: payloadString(payload, "sessionID"),
		Hostname:  payloadString(payload, "hostname"),
		Username:  payloadString(payload, "username"),
		Job:       payloadString(payload, "job"),
		Data:      payloadString(payload, "data"),
	}
	a.Events.Append(se)
}

func payloadString(payload map[string]interface{}, key string) string {
	if v, ok := payload[key].(string); ok {
		return v
	}
	return ""
}

// knownAgentFromEventPayload builds a known-agent record from a flattened
// sliver session event. Events without a session ID are ignored.
func knownAgentFromEventPayload(payload map[string]interface{}) (knownagents.Record, bool) {
	id := eventSessionID(payload)
	if id == "" {
		return knownagents.Record{}, false
	}
	return knownagents.Record{
		ID:            id,
		Kind:          "session",
		Name:          payloadString(payload, "name"),
		Hostname:      payloadString(payload, "hostname"),
		Username:      payloadString(payload, "username"),
		OS:            payloadString(payload, "os"),
		Arch:          payloadString(payload, "arch"),
		RemoteAddress: payloadString(payload, "remoteAddress"),
		Transport:     payloadString(payload, "transport"),
	}, true
}

func eventSessionID(payload map[string]interface{}) string {
	return payloadString(payload, "sessionID")
}
