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
}

type Store struct {
	mu     sync.Mutex
	events []StoredEvent
	path   string
	dirty  int
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

	s.events = append(s.events, ev)
	if len(s.events) > maxStored {
		s.events = s.events[len(s.events)-maxStored:]
	}

	s.dirty++
	if s.dirty >= persistInterval {
		s.persistLocked()
	}
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
