package gui

import (
	"encoding/json"
	"strings"
	"time"

	consts "github.com/bishopfox/sliver/client/constants"

	"siren/internal/bus"
	"siren/internal/journal"
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
		if payload, ok := ev.Payload.(map[string]interface{}); ok {
			p := copyPayload(payload)
			p["type"] = strings.TrimPrefix(ev.Type, "sliver.")
			a.bridge.Emit("sliver-event", p)
		}
		switch ev.Type {
		case "sliver." + consts.SessionOpenedEvent,
			"sliver." + consts.SessionClosedEvent,
			"sliver." + consts.BeaconRegisteredEvent:
			a.RPC.InvalidateAgentCache()
		}
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
	case ev.Type == "journal.action-recorded":
		if entry, ok := ev.Payload.(journal.Entry); ok {
			a.bridge.Emit("journal-event", entryToMap(entry))
		}
	}
}

func copyPayload(src map[string]interface{}) map[string]interface{} {
	dst := make(map[string]interface{}, len(src)+1)
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func entryToMap(e journal.Entry) map[string]interface{} {
	return map[string]interface{}{
		"type":           "action-recorded",
		"id":             e.ID,
		"time":           e.Time,
		"connection_id":  e.ConnectionID,
		"actor_kind":     e.ActorKind,
		"rule_id":        e.RuleID,
		"rule_name":      e.RuleName,
		"verb":           e.Verb,
		"command_line":   e.CommandLine,
		"target_id":      e.TargetID,
		"target_kind":    e.TargetKind,
		"hostname":       e.Hostname,
		"panel":          e.Panel,
		"status":         e.Status,
		"err":            e.Err,
		"duration_ms":    e.DurationMs,
		"correlation_id": e.CorrelationID,
	}
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
