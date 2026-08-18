package rpc

import (
	"siren/internal/bus"
)

// Emitter encapsulates event bus publishing for Sliver RPC services.
type Emitter struct {
	rpc *Client
	bus bus.Bus
}

// NewEmitter returns an Emitter associated with the given RPC client.
func NewEmitter(client *Client) Emitter {
	return Emitter{rpc: client}
}

// SetBus assigns the event bus for publishing.
func (e *Emitter) SetBus(b bus.Bus) {
	e.bus = b
}

// Publish emits a GUI event to the bus if configured.
func (e *Emitter) Publish(eventType string, payload map[string]any) {
	if e.bus == nil {
		return
	}
	var connID string
	if e.rpc != nil {
		connID = e.rpc.ConnectionID()
	}
	e.bus.Publish(bus.Event{
		Type:         eventType,
		Source:       "gui",
		ConnectionID: connID,
		Payload:      payload,
	})
}
