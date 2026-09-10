package events

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"sync"

	"siren/internal/localstate/jsonstore"
)

const maxStored = 10000

const persistInterval = 100

type StoredEvent struct {
	Type      string `json:"type"`
	SessionID string `json:"sessionID,omitempty"`
	Hostname  string `json:"hostname,omitempty"`
	Username  string `json:"username,omitempty"`
	Job       string `json:"job,omitempty"`
	Data      string `json:"data,omitempty"`
	Time      int64  `json:"time"`
	Seq       int64  `json:"seq"`
	Acked     bool   `json:"acked,omitempty"`
}

type Store struct {
	mu      sync.Mutex
	events  []StoredEvent
	path    string
	dirty   int
	nextSeq int64
}

func New(rootDir string) *Store {
	s := &Store{
		path: filepath.Join(rootDir, "gui-events.json"),
	}
	s.load()
	return s
}

func (s *Store) Append(ev StoredEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.nextSeq++
	ev.Seq = s.nextSeq
	s.events = append(s.events, ev)
	if len(s.events) > maxStored {
		s.events = s.events[len(s.events)-maxStored:]
	}

	s.dirty++
	if s.dirty >= persistInterval {
		s.persistLocked()
	}
}

// SetAcked marks events acknowledged (or back to unread) by seq. It returns
// how many matched, and persists immediately so ack state survives crashes.
func (s *Store) SetAcked(seqs []int64, acked bool) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	want := make(map[int64]struct{}, len(seqs))
	for _, seq := range seqs {
		want[seq] = struct{}{}
	}
	updated := 0
	for i := range s.events {
		if _, ok := want[s.events[i].Seq]; ok {
			s.events[i].Acked = acked
			updated++
		}
	}
	if updated > 0 {
		s.persistLocked()
	}
	return updated
}

func (s *Store) Query(since int64, limit int) []StoredEvent {
	s.mu.Lock()
	defer s.mu.Unlock()

	if limit <= 0 || limit > maxStored {
		limit = maxStored
	}

	var out []StoredEvent
	for i := len(s.events) - 1; i >= 0 && len(out) < limit; i-- {
		if s.events[i].Time >= since {
			out = append(out, s.events[i])
		}
	}

	reverse(out)
	return out
}

func (s *Store) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.persistLocked()
}

func (s *Store) load() {
	data, err := os.ReadFile(s.path)
	if err != nil {
		return
	}
	if err := json.Unmarshal(data, &s.events); err != nil {
		log.Printf("events: could not decode: %v", err)
		// The next persist would overwrite the only copy of the history.
		jsonstore.Quarantine(s.path)
		s.events = nil
	}
	s.assignMissingSeq()
}

// assignMissingSeq gives every stored event a stable identity. Events persisted
// before ack support carry Seq == 0 and are numbered in slice order, which is
// deterministic across restarts.
//
// nextSeq then continues above the highest seq in use rather than at
// len(events). Once the ring has wrapped, the surviving events hold the newest
// seqs and those exceed the slice length, so counting from the length re-issued
// seqs that were still in the list and SetAcked matched the wrong events.
func (s *Store) assignMissingSeq() {
	next := s.highestSeq() + 1
	for i := range s.events {
		if s.events[i].Seq == 0 {
			s.events[i].Seq = next
			next++
		}
	}
	s.nextSeq = next - 1
}

func (s *Store) highestSeq() int64 {
	var highest int64
	for i := range s.events {
		if s.events[i].Seq > highest {
			highest = s.events[i].Seq
		}
	}
	return highest
}

func (s *Store) persistLocked() {
	data, err := json.Marshal(s.events)
	if err != nil {
		log.Printf("events: could not encode: %v", err)
		return
	}
	temp := s.path + ".tmp"
	if err := os.WriteFile(temp, data, 0o600); err != nil {
		log.Printf("events: could not write: %v", err)
		return
	}
	if err := os.Rename(temp, s.path); err != nil {
		log.Printf("events: could not rename: %v", err)
	}
	s.dirty = 0
}

func reverse(s []StoredEvent) {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
}
