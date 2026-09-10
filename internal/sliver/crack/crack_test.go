package crack

import (
	"context"
	"testing"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"

	"siren/internal/sliver/rpc"
	"siren/internal/sliver/rpctest"
)

type fakeCrackstation struct {
	rpcpb.UnimplementedSliverRPCServer

	submitted []*clientpb.CrackCommand
}

func (f *fakeCrackstation) Crackstations(context.Context, *commonpb.Empty) (*clientpb.Crackstations, error) {
	return &clientpb.Crackstations{Crackstations: []*clientpb.Crackstation{{}}}, nil
}

func (f *fakeCrackstation) Crack(_ context.Context, cmd *clientpb.CrackCommand) (*clientpb.CrackResponse, error) {
	f.submitted = append(f.submitted, cmd)
	return &clientpb.CrackResponse{}, nil
}

func TestCrackstationsReturnsWhatTheServerSends(t *testing.T) {
	h := rpctest.Start(t, &fakeCrackstation{})
	svc := New(rpc.NewForTest(h.Client))

	stations, err := svc.Crackstations()
	if err != nil {
		t.Fatalf("Crackstations() = %v, want nil", err)
	}
	if len(stations.GetCrackstations()) != 1 {
		t.Fatalf("got %d stations, want the one the server sent", len(stations.GetCrackstations()))
	}
}

// The wrapper hands its CrackCommand straight to the RPC, so a field it failed to
// carry would otherwise go unnoticed.
func TestSubmitJobPassesTheCommandThrough(t *testing.T) {
	srv := &fakeCrackstation{}
	h := rpctest.Start(t, srv)
	svc := New(rpc.NewForTest(h.Client))

	cmd := &clientpb.CrackCommand{Hashes: []string{"h1", "h2"}, Quiet: true}
	if _, err := svc.SubmitJob(cmd); err != nil {
		t.Fatalf("SubmitJob() = %v, want nil", err)
	}

	if len(srv.submitted) != 1 {
		t.Fatalf("the server saw %d commands, want 1", len(srv.submitted))
	}
	got := srv.submitted[0]
	if len(got.GetHashes()) != 2 || got.GetHashes()[0] != "h1" || !got.GetQuiet() {
		t.Fatalf("the server saw %+v, want the command's hashes and quiet flag", got)
	}
}
