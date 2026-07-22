// Package casefile groups the loot, credentials, hosts, and agent context
// an operator collects during an engagement into one exportable record.
// Sliver's teamserver has no case concept — this is a client-side rollup
// that reads from the same source data the panels already show.
package casefile

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/bishopfox/sliver/client/assets"
	"github.com/google/uuid"
)

const casesDir = "gui-cases"

// Record bundles every ID the operator has tagged as belonging to this
// engagement. The IDs point at server-side records (loot, credentials,
// etc.) so the case stays small; export time we re-fetch the payloads.
type Record struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Notes       string   `json:"notes"`
	CreatedAt   int64    `json:"createdAt"`
	UpdatedAt   int64    `json:"updatedAt"`
	AgentIDs    []string `json:"agentIds"`
	LootIDs     []string `json:"lootIds"`
	CredIDs     []string `json:"credIds"`
	HostIDs     []string `json:"hostIds"`
	CanaryIDs   []string `json:"canaryIds"`
}

type Service struct {
	mu    sync.RWMutex
	root  string
	cases map[string]*Record
}

func New() *Service {
	s := &Service{
		root:  filepath.Join(assets.GetRootAppDir(), casesDir),
		cases: map[string]*Record{},
	}
	_ = os.MkdirAll(s.root, 0o700)
	_ = s.loadAll()
	return s
}

func (s *Service) Close() {}

func (s *Service) SetServer(host string, port uint32) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.root = filepath.Join(assets.GetRootAppDir(), fmt.Sprintf("%s-%s_%d", casesDir, host, port))
	_ = os.MkdirAll(s.root, 0o700)
	s.cases = map[string]*Record{}
	_ = s.loadAllLocked()
}

func (s *Service) loadAll() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.loadAllLocked()
}

func (s *Service) loadAllLocked() error {
	entries, err := os.ReadDir(s.root)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		c, err := s.readCase(filepath.Join(s.root, entry.Name()))
		if err != nil {
			continue
		}
		s.cases[c.ID] = c
	}
	return nil
}

func (s *Service) readCase(path string) (*Record, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var c Record
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

func (s *Service) persistLocked(c *Record) error {
	normalizeCase(c)
	c.UpdatedAt = time.Now().UnixMilli()
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	path := filepath.Join(s.root, c.ID+".json")
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// List returns cases newest-updated first.
func (s *Service) List() []*Record {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Record, 0, len(s.cases))
	for _, c := range s.cases {
		out = append(out, cloneCase(c))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].UpdatedAt > out[j].UpdatedAt })
	return out
}

func (s *Service) Get(id string) *Record {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if c, ok := s.cases[id]; ok {
		return cloneCase(c)
	}
	return nil
}

// Create allocates a case id and persists an empty record.
func (s *Service) Create(name, description string) (*Record, error) {
	if name == "" {
		return nil, fmt.Errorf("case name is required")
	}
	now := time.Now().UnixMilli()
	c := &Record{
		ID:          uuid.NewString(),
		Name:        name,
		Description: description,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	normalizeCase(c)
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.persistLocked(c); err != nil {
		return nil, err
	}
	s.cases[c.ID] = c
	return cloneCase(c), nil
}

// Update patches metadata (name, description, notes) — collections are
// managed through Add/Remove for atomicity.
func (s *Service) Update(id, name, description, notes string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.cases[id]
	if !ok {
		return fmt.Errorf("case %s not found", id)
	}
	if name != "" {
		c.Name = name
	}
	c.Description = description
	c.Notes = notes
	return s.persistLocked(c)
}

func (s *Service) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.cases[id]; !ok {
		return fmt.Errorf("case %s not found", id)
	}
	if err := os.Remove(filepath.Join(s.root, id+".json")); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	delete(s.cases, id)
	return nil
}

func cloneCase(c *Record) *Record {
	out := *c
	normalizeCase(&out)
	return &out
}

func cloneStringSlice(in []string) []string {
	if in == nil {
		return []string{}
	}
	return append([]string(nil), in...)
}

func normalizeCase(c *Record) {
	c.AgentIDs = cloneStringSlice(c.AgentIDs)
	c.LootIDs = cloneStringSlice(c.LootIDs)
	c.CredIDs = cloneStringSlice(c.CredIDs)
	c.HostIDs = cloneStringSlice(c.HostIDs)
	c.CanaryIDs = cloneStringSlice(c.CanaryIDs)
}
