package gui

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gubarz/revils/capture"
	"github.com/gubarz/revils/store"

	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"google.golang.org/protobuf/proto"

	"siren/internal/captureann"
	"siren/internal/execctx"
)

type seededTask struct {
	chainRef string
	request  []byte
	response []byte
}

func openTaskStore(t *testing.T) *store.Store {
	t.Helper()
	st, err := store.Open(store.Config{
		DBPath: filepath.Join(t.TempDir(), "tasks.sqlite"),
		Key:    make([]byte, 32),
	})
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return st
}

// seedTask records one Ls call plus a verified evidence row for its run.
func seedTask(t *testing.T, st *store.Store) seededTask {
	t.Helper()
	ctx := execctx.WithRun(context.Background(), "run-5", "stage-1")
	rec := capture.NewRecorder(st, captureann.New("op"))
	req, err := proto.Marshal(&sliverpb.LsReq{
		Path:    "/tmp",
		Request: &commonpb.Request{SessionID: "sess-9"},
	})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	call := rec.Begin(ctx, "/rpcpb.SliverRPC/Ls", req)
	if call == nil {
		t.Fatal("begin returned nil")
	}
	call.Message(store.DirectionRequest, req, "")
	resp, err := proto.Marshal(&sliverpb.Ls{
		Path:  "/tmp",
		Files: []*sliverpb.FileInfo{{Name: "a.txt", Size: 3}},
	})
	if err != nil {
		t.Fatalf("marshal response: %v", err)
	}
	call.Message(store.DirectionResponse, resp, "")
	call.End(nil)
	if _, err := st.Write(store.Record{
		Kind: store.KindEvidence, Status: store.StatusVerified,
		RunID: "run-5", StageID: "stage-1", Method: "automation.execute",
	}); err != nil {
		t.Fatalf("evidence: %v", err)
	}
	env, err := st.Query(store.Filter{
		Kind: store.KindCall, Direction: store.DirectionRequest, Limit: 1,
	})
	if err != nil || len(env) != 1 {
		t.Fatalf("envelope: %v %+v", err, env)
	}
	return seededTask{chainRef: env[0].ChainRef, request: req, response: resp}
}

func TestListSessionTasks(t *testing.T) {
	st := openTaskStore(t)
	seed := seedTask(t, st)

	rows, err := listSessionTasks(st, "sess-9", 10)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("rows = %d, want 1", len(rows))
	}
	row := rows[0]
	if row.Status != "OK" {
		t.Fatalf("status = %q", row.Status)
	}
	if row.Evidence != store.StatusVerified {
		t.Fatalf("evidence = %q", row.Evidence)
	}
	if row.RunID != "run-5" || row.StageID != "stage-1" {
		t.Fatalf("run/stage = %q/%q", row.RunID, row.StageID)
	}
	if row.ChainRef == "" || row.ChainRef != seed.chainRef {
		t.Fatalf("chain ref = %q", row.ChainRef)
	}
	if row.RequestBytes != int64(len(seed.request)) || row.ResponseBytes != int64(len(seed.response)) {
		t.Fatalf("bytes = %d/%d, want %d/%d",
			row.RequestBytes, row.ResponseBytes, len(seed.request), len(seed.response))
	}
	if row.Method != "/rpcpb.SliverRPC/Ls" {
		t.Fatalf("method = %q", row.Method)
	}
}

func TestGetTaskCallPayloads(t *testing.T) {
	st := openTaskStore(t)
	seed := seedTask(t, st)

	views, err := getTaskCallPayloads(st, seed.chainRef)
	if err != nil {
		t.Fatalf("payloads: %v", err)
	}
	if len(views) != 2 {
		t.Fatalf("views = %d, want 2", len(views))
	}
	for _, v := range views {
		if v.Base64 == "" {
			t.Fatalf("empty base64 for %s", v.Direction)
		}
		if !json.Valid([]byte(v.JSON)) {
			t.Fatalf("invalid json for %s: %q", v.Direction, v.JSON)
		}
	}
	var response TaskPayloadView
	for _, v := range views {
		if v.Direction == store.DirectionResponse {
			response = v
		}
	}
	if !strings.Contains(response.JSON, `"a.txt"`) {
		t.Fatalf("response json = %s", response.JSON)
	}
	raw, err := base64.StdEncoding.DecodeString(response.Base64)
	if err != nil {
		t.Fatalf("base64 decode: %v", err)
	}
	var ls sliverpb.Ls
	if err := proto.Unmarshal(raw, &ls); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(ls.Files) != 1 || ls.Files[0].Name != "a.txt" {
		t.Fatalf("decoded ls = %+v", &ls)
	}
}
