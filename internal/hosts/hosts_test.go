package hosts

import (
	"context"
	"errors"
	"testing"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"

	"sliver-gui/internal/rpc"
)

func TestListRequiresConnection(t *testing.T) {
	fake := &fakeHostRPC{}
	svc := &Service{rpc: fake}

	_, err := svc.List()

	if !errors.Is(err, rpc.ErrNotConnected) {
		t.Fatalf("List() error = %v, want %v", err, rpc.ErrNotConnected)
	}
	if fake.hostsCalls != 0 {
		t.Fatalf("Hosts called %d times while disconnected", fake.hostsCalls)
	}
}

func TestListReturnsHostsFromRPC(t *testing.T) {
	want := &clientpb.AllHosts{Hosts: []*clientpb.Host{{HostUUID: "host-1"}}}
	fake := &fakeHostRPC{connected: true, allHosts: want}
	svc := &Service{rpc: fake}

	got, err := svc.List()

	if err != nil {
		t.Fatalf("List() returned error: %v", err)
	}
	if got != want {
		t.Fatalf("List() = %p, want %p", got, want)
	}
	if fake.hostsCalls != 1 {
		t.Fatalf("Hosts called %d times, want 1", fake.hostsCalls)
	}
}

func TestGetTrimsHostUUIDBeforeCallingRPC(t *testing.T) {
	fake := &fakeHostRPC{connected: true, host: &clientpb.Host{HostUUID: "host-1"}}
	svc := &Service{rpc: fake}

	_, err := svc.Get(" host-1 ")

	if err != nil {
		t.Fatalf("Get() returned error: %v", err)
	}
	if fake.hostReq == nil || fake.hostReq.HostUUID != "host-1" {
		t.Fatalf("Host request = %#v, want HostUUID host-1", fake.hostReq)
	}
}

func TestGetRejectsEmptyHostUUID(t *testing.T) {
	fake := &fakeHostRPC{connected: true}
	svc := &Service{rpc: fake}

	_, err := svc.Get(" \t ")

	if err == nil || err.Error() != "host UUID is required" {
		t.Fatalf("Get() error = %v, want host UUID validation error", err)
	}
	if fake.hostReq != nil {
		t.Fatalf("Host request = %#v, want no RPC call", fake.hostReq)
	}
}

func TestRemoveTrimsHostUUIDAndPropagatesRPCError(t *testing.T) {
	wantErr := errors.New("rpc failed")
	fake := &fakeHostRPC{connected: true, hostRmErr: wantErr}
	svc := &Service{rpc: fake}

	err := svc.Remove(" host-1 ")

	if !errors.Is(err, wantErr) {
		t.Fatalf("Remove() error = %v, want %v", err, wantErr)
	}
	if fake.hostRmReq == nil || fake.hostRmReq.HostUUID != "host-1" {
		t.Fatalf("HostRm request = %#v, want HostUUID host-1", fake.hostRmReq)
	}
}

func TestRemoveIOCTrimsIDBeforeCallingRPC(t *testing.T) {
	fake := &fakeHostRPC{connected: true}
	svc := &Service{rpc: fake}

	err := svc.RemoveIOC(" ioc-1 ")

	if err != nil {
		t.Fatalf("RemoveIOC() returned error: %v", err)
	}
	if fake.iocRmReq == nil || fake.iocRmReq.ID != "ioc-1" {
		t.Fatalf("HostIOCRm request = %#v, want ID ioc-1", fake.iocRmReq)
	}
}

type fakeHostRPC struct {
	connected bool

	allHosts *clientpb.AllHosts
	host     *clientpb.Host

	hostsCalls int
	hostReq    *clientpb.Host
	hostRmReq  *clientpb.Host
	iocRmReq   *clientpb.IOC

	hostsErr  error
	hostErr   error
	hostRmErr error
	iocRmErr  error
}

func (f *fakeHostRPC) Connected() bool {
	return f.connected
}

func (f *fakeHostRPC) Hosts(context.Context, *commonpb.Empty) (*clientpb.AllHosts, error) {
	f.hostsCalls++
	return f.allHosts, f.hostsErr
}

func (f *fakeHostRPC) Host(_ context.Context, req *clientpb.Host) (*clientpb.Host, error) {
	f.hostReq = req
	return f.host, f.hostErr
}

func (f *fakeHostRPC) HostRm(_ context.Context, req *clientpb.Host) (*commonpb.Empty, error) {
	f.hostRmReq = req
	return &commonpb.Empty{}, f.hostRmErr
}

func (f *fakeHostRPC) HostIOCRm(_ context.Context, req *clientpb.IOC) (*commonpb.Empty, error) {
	f.iocRmReq = req
	return &commonpb.Empty{}, f.iocRmErr
}
