package journal

import (
	"context"
	"testing"
	"time"

	journalv1 "siren/internal/journal"
)

func openStore(t *testing.T) *SQLiteStore {
	t.Helper()
	store, err := NewSQLiteStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return store
}

func sampleEntries() []journalv1.Entry {
	base := time.Now().UnixMilli()
	return []journalv1.Entry{
		{ID: "1", Time: base - 3000, ConnectionID: "h:1", Verb: "Ps", ActorKind: "operator", TargetID: "t1", Status: "ok"},
		{ID: "2", Time: base - 2000, ConnectionID: "h:1", Verb: "Ps", ActorKind: "operator", TargetID: "t2", Status: "ok"},
		{ID: "3", Time: base - 1000, ConnectionID: "h:2", Verb: "Download", ActorKind: "automation", TargetID: "t1", Status: "error", Err: "x"},
	}
}

func TestInsertAndQueryRoundTrip(t *testing.T) {
	store := openStore(t)
	ctx := context.Background()
	if err := store.InsertBatch(ctx, sampleEntries()); err != nil {
		t.Fatal(err)
	}
	entries, total, err := store.Query(ctx, journalv1.Filter{})
	if err != nil {
		t.Fatal(err)
	}
	if total != 3 || len(entries) != 3 {
		t.Fatalf("got %d/%d, want 3/3", len(entries), total)
	}
	if entries[0].ID != "3" {
		t.Fatalf("expected newest first, got %s", entries[0].ID)
	}
}

func TestQueryFilters(t *testing.T) {
	store := openStore(t)
	ctx := context.Background()
	_ = store.InsertBatch(ctx, sampleEntries())

	entries, total, _ := store.Query(ctx, journalv1.Filter{ConnectionID: "h:1"})
	if total != 2 || len(entries) != 2 {
		t.Fatalf("connection filter: %d/%d", len(entries), total)
	}
	entries, total, _ = store.Query(ctx, journalv1.Filter{Verb: "Ps", TargetID: "t2"})
	if total != 1 || entries[0].ID != "2" {
		t.Fatalf("verb+target filter: %d", total)
	}
	entries, total, _ = store.Query(ctx, journalv1.Filter{ActorKind: "automation"})
	if total != 1 || entries[0].ID != "3" {
		t.Fatalf("actor filter: %d", total)
	}
}

func TestQueryPagination(t *testing.T) {
	store := openStore(t)
	ctx := context.Background()
	_ = store.InsertBatch(ctx, sampleEntries())

	page, total, _ := store.Query(ctx, journalv1.Filter{Limit: 2})
	if total != 3 || len(page) != 2 || page[0].ID != "3" {
		t.Fatalf("page 1: %d/%d", len(page), total)
	}
	page, total, _ = store.Query(ctx, journalv1.Filter{Limit: 2, Offset: 2})
	if total != 3 || len(page) != 1 || page[0].ID != "1" {
		t.Fatalf("page 2: %d/%d", len(page), total)
	}
}

func TestVerbCounts(t *testing.T) {
	store := openStore(t)
	ctx := context.Background()
	_ = store.InsertBatch(ctx, sampleEntries())

	counts, err := store.VerbCounts(ctx, journalv1.Filter{})
	if err != nil {
		t.Fatal(err)
	}
	if counts["Ps"] != 2 || counts["Download"] != 1 {
		t.Fatalf("counts: %v", counts)
	}
	counts, _ = store.VerbCounts(ctx, journalv1.Filter{ConnectionID: "h:2"})
	if counts["Download"] != 1 || counts["Ps"] != 0 {
		t.Fatalf("filtered counts: %v", counts)
	}
}

func TestReopenPersists(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()
	store, err := NewSQLiteStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	_ = store.InsertBatch(ctx, sampleEntries())
	_ = store.Close()

	reopened, err := NewSQLiteStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	_, total, _ := reopened.Query(ctx, journalv1.Filter{})
	if total != 3 {
		t.Fatalf("after reopen: %d", total)
	}
}

