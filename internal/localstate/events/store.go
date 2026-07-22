package events

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/bishopfox/sliver/client/assets"
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

func New() *Store {
	s := &Store{
		path: filepath.Join(assets.GetRootAppDir(), "gui-events.json"),
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
		s.events = nil
	}
	s.assignMissingSeq()
}

// assignMissingSeq backfills seqs for events persisted before ack support
// (all Seq == 0), so every stored event has a stable identity. Numbering in
// slice order is deterministic across restarts, keeping ack state valid.
func (s *Store) assignMissingSeq() {
	needsSeq := false
	for i := range s.events {
		if s.events[i].Seq == 0 {
			needsSeq = true
			break
		}
	}
	if needsSeq {
		for i := range s.events {
			s.events[i].Seq = int64(i + 1)
		}
	}
	s.nextSeq = int64(len(s.events))
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
