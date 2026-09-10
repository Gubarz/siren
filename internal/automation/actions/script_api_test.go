package actions

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/gubarz/revils/capture"
	"github.com/gubarz/revils/store"

	"siren/internal/captureann"
)

func TestNormalizeRecordStatus(t *testing.T) {
	cases := map[string]string{
		"OK":        "ok",
		"ok":        "ok",
		"":          "attempted",
		"attempted": "attempted",
		"Canceled":  "error",
		"Unknown":   "error",
		"error":     "error",
	}
	for in, want := range cases {
		if got := normalizeRecordStatus(in); got != want {
			t.Fatalf("normalize(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestMapRecordRowsNormalizesRealCompletionStatus(t *testing.T) {
	st, err := store.Open(store.Config{
		DBPath: filepath.Join(t.TempDir(), "records.sqlite"),
		Key:    make([]byte, 32),
	})
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	rec := capture.NewRecorder(st, captureann.New("op"))
	call := rec.Begin(context.Background(), "/rpcpb.SliverRPC/Ls", nil)
	if call == nil {
		t.Fatal("begin returned nil")
	}
	call.End(nil)

	rows, err := st.Query(store.Filter{
		Kind: store.KindCall, Direction: store.DirectionComplete, Limit: 10,
	})
	if err != nil || len(rows) != 1 {
		t.Fatalf("rows: %v %+v", err, rows)
	}
	if rows[0].Status != "OK" {
		t.Fatalf("seeded status = %q, want gRPC code OK", rows[0].Status)
	}
	entries, total := mapRecordRows(rows, recordFilter{})
	if total != 1 || len(entries) != 1 || entries[0].Status != "ok" {
		t.Fatalf("normalized: %+v total %d", entries, total)
	}
	entries, total = mapRecordRows(rows, recordFilter{Status: "ok"})
	if total != 1 || len(entries) != 1 {
		t.Fatalf("status ok filter: %+v total %d", entries, total)
	}
	entries, total = mapRecordRows(rows, recordFilter{Status: "OK"})
	if total != 1 || len(entries) != 1 {
		t.Fatalf("raw code filter: %+v total %d", entries, total)
	}
	entries, total = mapRecordRows(rows, recordFilter{Status: "error"})
	if total != 0 || len(entries) != 0 {
		t.Fatalf("status error filter: %+v total %d", entries, total)
	}
}

func TestMapRecordRowsMapsCompletionFields(t *testing.T) {
	rows := []store.RecordRow{
		{ID: 7, TS: 1000, Record: store.Record{
			Method: "/rpcpb.SliverRPC/Ls", Status: store.StatusOK, SessionID: "sess-1",
			RunID: "run-1", JSON: `{"error":"","request_bytes":1,"response_bytes":2}`,
		}},
		{ID: 6, TS: 900, Record: store.Record{
			Method: "/rpcpb.SliverRPC/Ps", Status: store.StatusError, BeaconID: "b-1",
			JSON: `{"error":"rpc error: boom"}`,
		}},
	}
	entries, total := mapRecordRows(rows, recordFilter{})
	if total != 2 || len(entries) != 2 {
		t.Fatalf("entries: %d total %d", len(entries), total)
	}
	first := entries[0]
	if first.Verb != "Ls" || first.Status != "ok" || first.TargetID != "sess-1" || first.TargetKind != "session" || first.CorrelationID != "run-1" {
		t.Fatalf("entry: %+v", first)
	}
	second := entries[1]
	if second.Err != "rpc error: boom" || second.TargetID != "b-1" || second.TargetKind != "beacon" {
		t.Fatalf("entry: %+v", second)
	}
}

func TestMapRecordRowsFilters(t *testing.T) {
	rows := []store.RecordRow{
		{ID: 3, TS: 3000, Record: store.Record{Method: "/rpcpb.SliverRPC/Ls", Status: store.StatusOK, SessionID: "sess-1"}},
		{ID: 2, TS: 2000, Record: store.Record{Method: "/rpcpb.SliverRPC/Ps", Status: store.StatusError, BeaconID: "b-1"}},
		{ID: 1, TS: 1000, Record: store.Record{Method: "/rpcpb.SliverRPC/Ls", Status: store.StatusError, SessionID: "sess-2"}},
	}
	entries, total := mapRecordRows(rows, recordFilter{Verb: "Ls", TargetID: "sess-1", Since: 500, Until: 3500})
	if total != 1 || len(entries) != 1 || entries[0].ID != 3 {
		t.Fatalf("filtered: %+v total %d", entries, total)
	}
	entries, total = mapRecordRows(rows, recordFilter{Limit: 2})
	if total != 3 || len(entries) != 2 {
		t.Fatalf("limited: %d total %d", len(entries), total)
	}
}
