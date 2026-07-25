package triggers

import (
	"sliver-gui/internal/automation"
	"sliver-gui/internal/bus"
)

func SessionConnected(b bus.Bus) automation.Trigger {
	return BusEvent("session-connected", "sliver.session-opened", nil, b, targetMapper("session"))
}

func BeaconRegistered(b bus.Bus) automation.Trigger {
	return BusEvent("beacon-registered", "sliver.beacon-registered", nil, b, targetMapper("beacon"))
}

func BeaconCheckin(b bus.Bus) automation.Trigger {
	return BusEvent("beacon-checkin", "sliver.beacon-checkin", nil, b, targetMapper("beacon"))
}

func targetMapper(kind string) MapFunc {
	return func(ev bus.Event, _ map[string]any) (automation.FireEvent, bool) {
		payload, ok := ev.Payload.(map[string]any)
		if !ok {
			return automation.FireEvent{}, false
		}
		target := &automation.Target{
			ID:       str(payload, idKey(kind)),
			Name:     str(payload, "name"),
			Hostname: str(payload, "hostname"),
			Username: str(payload, "username"),
			OS:       str(payload, "os"),
			Arch:     str(payload, "arch"),
			Kind:     kind,
		}
		if target.ID == "" {
			return automation.FireEvent{}, false
		}
		return automation.FireEvent{Target: target, Data: payload}, true
	}
}

func idKey(kind string) string {
	if kind == "beacon" {
		return "beaconID"
	}
	return "sessionID"
}

func str(payload map[string]any, key string) string {
	if v, ok := payload[key].(string); ok {
		return v
	}
	return ""
}