func TestQuerySearch(t *testing.T) {
	store := openStore(t)
	ctx := context.Background()
	entries := []journalv1.Entry{
		{ID: "s1", Time: 1000, Verb: "Ps", CommandLine: "ps aux", Hostname: "webserver", Status: "ok"},
		{ID: "s2", Time: 2000, Verb: "Download", CommandLine: "download /etc/passwd", Hostname: "webserver", Status: "ok"},
		{ID: "s3", Time: 3000, Verb: "Ps", CommandLine: "ps", Hostname: "dbserver", Status: "ok"},
		{ID: "s4", Time: 4000, Verb: "Execute", CommandLine: "execute whoami", Hostname: "dbserver", Status: "ok"},
	}
	_ = store.InsertBatch(ctx, entries)

	entries, total, _ := store.Query(ctx, journalv1.Filter{Search: "aux"})
	if total != 1 || entries[0].ID != "s1" {
		t.Fatalf("search 'aux': got %d, want 1 (id %s)", total, entries[0].ID)
	}

	_, total, _ = store.Query(ctx, journalv1.Filter{Search: "dbserver"})
	if total != 2 {
		t.Fatalf("search 'dbserver': got %d, want 2", total)
	}

	_, total, _ = store.Query(ctx, journalv1.Filter{Search: "Ps"})
	if total != 2 {
		t.Fatalf("search 'Ps': got %d, want 2", total)
	}

	_, total, _ = store.Query(ctx, journalv1.Filter{Search: "nonexistent"})
	if total != 0 {
		t.Fatalf("search 'nonexistent': got %d, want 0", total)
	}
}

func TestTimeSeries(t *testing.T) {
	store := openStore(t)
	ctx := context.Background()
	now := time.Now().UnixMilli()
	hourMs := int64(3600 * 1000)
	entries := []journalv1.Entry{
		{ID: "ts1", Time: now, Verb: "Ps", ActorKind: "operator", Status: "ok"},
		{ID: "ts2", Time: now, Verb: "Ps", ActorKind: "operator", Status: "ok"},
		{ID: "ts3", Time: now + hourMs, Verb: "Ps", ActorKind: "operator", Status: "ok"},
		{ID: "ts4", Time: now + hourMs, Verb: "Download", ActorKind: "operator", Status: "ok"},
		{ID: "ts5", Time: now + hourMs, Verb: "Ps", ActorKind: "automation", Status: "error", Err: "x"},
		{ID: "ts6", Time: now + 2*hourMs, Verb: "Execute", ActorKind: "operator", Status: "ok"},
	}
	_ = store.InsertBatch(ctx, entries)

	buckets, err := store.TimeSeries(ctx, journalv1.TimeSeriesFilter{BucketSeconds: 3600})
	if err != nil {
		t.Fatal(err)
	}
	if len(buckets) == 0 {
		t.Fatal("expected buckets, got none")
	}

	var total int64
	for _, b := range buckets {
		total += b.Count
	}
	if total != 6 {
		t.Fatalf("total count across buckets: %d, want 6", total)
	}

	buckets, _ = store.TimeSeries(ctx, journalv1.TimeSeriesFilter{BucketSeconds: 3600, Verb: "Download"})
	if len(buckets) != 1 || buckets[0].Count != 1 {
		t.Fatalf("verb filter: %d buckets, count %d", len(buckets), buckets[0].Count)
	}

	buckets, err = store.TimeSeries(ctx, journalv1.TimeSeriesFilter{BucketSeconds: 0})
	if err != nil {
		t.Fatalf("zero bucket should not error: %v", err)
	}
	var ct int64
	for _, b := range buckets {
		ct += b.Count
	}
	if ct != 6 {
		t.Fatalf("zero bucket clamp: got %d, want 6", ct)
	}
}

func TestQueryVerbsFilter(t *testing.T) {
	store := openStore(t)
	ctx := context.Background()
	entries := []journalv1.Entry{
		{ID: "v1", Time: 1000, Verb: "Ps", Status: "ok"},
		{ID: "v2", Time: 2000, Verb: "Download", Status: "ok"},
		{ID: "v3", Time: 3000, Verb: "Execute", Status: "ok"},
		{ID: "v4", Time: 4000, Verb: "Ps", Status: "error", Err: "x"},
	}
	_ = store.InsertBatch(ctx, entries)

	_, total, _ := store.Query(ctx, journalv1.Filter{Verbs: []string{"Ps"}})
	if total != 2 {
		t.Fatalf("Verbs=['Ps']: got %d, want 2", total)
	}

	_, total, _ = store.Query(ctx, journalv1.Filter{Verbs: []string{"Ps", "Execute"}})
	if total != 3 {
		t.Fatalf("Verbs=['Ps','Execute']: got %d, want 3", total)
	}

	_, total, _ = store.Query(ctx, journalv1.Filter{Verbs: []string{}})
	if total != 4 {
		t.Fatalf("empty Verbs: got %d, want 4 (no filter)", total)
	}

	_, total, _ = store.Query(ctx, journalv1.Filter{Verbs: []string{"Ps"}})
	if total != 2 {
		t.Fatalf("Verbs with no other filter: got %d, want 2", total)
	}
}
