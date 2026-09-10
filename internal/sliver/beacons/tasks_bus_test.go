package beacons

import (
	"context"
	"errors"
	"testing"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"google.golang.org/grpc"

	automationevents "siren/internal/automation/events"
	"siren/internal/bus"
	"siren/internal/execctx"
	"siren/internal/sliver/console"
	"siren/internal/sliver/rpc"
)

type recordingBus struct {
	events []bus.Event
}

func (b *recordingBus) Publish(ev bus.Event) {
	b.events = append(b.events, ev)
}

func (b *recordingBus) Subscribe(_ []string, _ bus.Handler) func() {
	return func() {}
}

type fakeAwaitRPC struct {
	rpcpb.SliverRPCClient
	getBeaconTasks       func(context.Context, *clientpb.Beacon, ...grpc.CallOption) (*clientpb.BeaconTasks, error)
	getBeaconTaskContent func(context.Context, *clientpb.BeaconTask, ...grpc.CallOption) (*clientpb.BeaconTask, error)
}

func (f *fakeAwaitRPC) GetBeaconTasks(
	ctx context.Context, in *clientpb.Beacon, opts ...grpc.CallOption,
) (*clientpb.BeaconTasks, error) {
	return f.getBeaconTasks(ctx, in, opts...)
}

func (f *fakeAwaitRPC) GetBeaconTaskContent(
	ctx context.Context, in *clientpb.BeaconTask, opts ...grpc.CallOption,
) (*clientpb.BeaconTask, error) {
	return f.getBeaconTaskContent(ctx, in, opts...)
}

func awaitTestService(b *recordingBus, fake *fakeAwaitRPC) *Service {
	return &Service{rpc: &rpc.Client{RPC: fake}, console: &console.Service{}, bus: b}
}

func TestAwaitBeaconTaskResolveFailurePublishesNothing(t *testing.T) {
	b := &recordingBus{}
	fake := &fakeAwaitRPC{
		getBeaconTasks: func(context.Context, *clientpb.Beacon, ...grpc.CallOption) (*clientpb.BeaconTasks, error) {
			return nil, errors.New("list failed")
		},
	}
	s := awaitTestService(b, fake)

	_, awaited, err := s.AwaitBeaconTask(context.Background(), "beacon-1", "Tasked beacon demo (deadbeef)", "")
	if err == nil || !awaited {
		t.Fatalf("err=%v awaited=%v, want resolve error", err, awaited)
	}
	if len(b.events) != 0 {
		t.Fatalf("resolve failure published %d events, want 0", len(b.events))
	}
}

func TestAwaitBeaconTaskWaitFailurePublishesOnce(t *testing.T) {
	b := &recordingBus{}
	fake := &fakeAwaitRPC{
		getBeaconTaskContent: func(context.Context, *clientpb.BeaconTask, ...grpc.CallOption) (*clientpb.BeaconTask, error) {
			return nil, errors.New("poll failed")
		},
	}
	s := awaitTestService(b, fake)
	ctx := execctx.WithTarget(context.Background(), "beacon-1", "beacon", "host-1")

	_, awaited, err := s.AwaitBeaconTask(ctx, "beacon-1", "Tasked beacon demo (deadbeef)", "deadbeef-0000")
	if err == nil || !awaited {
		t.Fatalf("err=%v awaited=%v, want wait error", err, awaited)
	}
	if len(b.events) != 1 {
		t.Fatalf("wait failure published %d events, want 1", len(b.events))
	}
	result, ok := b.events[0].Payload.(automationevents.TaskResult)
	if !ok || result.Status != "error" || result.TargetID != "beacon-1" || result.Hostname != "host-1" {
		t.Fatalf("payload: %#v", b.events[0].Payload)
	}
}

func TestPublishBeaconTaskResultNilSafe(t *testing.T) {
	s := &Service{}
	s.publishBeaconTaskResult(context.Background(), "b1", nil)
}

func TestPublishBeaconTaskResultPublishesStatus(t *testing.T) {
	b := &recordingBus{}
	s := &Service{bus: b}
	ctx := execctx.WithTarget(context.Background(), "sess-1", "session", "host-1")

	s.publishBeaconTaskResult(ctx, "b1", nil)
	s.publishBeaconTaskResult(ctx, "b2", errors.New("boom"))

	if len(b.events) != 2 {
		t.Fatalf("events: %d", len(b.events))
	}
	if b.events[0].Type != "beacon.task-result" {
		t.Fatalf("type: %q", b.events[0].Type)
	}
	first, isResult := b.events[0].Payload.(automationevents.TaskResult)
	if !isResult {
		t.Fatalf("payload: %T", b.events[0].Payload)
	}
	if first.Verb != "BeaconTaskResult" || first.TargetID != "b1" || first.TargetKind != "beacon" {
		t.Fatalf("result: %+v", first)
	}
	if first.Status != "ok" || first.Error != "" {
		t.Fatalf("result: %+v", first)
	}
	if first.Hostname != "host-1" {
		t.Fatalf("hostname: %q", first.Hostname)
	}
	failed, _ := b.events[1].Payload.(automationevents.TaskResult)
	if failed.TargetID != "b2" || failed.Status != "error" || failed.Error != "boom" {
		t.Fatalf("result: %+v", failed)
	}
}
