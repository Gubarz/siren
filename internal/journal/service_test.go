package journal

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"sliver-gui/internal/bus"
)

type fakeStore struct {
	mu      sync.Mutex
	batches [][]Entry
}

func (f *fakeStore) InsertBatch(_ context.Context, entries []Entry) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.batches = append(f.batches, append([]Entry(nil), entries...))
	return nil
}

func (f *fakeStore) Query(_ context.Context, _ Filter) ([]Entry, int, error) {
	return nil, 0, nil
}

func (f *fakeStore) VerbCounts(_ context.Context, _ Filter) (map[string]int64, error) {
	return nil, nil
}

func (f *fakeStore) Close() error { return nil }

func (f *fakeStore) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	n := 0
	for _, b := range f.batches {
		n += len(b)
	}
	return n
}

func waitForCond(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", what)
}

func TestRecordFlushesToStore(t *testing.T) {
	store := &fakeStore{}
	svc := NewService(store, nil)
	defer svc.Close()
	for i := 0; i < 5; i++ {
		svc.Record(Entry{Verb: "Ps"})
	}
	waitForCond(t, "flush", func() bool { return store.count() == 5 })
}

func TestRecordFillsIDAndTime(t *testing.T) {
	store := &fakeStore{}
	svc := NewService(store, nil)
	defer svc.Close()
	svc.Record(Entry{Verb: "Ps"})
	waitForCond(t, "flush", func() bool { return store.count() == 1 })
	e := store.batches[0][0]
	if e.ID == "" || e.Time == 0 {
		t.Fatalf("ID/Time not filled: %+v", e)
	}
}

func TestPublishesActionRecordedPerEntry(t *testing.T) {
	store := &fakeStore{}
	b := bus.New()
	received := make(chan Entry, 4)
	b.Subscribe([]string{"journal.action-recorded"}, func(ev bus.Event) {
		received <- ev.Payload.(Entry)
	})
	svc := NewService(store, b)
	defer svc.Close()
	svc.Record(Entry{Verb: "Ps", ConnectionID: "h:1"})
	select {
	case e := <-received:
		if e.Verb != "Ps" {
			t.Fatalf("payload: %+v", e)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("no journal.action-recorded event")
	}
}

func TestCloseDrainsQueue(t *testing.T) {
	store := &fakeStore{}
	svc := NewService(store, nil)
	for i := 0; i < 50; i++ {
		svc.Record(Entry{Verb: "Ps"})
	}
	if err := svc.Close(); err != nil {
		t.Fatal(err)
	}
	if store.count() != 50 {
		t.Fatalf("after Close: %d, want 50", store.count())
	}
}

func TestDisabledMode(t *testing.T) {
	svc := NewService(nil, nil)
	svc.Record(Entry{Verb: "Ps"}) // must not panic
	if _, _, err := svc.Query(context.Background(), Filter{}); !errors.Is(err, ErrDisabled) {
		t.Fatalf("Query: %v", err)
	}
	if _, err := svc.VerbCounts(context.Background(), Filter{}); !errors.Is(err, ErrDisabled) {
		t.Fatalf("VerbCounts: %v", err)
	}
	if err := svc.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestQueryProxiesToStore(t *testing.T) {
	store := &fakeStore{}
	svc := NewService(store, nil)
	defer svc.Close()
	if _, total, err := svc.Query(context.Background(), Filter{Verb: "Ps"}); err != nil || total != 0 {
		t.Fatalf("Query: %d %v", total, err)
	}
}
