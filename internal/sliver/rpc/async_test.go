package rpc

import (
	"context"
	"testing"
	"time"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
)

type fakeSliverRPC struct {
	rpcpb.SliverRPCClient
	getBeaconTaskContent func(ctx context.Context, in *clientpb.BeaconTask, opts ...grpc.CallOption) (*clientpb.BeaconTask, error)
}

func (f *fakeSliverRPC) GetBeaconTaskContent(
	ctx context.Context, in *clientpb.BeaconTask, opts ...grpc.CallOption,
) (*clientpb.BeaconTask, error) {
	return f.getBeaconTaskContent(ctx, in, opts...)
}

func clientWithAgents(sessions []string, beacons []string) *Client {
	c := &Client{}
	ss := &clientpb.Sessions{}
	for _, id := range sessions {
		ss.Sessions = append(ss.Sessions, &clientpb.Session{ID: id})
	}
	c.PopulateSessions(ss)
	bs := &clientpb.Beacons{}
	for _, id := range beacons {
		bs.Beacons = append(bs.Beacons, &clientpb.Beacon{ID: id})
	}
	c.PopulateBeacons(bs)
	return c
}

func TestTargetRequest_Session(t *testing.T) {
	c := clientWithAgents([]string{"sess-1"}, []string{"beacon-1"})
	req, err := c.TargetRequest("sess-1", time.Minute)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.SessionID != "sess-1" || req.Async || req.BeaconID != "" {
		t.Fatalf("expected sync session request, got %+v", req)
	}
}

func TestTargetRequest_Beacon(t *testing.T) {
	c := clientWithAgents([]string{"sess-1"}, []string{"beacon-1"})
	req, err := c.TargetRequest("beacon-1", time.Minute)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.BeaconID != "beacon-1" || !req.Async || req.SessionID != "" {
		t.Fatalf("expected async beacon request, got %+v", req)
	}
}

func TestTargetRequest_Unknown(t *testing.T) {
	c := clientWithAgents(nil, nil)
	if _, err := c.TargetRequest("ghost", time.Minute); err == nil {
		t.Fatal("expected error for unknown agent, got nil")
	}
}

func TestAwaitAsyncResponse_SyncPassthrough(t *testing.T) {
	c := &Client{}
	resp := &sliverpb.Download{Data: []byte("sync")}
	if err := c.AwaitAsyncResponse(context.Background(), resp, resp); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(resp.Data) != "sync" {
		t.Fatalf("sync response mutated: %q", resp.Data)
	}
}

func TestAwaitAsyncResponse_BeaconTask(t *testing.T) {
	payload := []byte("from-beacon")
	taskData, err := proto.Marshal(&sliverpb.Download{Data: payload})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	c := &Client{}
	c.RPC = &fakeSliverRPC{
		getBeaconTaskContent: func(_ context.Context, in *clientpb.BeaconTask, _ ...grpc.CallOption) (*clientpb.BeaconTask, error) {
			if in.ID != "task-1" {
				t.Fatalf("polled wrong task ID: %q", in.ID)
			}
			return &clientpb.BeaconTask{ID: in.ID, State: "completed", Response: taskData}, nil
		},
	}

	resp := &sliverpb.Download{
		Response: &commonpb.Response{Async: true, TaskID: "task-1"},
	}
	if err := c.AwaitAsyncResponse(context.Background(), resp, resp); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(resp.Data) != string(payload) {
		t.Fatalf("expected %q, got %q", payload, resp.Data)
	}
}

func TestAwaitAsyncResponse_BeaconTaskFailed(t *testing.T) {
	c := &Client{}
	c.RPC = &fakeSliverRPC{
		getBeaconTaskContent: func(_ context.Context, in *clientpb.BeaconTask, _ ...grpc.CallOption) (*clientpb.BeaconTask, error) {
			return &clientpb.BeaconTask{ID: in.ID, State: "failed"}, nil
		},
	}
	resp := &sliverpb.Download{
		Response: &commonpb.Response{Async: true, TaskID: "task-1"},
	}
	if err := c.AwaitAsyncResponse(context.Background(), resp, resp); err == nil {
		t.Fatal("expected error for failed task, got nil")
	}
}
