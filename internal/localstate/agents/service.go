// Package agents persists the agents Siren has seen for each teamserver so
// closed sessions stay visible after they go away and across restarts.
// Sliver's teamserver has no such history — this is a client-side record
// keyed by agent ID and scoped per host+port so operators sharing a GUI
// install don't see each other's history.
package agents

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

const (
	persistPrefix = "gui-agents"
	statusActive  = "active"
	statusLost    = "lost"
	statusRemoved = "removed"

	lastSeenPersistInterval = 30 * time.Second
)

// Record is the persisted view of an agent Siren has seen.
type Record struct {
	ID            string `json:"id"`
	Kind          string `json:"kind"`
	Name          string `json:"name"`
	Hostname      string `json:"hostname"`
	Username      string `json:"username"`
	OS            string `json:"os"`
	Arch          string `json:"arch"`
	RemoteAddress string `json:"remoteAddress"`
	Transport     string `json:"transport"`
	Status        string `json:"status"`
	FirstSeen     int64  `json:"firstSeen"`
	LastSeen      int64  `json:"lastSeen"`
	LostAt        int64  `json:"lostAt"`
}

type Service struct {
	mu          sync.RWMutex
	rootDir     string
	root        string
	records     map[string]*Record
	persistedAt map[string]int64
	nowFn       func() time.Time
}

func New(rootDir string) *Service {
	s := &Service{
		rootDir:     rootDir,
		root:        filepath.Join(rootDir, persistPrefix+".json"),
		records:     map[string]*Record{},
		persistedAt: map[string]int64{},
		nowFn:       time.Now,
	}
	_ = os.MkdirAll(s.rootDir, 0o700)
	if err := s.loadAll(); err != nil {
		log.Printf("agents: load known state: %v", err)
	}
	return s
}

func (s *Service) SetServer(host string, port uint32) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.root = filepath.Join(s.rootDir, fmt.Sprintf("%s-%s_%d.json", persistPrefix, host, port))
	_ = os.MkdirAll(s.rootDir, 0o700)
	s.records = map[string]*Record{}
	s.persistedAt = map[string]int64{}
	if err := s.loadAllLocked(); err != nil {
		log.Printf("agents: load known state: %v", err)
	}
}

func (s *Service) loadAll() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.loadAllLocked()
}

// loadAllLocked treats unreadable or corrupt state as empty: history is
// best-effort and must never block a connection.
func (s *Service) loadAllLocked() error {
	data, err := os.ReadFile(s.root)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	var stored []Record
	if err := json.Unmarshal(data, &stored); err != nil {
		return err
	}
	for i := range stored {
		r := stored[i]
		if r.ID == "" {
			continue
		}
		s.records[r.ID] = &r
	}
	return nil
}

// Observe upserts the live list. A removed record is never resurrected, and
// a lost record stays lost because a list fetch can race a close event.
// New records and identity changes persist immediately; LastSeen-only movement
// persists at most once per lastSeenPersistInterval.
func (s *Service) Observe(records []Record) error {
	if len(records) == 0 {
		return nil
	}
	now := s.nowFn().UnixMilli()
	s.mu.Lock()
	defer s.mu.Unlock()
	dirty := false
	touched := make([]string, 0, len(records))
	for i := range records {
		in := records[i]
		if in.ID == "" {
			continue
		}
		existing, ok := s.records[in.ID]
		if !ok {
			r := in
			r.Status = statusActive
			r.FirstSeen = now
			r.LastSeen = now
			r.LostAt = 0
			s.records[in.ID] = &r
			touched = append(touched, in.ID)
			dirty = true
			continue
		}
		if existing.Status == statusRemoved {
			continue
		}
		touched = append(touched, in.ID)
		if existing.Kind != in.Kind ||
			existing.Name != in.Name ||
			existing.Hostname != in.Hostname ||
			existing.Username != in.Username ||
			existing.OS != in.OS ||
			existing.Arch != in.Arch ||
			existing.RemoteAddress != in.RemoteAddress ||
			existing.Transport != in.Transport {
			dirty = true
		}
		existing.Kind = in.Kind
		existing.Name = in.Name
		existing.Hostname = in.Hostname
		existing.Username = in.Username
		existing.OS = in.OS
		existing.Arch = in.Arch
		existing.RemoteAddress = in.RemoteAddress
		existing.Transport = in.Transport
		existing.LastSeen = now
		if now-s.persistedAt[in.ID] >= lastSeenPersistInterval.Milliseconds() {
			dirty = true
		}
	}
	if !dirty {
		return nil
	}
	if err := s.persistLocked(); err != nil {
		return err
	}
	for _, id := range touched {
		s.persistedAt[id] = now
	}
	return nil
}

func (s *Service) MarkLost(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, ok := s.records[id]
	if !ok || r.Status == statusRemoved {
		return nil
	}
	r.Status = statusLost
	r.LostAt = s.nowFn().UnixMilli()
	return s.persistLocked()
}

func (s *Service) Remove(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, ok := s.records[id]
	if !ok {
		return nil
	}
	r.Status = statusRemoved
	r.LastSeen = s.nowFn().UnixMilli()
	return s.persistLocked()
}

// List returns non-removed records newest-seen first, ties by ID, as copies.
func (s *Service) List() []Record {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Record, 0, len(s.records))
	for _, r := range s.records {
		if r.Status == statusRemoved {
			continue
		}
		out = append(out, *r)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].LastSeen != out[j].LastSeen {
			return out[i].LastSeen > out[j].LastSeen
		}
		return out[i].ID < out[j].ID
	})
	return out
}

func (s *Service) persistLocked() error {
	stored := make([]Record, 0, len(s.records))
	for _, r := range s.records {
		stored = append(stored, *r)
	}
	sort.Slice(stored, func(i, j int) bool { return stored[i].ID < stored[j].ID })
	data, err := json.MarshalIndent(stored, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(s.root), 0o700); err != nil {
		return err
	}
	tmp := s.root + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.root)
}
