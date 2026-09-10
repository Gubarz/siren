package gui

import (
	"encoding/json"
	"strings"
	"time"

	consts "github.com/bishopfox/sliver/client/constants"

	"siren/internal/bus"
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
	}
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
