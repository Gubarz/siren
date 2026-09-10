package rpc

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/gubarz/revils/capture"
	"github.com/gubarz/revils/store"

	"siren/internal/captureann"
	"siren/internal/sliver/rpctest"
)

type versionTeamserver struct {
	rpcpb.UnimplementedSliverRPCServer
}

func (versionTeamserver) GetVersion(context.Context, *commonpb.Empty) (*clientpb.Version, error) {
	return &clientpb.Version{Major: 9, Minor: 9, Patch: 9}, nil
}

// The decorator passes every RPC through to Sliver and records it on the way.
// Exercising that over a real gRPC server covers the pass-through, the response
// and the recording together, which a stub client cannot: the stub panics on any
// method it does not declare.
func TestCaptureDecoratorPassesThroughAndRecordsOverRealGRPC(t *testing.T) {
	h := rpctest.Start(t, versionTeamserver{})

	st, err := store.Open(store.Config{
		DBPath: filepath.Join(t.TempDir(), "capture.sqlite"),
		Key:    make([]byte, 32),
	})
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	client := WrapCapture(h.Client, capture.NewRecorder(st, captureann.New("operator")))

	resp, err := client.GetVersion(context.Background(), &commonpb.Empty{})
	if err != nil {
		t.Fatalf("GetVersion() = %v, want nil", err)
	}
	if resp.GetMajor() != 9 || resp.GetMinor() != 9 {
		t.Fatalf("version = %d.%d, want 9.9", resp.GetMajor(), resp.GetMinor())
	}

	rows, err := st.Query(store.Filter{
		Kind:      store.KindCall,
		Direction: store.DirectionComplete,
		Limit:     10,
	})
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("recorded %d calls, want 1", len(rows))
	}
	if !strings.Contains(rows[0].Method, "GetVersion") {
		t.Fatalf("recorded method = %q, want it to name GetVersion", rows[0].Method)
	}
}
