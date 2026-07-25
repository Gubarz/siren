package automationexec

import (
	"testing"
	"time"

	"sliver-gui/internal/automation"
	"sliver-gui/internal/bus"
)

func TestHandleBusEventSessionOpened(t *testing.T) {
	fired := make(chan automation.Target, 1)
	src := &EventSource{handler: func(trigger string, target automation.Target) {
		if trigger == "session-connected" {
			fired <- target
		}
	}}
	src.HandleBusEvent(bus.Event{
		Type: "sliver.session-opened",
		Payload: map[string]any{
			"type": "session-opened", "sessionID": "s1", "name": "n",
			"hostname": "h", "username": "u", "os": "linux", "arch": "amd64",
		},
	})
	select {
	case target := <-fired:
		if target.ID != "s1" || target.Kind != "session" || target.OS != "linux" {
			t.Fatalf("target: %+v", target)
		}
	case <-time.After(time.Second):
		t.Fatal("no trigger fired")
	}
}

func TestHandleBusEventBeaconRegistered(t *testing.T) {
	fired := make(chan automation.Target, 1)
	src := &EventSource{handler: func(trigger string, target automation.Target) {
		if trigger == "beacon-registered" {
			fired <- target
		}
	}}
	src.HandleBusEvent(bus.Event{
		Type: "sliver.beacon-registered",
		Payload: map[string]any{
			"type": "beacon-registered", "beaconID": "b1", "hostname": "h",
			"username": "u", "os": "windows", "arch": "amd64",
		},
	})
	select {
	case target := <-fired:
		if target.ID != "b1" || target.Kind != "beacon" {
			t.Fatalf("target: %+v", target)
		}
	case <-time.After(time.Second):
		t.Fatal("no trigger fired")
	}
}

func TestHandleBusEventIgnoresOthersAndNilHandler(t *testing.T) {
	src := &EventSource{}
	src.HandleBusEvent(bus.Event{Type: "sliver.job-started"}) // no handler — must not panic
}
