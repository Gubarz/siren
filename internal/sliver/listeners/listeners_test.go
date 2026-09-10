package listeners

import (
	"context"
	"testing"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"

	"siren/internal/sliver/rpc"
	"siren/internal/sliver/rpctest"
)

type fakeTeamserver struct {
	rpcpb.UnimplementedSliverRPCServer
}

func (fakeTeamserver) GetJobs(context.Context, *commonpb.Empty) (*clientpb.Jobs, error) {
	return &clientpb.Jobs{Active: []*clientpb.Job{{ID: 7, Name: "listener-1"}}}, nil
}

// The service wrappers sit between the UI and the RPC surface and nothing
// verified they call the method they claim to. This one goes over real gRPC, so
// a wrapper wired to the wrong method comes back Unimplemented rather than
// quietly returning something.
func TestGetJobsReturnsWhatTheServerSends(t *testing.T) {
	h := rpctest.Start(t, fakeTeamserver{})
	svc := New(rpc.NewForTest(h.Client))

	jobs, err := svc.GetJobs()
	if err != nil {
		t.Fatalf("GetJobs() = %v, want nil", err)
	}
	got := jobs.GetActive()
	if len(got) != 1 {
		t.Fatalf("got %d jobs, want 1", len(got))
	}
	if got[0].GetID() != 7 || got[0].GetName() != "listener-1" {
		t.Fatalf("job = %+v, want the server's listener-1", got[0])
	}
}

// An unimplemented method answers Unimplemented rather than panicking, which is
// what makes the harness usable for methods a test does not care about.
func TestUnimplementedMethodsReportUnimplemented(t *testing.T) {
	h := rpctest.Start(t, fakeTeamserver{})

	if _, err := h.Client.GetVersion(context.Background(), &commonpb.Empty{}); err == nil {
		t.Fatal("GetVersion() = nil error on a server that does not implement it")
	}
}
