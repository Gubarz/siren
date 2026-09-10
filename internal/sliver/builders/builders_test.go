package builders

import (
	"context"
	"testing"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"

	"siren/internal/sliver/rpc"
	"siren/internal/sliver/rpctest"
)

type fakeBuilder struct {
	rpcpb.UnimplementedSliverRPCServer

	generated []*clientpb.ExternalGenerateReq
}

func (f *fakeBuilder) Builders(context.Context, *commonpb.Empty) (*clientpb.Builders, error) {
	return &clientpb.Builders{}, nil
}

func (f *fakeBuilder) GenerateExternal(_ context.Context, req *clientpb.ExternalGenerateReq) (*clientpb.ExternalImplantConfig, error) {
	f.generated = append(f.generated, req)
	return &clientpb.ExternalImplantConfig{}, nil
}

// This is the distributed-builder path. The request names the builder to run and
// the implant to produce, and losing either sends the build to the wrong place,
// which is not something a return value would reveal.
func TestGenerateExternalCarriesTheBuilderAndTheImplantName(t *testing.T) {
	srv := &fakeBuilder{}
	h := rpctest.Start(t, srv)
	svc := New(rpc.NewForTest(h.Client))

	req := &clientpb.ExternalGenerateReq{BuilderName: "builder-a", Name: "implant-b"}
	if _, err := svc.GenerateExternal(req); err != nil {
		t.Fatalf("GenerateExternal() = %v, want nil", err)
	}

	if len(srv.generated) != 1 {
		t.Fatalf("the server saw %d requests, want 1", len(srv.generated))
	}
	got := srv.generated[0]
	if got.GetBuilderName() != "builder-a" || got.GetName() != "implant-b" {
		t.Fatalf("the server saw %+v, want builder-a building implant-b", got)
	}
}
