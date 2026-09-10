package rpc

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"google.golang.org/grpc"
)

// failingEventsRPC fails the event stream while every other call stays usable,
// which is what a transient stream failure looks like to the rest of the client.
type failingEventsRPC struct {
	rpcpb.SliverRPCClient
}

func (failingEventsRPC) Events(
	context.Context, *commonpb.Empty, ...grpc.CallOption,
) (rpcpb.SliverRPC_EventsClient, error) {
	return nil, errors.New("stream broke")
}

// A dropped event stream is not a dropped connection. The stream already has its
// own signal, the "stream-closed" event the frontend reconnects from, and
// Connected() answers whether there is a connection to call at all. Conflating
// the two made every guarded RPC refuse after a transient stream failure even
// though the connection was still usable.
func TestEventStreamFailureLeavesTheConnectionIntact(t *testing.T) {
	c := NewForTest(failingEventsRPC{})

	closed := make(chan struct{}, 1)
	c.StartEventStream(context.Background(), func(ev *clientpb.Event) {
		if ev.EventType == "stream-closed" {
			closed <- struct{}{}
		}
	})

	select {
	case <-closed:
	case <-time.After(5 * time.Second):
		t.Fatal("the stream failure was never reported")
	}

	if !c.Connected() {
		t.Error("Connected() = false after only the event stream failed")
	}
	if c.RPC() == nil {
		t.Error("RPC() = nil after only the event stream failed, so every guarded call would refuse")
	}
}
