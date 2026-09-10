package captureann

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
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

func TestAnnotateFallsBackToCurrentExecution(t *testing.T) {
	ann := New("operator-a")
	req := &sliverpb.LsReq{
		Path:    "/tmp",
		Request: &commonpb.Request{SessionID: "sess-77"},
	}
	payload, err := proto.Marshal(req)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	restore := execctx.SetCurrent(execctx.Snapshot{
		RunID: "run-9", StageID: "commands#1",
		TargetID: "sess-77", TargetKind: "session", Hostname: "host-9",
	})
	defer restore()
	got := ann.Annotate(context.Background(), "/rpcpb.SliverRPC/Ls", "request", payload)
	if got.RunID != "run-9" || got.StageID != "commands#1" {
		t.Fatalf("run/stage = %q/%q", got.RunID, got.StageID)
	}
	if got.SessionID != "sess-77" {
		t.Fatalf("session = %q", got.SessionID)
	}
}

func TestAnnotateUnknownPayloadIsSafe(t *testing.T) {
	ann := New("op")
	got := ann.Annotate(context.Background(), "/not/a/method", "request", []byte("garbage"))
	if got.Operator != "op" || got.SessionID != "" {
		t.Fatalf("unexpected annotation: %+v", got)
	}
}

// previewLeakCases are requests carrying operator secrets. The preview is the
// only part of a record stored in plaintext, so it must never repeat any of them.
var previewLeakCases = []struct {
	name    string
	method  string
	payload proto.Message
	secret  string
}{
	{
		name:   "make-token password",
		method: "/rpcpb.SliverRPC/MakeToken",
		payload: &sliverpb.MakeTokenReq{
			Username: "svc-backup",
			Password: "Sup3rSecret!",
			Domain:   "CORP",
			Request:  &commonpb.Request{SessionID: "sess-1"},
		},
		secret: "Sup3rSecret!",
	},
	{
		name:   "execute-assembly arguments",
		method: "/rpcpb.SliverRPC/ExecuteAssembly",
		payload: &sliverpb.ExecuteAssemblyReq{
			Assembly:  []byte("MZ-assembly-marker"),
			Arguments: []string{"-password", "Hunter2!"},
			Request:   &commonpb.Request{SessionID: "sess-1"},
		},
		secret: "Hunter2!",
	},
}

// Each case marshals a request carrying an operator secret, checks the secret is
// really in the bytes, then asserts the preview does not repeat it.
func TestPreviewCarriesNoPayloadContent(t *testing.T) {
	ann := New("operator-a")

	for _, tc := range previewLeakCases {
		t.Run(tc.name, func(t *testing.T) {
			payload, err := proto.Marshal(tc.payload)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			if !bytes.Contains(payload, []byte(tc.secret)) {
				t.Fatalf("the payload does not contain %q, so this case proves nothing", tc.secret)
			}

			got := ann.Annotate(context.Background(), tc.method, "request", payload)
			if strings.Contains(got.RequestPreview, tc.secret) {
				t.Errorf("request preview leaked payload content: %s", got.RequestPreview)
			}
			if strings.Contains(got.ResponsePreview, tc.secret) {
				t.Errorf("response preview leaked payload content: %s", got.ResponsePreview)
			}
		})
	}
}

// The preview is correlation metadata. A new key in it is a new decision about
// what gets stored in the clear, so the shape is pinned here.
func TestPreviewContainsOnlyCorrelationMetadata(t *testing.T) {
	ann := New("operator-a")
	payload, err := proto.Marshal(&sliverpb.MakeTokenReq{
		Password: "Sup3rSecret!",
		Request:  &commonpb.Request{SessionID: "sess-1"},
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	got := ann.Annotate(context.Background(), "/rpcpb.SliverRPC/MakeToken", "request", payload)

	var preview map[string]any
	if err := json.Unmarshal([]byte(got.RequestPreview), &preview); err != nil {
		t.Fatalf("preview is not json: %v (%s)", err, got.RequestPreview)
	}

	allowed := map[string]bool{"method": true, "bytes": true, "session_id": true, "beacon_id": true}
	for key := range preview {
		if !allowed[key] {
			t.Errorf("preview carries unexpected key %q: %s", key, got.RequestPreview)
		}
	}
	for _, key := range []string{"method", "bytes", "session_id"} {
		if _, ok := preview[key]; !ok {
			t.Errorf("preview is missing %q: %s", key, got.RequestPreview)
		}
	}
}

var _ capture.Annotator = (*Annotator)(nil)
