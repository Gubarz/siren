package captureann

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/gubarz/revils/capture"

	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"google.golang.org/protobuf/proto"

	"siren/internal/execctx"

	_ "github.com/bishopfox/sliver/protobuf/rpcpb"
)

func TestAnnotateExtractsSessionFromRequestField(t *testing.T) {
	ann := New("operator-a")
	req := &sliverpb.LsReq{
		Path:    "/tmp",
		Request: &commonpb.Request{SessionID: "sess-42"},
	}
	payload, err := proto.Marshal(req)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	ctx := execctx.WithRun(context.Background(), "run-1", "commands#0")
	got := ann.Annotate(ctx, "/rpcpb.SliverRPC/Ls", "request", payload)
	if got.Operator != "operator-a" {
		t.Fatalf("operator = %q", got.Operator)
	}
	if got.SessionID != "sess-42" {
		t.Fatalf("session = %q", got.SessionID)
	}
	if got.RunID != "run-1" || got.StageID != "commands#0" {
		t.Fatalf("run/stage = %q/%q", got.RunID, got.StageID)
	}
	var preview map[string]any
	if err := json.Unmarshal([]byte(got.RequestPreview), &preview); err != nil {
		t.Fatalf("preview not json: %v (%s)", err, got.RequestPreview)
	}
	if preview["method"] != "/rpcpb.SliverRPC/Ls" {
		t.Fatalf("preview = %s", got.RequestPreview)
	}
}

func TestAnnotateUnknownPayloadIsSafe(t *testing.T) {
	ann := New("op")
	got := ann.Annotate(context.Background(), "/not/a/method", "request", []byte("garbage"))
	if got.Operator != "op" || got.SessionID != "" {
		t.Fatalf("unexpected annotation: %+v", got)
	}
}

var _ capture.Annotator = (*Annotator)(nil)
