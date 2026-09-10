package events

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// Once the ring has wrapped, the stored events hold seqs larger than the slice
// length. Deriving nextSeq from the length re-issued seqs that were still in
// the list, so acking one event also acked an unrelated one.
func TestAppendAfterLoadDoesNotReuseASeq(t *testing.T) {
	dir := t.TempDir()
	seeded := []StoredEvent{
		{Type: "a", Time: 1, Seq: 2},
		{Type: "b", Time: 2, Seq: 3},
		{Type: "c", Time: 3, Seq: 4},
	}
	data, err := json.Marshal(seeded)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "gui-events.json"), data, 0o600); err != nil {
		t.Fatal(err)
	}

	s := New(dir)
	s.Append(StoredEvent{Type: "d", Time: 4, SessionID: "newest"})

	events := s.Query(0, 0)
	seen := make(map[int64]int, len(events))
	for _, ev := range events {
		seen[ev.Seq]++
	}
	for seq, count := range seen {
		if count > 1 {
			t.Fatalf("seq %d is used by %d stored events, want unique seqs", seq, count)
		}
	}

	latest := events[len(events)-1]
	if latest.SessionID != "newest" {
		t.Fatalf("newest stored event = %+v, want the appended one", latest)
	}
	if acked := s.SetAcked([]int64{latest.Seq}, true); acked != 1 {
		t.Fatalf("SetAcked matched %d events, want exactly the newest", acked)
	}
}

// A file written before ack support has no seqs at all and still gets 1..n.
func TestLoadNumbersLegacyEventsFromOne(t *testing.T) {
	dir := t.TempDir()
	legacy := []StoredEvent{
		{Type: "a", Time: 1},
		{Type: "b", Time: 2},
	}
	data, err := json.Marshal(legacy)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "gui-events.json"), data, 0o600); err != nil {
		t.Fatal(err)
	}

	s := New(dir)
	events := s.Query(0, 0)
	if len(events) != 2 {
		t.Fatalf("loaded %d events, want 2", len(events))
	}
	if events[0].Seq != 1 || events[1].Seq != 2 {
		t.Fatalf("legacy seqs = %d,%d, want 1,2", events[0].Seq, events[1].Seq)
	}

	s.Append(StoredEvent{Type: "c", Time: 3})
	if got := s.Query(0, 0); got[len(got)-1].Seq != 3 {
		t.Fatalf("appended seq = %d, want 3", got[len(got)-1].Seq)
	}
}
