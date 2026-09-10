package gui

import (
	"testing"

	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/protobuf/clientpb"
)

func TestSliverEventPayloadIncludesSessionRemoteAddressAndTransport(t *testing.T) {
	payload := sliverEventPayload(&clientpb.Event{
		EventType: consts.SessionOpenedEvent,
		Session: &clientpb.Session{
			ID:            "sess-a",
			RemoteAddress: "10.0.0.1:443",
			Transport:     "mtls",
		},
	})
	if got := payload["sessionID"]; got != "sess-a" {
		t.Fatalf("sessionID = %v, want sess-a", got)
	}
	if got := payload["remoteAddress"]; got != "10.0.0.1:443" {
		t.Fatalf("remoteAddress = %v, want 10.0.0.1:443", got)
	}
	if got := payload["transport"]; got != "mtls" {
		t.Fatalf("transport = %v, want mtls", got)
	}
}
