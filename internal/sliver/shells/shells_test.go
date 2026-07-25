package shells

import (
	"context"
	"strings"
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

func (f *fakeJournalStore) TimeSeries(_ context.Context, _ journal.TimeSeriesFilter) ([]journal.TimeBucket, error) {
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

func TestJournalShellInputBuffersPartials(t *testing.T) {
	store := newFakeJournalStore(t)
	svc := journal.NewService(store, nil)
	defer svc.Close()
	s := &Service{journal: svc}
	shell := &guiShell{info: ShellInfo{ID: "1", SessionID: "sess-1"}}

	s.journalShellInput(shell, "who")
	s.journalShellInput(shell, "ami\nls /\r\npartial")
	entries := waitForStoreEntries(t, store, 2)
	if entries[0].CommandLine != "whoami" || entries[1].CommandLine != "ls /" {
		t.Fatalf("lines: %+v", entries)
	}
	if entries[0].Verb != "ShellInput" || entries[0].TargetID != "sess-1" {
		t.Fatalf("entry: %+v", entries[0])
	}
	if !strings.HasSuffix(shell.inputTail, "partial") {
		t.Fatalf("tail: %q", shell.inputTail)
	}
}
