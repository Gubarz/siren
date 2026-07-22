package events

import (
	"os"
	"path/filepath"
	"testing"
)

func TestQueryReturnsMatchingEventsInChronologicalOrder(t *testing.T) {
	store := newTestStore(t)
	store.Append(StoredEvent{Type: "one", Time: 1})
	store.Append(StoredEvent{Type: "two", Time: 2})
	store.Append(StoredEvent{Type: "three", Time: 3})

	got := store.Query(2, 10)
	want := []StoredEvent{
		{Type: "two", Time: 2, Seq: 2},
		{Type: "three", Time: 3, Seq: 3},
	}
	assertEvents(t, got, want)
}

func TestQueryLimitKeepsMostRecentEvents(t *testing.T) {
	store := newTestStore(t)
	store.Append(StoredEvent{Type: "one", Time: 1})
	store.Append(StoredEvent{Type: "two", Time: 2})
	store.Append(StoredEvent{Type: "three", Time: 3})

	got := store.Query(0, 2)
	want := []StoredEvent{
		{Type: "two", Time: 2, Seq: 2},
		{Type: "three", Time: 3, Seq: 3},
	}
	assertEvents(t, got, want)
}

func TestAppendEvictsOldestEventsPastStorageLimit(t *testing.T) {
	store := newTestStore(t)
	for i := 0; i < maxStored+2; i++ {
		store.Append(StoredEvent{Type: "event", Time: int64(i)})
	}

	got := store.Query(0, maxStored+10)
	if len(got) != maxStored {
		t.Fatalf("Query() returned %d events, want %d", len(got), maxStored)
	}
	if got[0].Time != 2 {
		t.Fatalf("oldest event time = %d, want 2", got[0].Time)
	}
	if got[len(got)-1].Time != maxStored+1 {
		t.Fatalf("newest event time = %d, want %d", got[len(got)-1].Time, maxStored+1)
	}
}

func TestClosePersistsEventsForLaterLoad(t *testing.T) {
	store := newTestStore(t)
	store.Append(StoredEvent{Type: "session-opened", SessionID: "abc", Time: 42})
	store.Close()

	loaded := newTestStore(t)
	loaded.path = store.path
	loaded.load()

	assertEvents(t, loaded.Query(0, 10), []StoredEvent{
		{Type: "session-opened", SessionID: "abc", Time: 42, Seq: 1},
	})
}

func TestSetAckedMarksEventsAndPersists(t *testing.T) {
	store := newTestStore(t)
	store.Append(StoredEvent{Type: "one", Time: 1})
	store.Append(StoredEvent{Type: "two", Time: 2})

	if updated := store.SetAcked([]int64{2}, true); updated != 1 {
		t.Fatalf("SetAcked updated %d events, want 1", updated)
	}
	got := store.Query(0, 10)
	if got[0].Acked || !got[1].Acked {
		t.Fatalf("acked flags = [%v %v], want [false true]", got[0].Acked, got[1].Acked)
	}

	loaded := newTestStore(t)
	loaded.path = store.path
	loaded.load()
	reloaded := loaded.Query(0, 10)
	if !reloaded[1].Acked {
		t.Fatal("ack state did not survive a reload")
	}
}

func TestSetAckedIgnoresUnknownSeqs(t *testing.T) {
	store := newTestStore(t)
	store.Append(StoredEvent{Type: "one", Time: 1})

	if updated := store.SetAcked([]int64{99}, true); updated != 0 {
		t.Fatalf("SetAcked updated %d events, want 0", updated)
	}
}

func TestLoadAssignsSeqsToLegacyEvents(t *testing.T) {
	store := newTestStore(t)
	legacy := `[{"type":"one","time":1},{"type":"two","time":2}]`
	if err := os.WriteFile(store.path, []byte(legacy), 0o600); err != nil {
		t.Fatalf("write legacy file: %v", err)
	}
	store.load()

	got := store.Query(0, 10)
	if got[0].Seq != 1 || got[1].Seq != 2 {
		t.Fatalf("legacy seqs = [%d %d], want [1 2]", got[0].Seq, got[1].Seq)
	}
	store.Append(StoredEvent{Type: "three", Time: 3})
	if got := store.Query(0, 10); got[2].Seq != 3 {
		t.Fatalf("appended seq = %d, want 3", got[2].Seq)
	}
}

func newTestStore(t *testing.T) *Store {
	t.Helper()
	return &Store{path: filepath.Join(t.TempDir(), "events.json")}
}

func assertEvents(t *testing.T, got, want []StoredEvent) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("len(%v) = %d, want %d", got, len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("event[%d] = %#v, want %#v", i, got[i], want[i])
		}
	}
}
