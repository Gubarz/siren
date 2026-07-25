package gui

import (
	"strings"
	"time"

	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/wailsapp/wails/v2/pkg/runtime"

	"sliver-gui/internal/bus"
	"sliver-gui/internal/localstate/events"
)

func (a *App) startBusSubscribers() {
	a.Bus.Subscribe(nil, a.frontendBusSubscriber)
	a.Bus.Subscribe(nil, a.eventsStoreBusSubscriber)
}

func (a *App) frontendBusSubscriber(ev bus.Event) {
	payload, ok := ev.Payload.(map[string]interface{})
	if !ok {
		return
	}
	if ev.Type == "sliver.stream-closed" {
		a.RPC.InvalidateAgentCache()
		a.Console.ResetConsole()
		runtime.EventsEmit(a.ctx, "sliver-event", map[string]interface{}{"type": "stream-closed"})
		return
	}
	switch ev.Type {
	case "sliver." + consts.SessionOpenedEvent,
		"sliver." + consts.SessionClosedEvent,
		"sliver." + consts.BeaconRegisteredEvent:
		a.RPC.InvalidateAgentCache()
	}
	runtime.EventsEmit(a.ctx, "sliver-event", payload)
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
