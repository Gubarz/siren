package actions

import (
	"testing"

	"github.com/gubarz/revils/store"
)

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
