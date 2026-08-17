package staging

import (
	"context"
	"errors"
	"testing"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"siren/internal/sliver/rpc"
)

func TestUnstageImplantBuildRequiresConnection(t *testing.T) {
	fake := &fakeStagingRPC{}
	svc := &Service{rpc: fake}

	err := svc.UnstageImplantBuild("alpha")

	if !errors.Is(err, rpc.ErrNotConnected) {
		t.Fatalf("UnstageImplantBuild() error = %v, want %v", err, rpc.ErrNotConnected)
	}
	if fake.stageReq != nil {
		t.Fatal("StageImplantBuild was called while disconnected")
	}
}

func TestUnstageImplantBuildKeepsOtherStagedBuilds(t *testing.T) {
	fake := &fakeStagingRPC{
		connected: true,
		builds: &clientpb.ImplantBuilds{Staged: map[string]bool{
			"alpha": true,
			"beta":  true,
			"gamma": false,
		}},
	}
	svc := &Service{rpc: fake}

	err := svc.UnstageImplantBuild("alpha")

	if err != nil {
		t.Fatalf("UnstageImplantBuild() returned error: %v", err)
	}
	if fake.stageReq == nil {
		t.Fatal("StageImplantBuild was not called")
	}
	assertStrings(t, fake.stageReq.Build, []string{"beta"})
}

func TestUnstageImplantBuildErrorsWhenNotStaged(t *testing.T) {
	fake := &fakeStagingRPC{
		connected: true,
		builds:    &clientpb.ImplantBuilds{Staged: map[string]bool{"alpha": true}},
	}
	svc := &Service{rpc: fake}

	err := svc.UnstageImplantBuild("beta")

	if err == nil || err.Error() != `build "beta" is not staged` {
		t.Fatalf("UnstageImplantBuild() error = %v, want not-staged error", err)
	}
	if fake.stageReq != nil {
		t.Fatal("StageImplantBuild was called for an unstaged build")
	}
}

func TestUnstageImplantBuildTreatsEmptyTableAsNotStaged(t *testing.T) {
	fake := &fakeStagingRPC{
		connected: true,
		buildsErr: status.Error(codes.NotFound, "record not found"),
	}
	svc := &Service{rpc: fake}

	err := svc.UnstageImplantBuild("alpha")

	if err == nil || err.Error() != `build "alpha" is not staged` {
		t.Fatalf("UnstageImplantBuild() error = %v, want not-staged error", err)
	}
	if fake.stageReq != nil {
		t.Fatal("StageImplantBuild was called for an empty build table")
	}
}

func TestUnstageAllImplantBuildsRequiresConnection(t *testing.T) {
	fake := &fakeStagingRPC{}
	svc := &Service{rpc: fake}

	err := svc.UnstageAllImplantBuilds()

	if !errors.Is(err, rpc.ErrNotConnected) {
		t.Fatalf("UnstageAllImplantBuilds() error = %v, want %v", err, rpc.ErrNotConnected)
	}
	if fake.stageReq != nil {
		t.Fatal("StageImplantBuild was called while disconnected")
	}
}

func TestUnstageAllImplantBuildsSubmitsEmptyBuildList(t *testing.T) {
	fake := &fakeStagingRPC{connected: true}
	svc := &Service{rpc: fake}

	err := svc.UnstageAllImplantBuilds()

	if err != nil {
		t.Fatalf("UnstageAllImplantBuilds() returned error: %v", err)
	}
	if fake.stageReq == nil {
		t.Fatal("StageImplantBuild was not called")
	}
	if len(fake.stageReq.Build) != 0 {
		t.Fatalf("StageImplantBuild Build = %v, want empty list", fake.stageReq.Build)
	}
}

func TestUnstageImplantBuildPropagatesRPCError(t *testing.T) {
	wantErr := errors.New("listing failed")
	fake := &fakeStagingRPC{connected: true, buildsErr: wantErr}
	svc := &Service{rpc: fake}

	err := svc.UnstageImplantBuild("alpha")

	if !errors.Is(err, wantErr) {
		t.Fatalf("UnstageImplantBuild() error = %v, want %v", err, wantErr)
	}
}

type fakeStagingRPC struct {
	connected bool
	builds    *clientpb.ImplantBuilds
	buildsErr error
	stageReq  *clientpb.ImplantStageReq
	stageErr  error
}

func (f *fakeStagingRPC) Connected() bool {
	return f.connected
}

func (f *fakeStagingRPC) StageImplantBuild(
	_ context.Context,
	req *clientpb.ImplantStageReq,
) (*commonpb.Empty, error) {
	f.stageReq = req
	return &commonpb.Empty{}, f.stageErr
}

func (f *fakeStagingRPC) ImplantBuilds(
	_ context.Context,
	_ *commonpb.Empty,
) (*clientpb.ImplantBuilds, error) {
	return f.builds, f.buildsErr
}

func (f *fakeStagingRPC) GenerateStage(
	_ context.Context,
	_ *clientpb.GenerateStageReq,
) (*clientpb.Generate, error) {
	return nil, nil
}

func (f *fakeStagingRPC) StartTCPStagerListener(
	_ context.Context,
	_ *clientpb.StagerListenerReq,
) (*clientpb.StagerListener, error) {
	return nil, nil
}

func assertStrings(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("len(%v) = %d, want %d", got, len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("slice[%d] = %q, want %q (full slice %v)", i, got[i], want[i], got)
		}
	}
}
