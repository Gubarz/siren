package journal

import (
	"context"
	"testing"
	"time"

	journalv1 "sliver-gui/internal/journal"
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
