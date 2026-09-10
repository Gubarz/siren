package rpc

import "testing"

// The connection fields are unexported so a reader cannot race a reconnect.
// These are the accessors every other package goes through.
func TestAccessorsReturnNothingBeforeConnect(t *testing.T) {
	c := NewClient()

	if c.RPC() != nil {
		t.Fatal("RPC() on a fresh client is non-nil")
	}
	if c.Config() != nil {
		t.Fatal("Config() on a fresh client is non-nil")
	}
	if c.Conn() != nil {
		t.Fatal("Conn() on a fresh client is non-nil")
	}
	if got := c.ConnectionID(); got != "" {
		t.Fatalf("ConnectionID() = %q, want empty", got)
	}
	if c.IsConnectedTo("") {
		t.Fatal("IsConnectedTo() = true on a fresh client")
	}
}

func TestNewForTestMarksTheClientConnected(t *testing.T) {
	c := NewForTest(&fakeSliverRPC{})

	if !c.Connected() {
		t.Fatal("Connected() = false after NewForTest")
	}
	if c.RPC() == nil {
		t.Fatal("RPC() = nil after NewForTest")
	}
}
