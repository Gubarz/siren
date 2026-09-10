package rpc

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/gubarz/revils/capture"
	"github.com/gubarz/revils/store"

	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"google.golang.org/grpc"

	"siren/internal/captureann"
)

// fakeRPC embeds the interface (unimplemented methods panic) and stubs the
// verb the test exercises.
type fakeRPC struct {
	rpcpb.SliverRPCClient
	lsCalls int
}

func (f *fakeRPC) Ls(_ context.Context, req *sliverpb.LsReq, _ ...grpc.CallOption) (*sliverpb.Ls, error) {
	f.lsCalls++
	return &sliverpb.Ls{Path: req.GetPath()}, nil
}

func TestCaptureDecoratorRecordsUnaryCall(t *testing.T) {
	key := make([]byte, 32)
	st, err := store.Open(store.Config{DBPath: filepath.Join(t.TempDir(), "d.sqlite"), Key: key})
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	defer st.Close()

	rec := capture.NewRecorder(st, captureann.New("op"))
	inner := &fakeRPC{}
	dec := WrapCapture(inner, rec)

	_, err = dec.Ls(context.Background(), &sliverpb.LsReq{
		Path:    "/tmp",
		Request: &commonpb.Request{SessionID: "sess-9"},
	})
	if err != nil {
		t.Fatalf("ls: %v", err)
	}
	if inner.lsCalls != 1 {
		t.Fatalf("inner calls = %d", inner.lsCalls)
	}
	env, err := st.Query(store.Filter{Kind: store.KindCall, Direction: store.DirectionRequest, Limit: 10})
	if err != nil || len(env) != 1 {
		t.Fatalf("envelopes: %v %+v", err, env)
	}
	if env[0].SessionID != "sess-9" {
		t.Fatalf("session = %q", env[0].SessionID)
	}
	if n, _ := st.Count(store.Filter{Kind: store.KindMessage, ChainRef: env[0].ChainRef}); n != 2 {
		t.Fatalf("messages = %d, want 2", n)
	}
	if n, _ := st.Count(store.Filter{Kind: store.KindCall, Direction: store.DirectionComplete, ChainRef: env[0].ChainRef}); n != 1 {
		t.Fatalf("completions = %d, want 1", n)
	}
}

func TestGeneratedDecoratorUpToDate(t *testing.T) {
	if testing.Short() {
		t.Skip("generator check runs the generator")
	}
	tmp := t.TempDir() + "/capture_decorator.go"
	out, err := exec.Command("go", "run", "./gen", "-out", tmp).CombinedOutput()
	if err != nil {
		t.Fatalf("generator: %v\n%s", err, out)
	}
	want, _ := os.ReadFile("capture_decorator.go")
	got, _ := os.ReadFile(tmp)
	if !bytes.Equal(got, want) {
		t.Fatal("capture_decorator.go is stale — run: go generate ./internal/sliver/rpc")
	}
}
