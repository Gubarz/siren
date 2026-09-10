package clientlog

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"google.golang.org/grpc/metadata"
)

type failingStream struct {
	ctx context.Context
}

func (f *failingStream) Send(*clientpb.ClientLogData) error { return errors.New("stream broken") }
func (f *failingStream) CloseAndRecv() (*commonpb.Empty, error) {
	return &commonpb.Empty{}, nil
}
func (f *failingStream) Header() (metadata.MD, error) { return nil, nil }
func (f *failingStream) Trailer() metadata.MD         { return nil }
func (f *failingStream) CloseSend() error             { return nil }
func (f *failingStream) Context() context.Context     { return f.ctx }
func (f *failingStream) SendMsg(any) error            { return nil }
func (f *failingStream) RecvMsg(any) error            { return nil }

// A pump that dies used to leave running set, so every later Start returned
// early and the client log stayed dead until a reconnect. Its context was also
// never cancelled, leaving the stream's capture record open.
func TestFailedPumpReleasesTheService(t *testing.T) {
	parent, parentCancel := context.WithCancel(context.Background())
	defer parentCancel()
	// Start derives the pump's context from the caller's, so the pump may only
	// cancel its own.
	pumpCtx, pumpCancel := context.WithCancel(parent)

	svc := &Service{running: true, cancel: pumpCancel, queue: make(chan []byte, 4)}
	svc.queue <- []byte("a line the stream will reject")

	done := make(chan struct{})
	go func() {
		defer close(done)
		svc.pump(pumpCtx, &failingStream{ctx: pumpCtx})
	}()

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("the pump never returned")
	}

	if svc.running {
		t.Fatal("running is still set after the pump failed")
	}
	if svc.cancel != nil {
		t.Fatal("cancel was not cleared after the pump failed")
	}
	if svc.queue != nil {
		t.Fatal("queue was not cleared after the pump failed")
	}
	if parent.Err() != nil {
		t.Fatal("pump cancelled the caller's context, not just its own")
	}
}
