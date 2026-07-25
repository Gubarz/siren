package beacons

import (
	"context"
	"sync"
	"testing"
	"time"

	"sliver-gui/internal/journal"
)

type fakeJournalStore struct {
	mu      sync.Mutex
	entries []journal.Entry
}

func (f *fakeJournalStore) InsertBatch(_ context.Context, entries []journal.Entry) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.entries = append(f.entries, entries...)
	return nil
}

func (f *fakeJournalStore) Query(_ context.Context, _ journal.Filter) ([]journal.Entry, int, error) {
	return nil, 0, nil
}

func (f *fakeJournalStore) VerbCounts(_ context.Context, _ journal.Filter) (map[string]int64, error) {
	return nil, nil
}

func (f *fakeJournalStore) Close() error { return nil }

func newFakeJournalStore(t *testing.T) *fakeJournalStore {
	t.Helper()
	return &fakeJournalStore{}
}

func waitForStoreEntries(t *testing.T, store *fakeJournalStore, want int) []journal.Entry {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		store.mu.Lock()
		count := len(store.entries)
		if count >= want {
			out := append([]journal.Entry(nil), store.entries...)
			store.mu.Unlock()
			return out
		}
		store.mu.Unlock()
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %d journal entries", want)
	return nil
}

func TestJournalBeaconTaskResultNilSafe(t *testing.T) {
	s := &Service{}
	s.journalBeaconTaskResult(context.Background(), "b1", nil)
}

func TestJournalBeaconTaskResultAppliesOverlay(t *testing.T) {
	store := newFakeJournalStore(t)
	svc := journal.NewService(store, nil)
	defer svc.Close()
	s := &Service{journal: svc}

	ctx := journal.WithContext(context.Background(), journal.Overlay{
		ActorKind: "automation", RuleID: "r1", CorrelationID: "run-9",
	})
	s.journalBeaconTaskResult(ctx, "b1", nil)

	entries := waitForStoreEntries(t, store, 1)
	e := entries[0]
	if e.Verb != "BeaconTaskResult" || e.TargetKind != "beacon" || e.Status != "ok" {
		t.Fatalf("entry: %+v", e)
	}
	if e.ActorKind != "automation" || e.RuleID != "r1" || e.CorrelationID != "run-9" {
		t.Fatalf("overlay: %+v", e)
	}
}
