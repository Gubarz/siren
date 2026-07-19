// Package tags stores per-agent operator tags. Sliver's teamserver has no
// tag RPC — this is a client-side persistence layer keyed by agent ID and
// scoped per teamserver profile (host+port), so tags don't leak between
// operators sharing the same GUI install.
package tags

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/bishopfox/sliver/client/assets"
)

const persistFilename = "gui-agent-tags.json"

type Service struct {
	mu   sync.RWMutex
	path string
	// tags maps agent ID -> sorted+dedup'd tag list. A missing key means the
	// agent has no operator tags — a nil slice is preserved on read as an
	// empty list. Notes have their own home in internal/loot; this service
	// only owns tags.
	tags map[string][]string
}

type persisted struct {
	Tags map[string][]string `json:"tags"`
}

func New() *Service {
	s := &Service{
		path: filepath.Join(assets.GetRootAppDir(), persistFilename),
		tags: map[string][]string{},
	}
	if err := s.load(); err != nil {
		// Startup shouldn't fatal on a missing/corrupt file — treat as empty.
		s.tags = map[string][]string{}
	}
	return s
}

func (s *Service) Close() {}

// SetServer scopes persistence per teamserver, matching the automation
// engine's pattern. Missing files start empty on next Set.
func (s *Service) SetServer(host string, port uint32) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.path = filepath.Join(assets.GetRootAppDir(), fmt.Sprintf("gui-agent-tags-%s_%d.json", host, port))
	s.tags = map[string][]string{}
	_ = s.loadLocked()
}

func (s *Service) load() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.loadLocked()
}

func (s *Service) loadLocked() error {
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	var p persisted
	if err := json.Unmarshal(data, &p); err != nil {
		return err
	}
	if p.Tags != nil {
		s.tags = p.Tags
	}
	return nil
}

func (s *Service) persistLocked() error {
	data, err := json.MarshalIndent(persisted{Tags: s.tags}, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

// GetAgentTags returns the tag list for one agent, empty slice if none.
func (s *Service) GetAgentTags(agentID string) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]string, len(s.tags[agentID]))
	copy(out, s.tags[agentID])
	return out
}

// SetAgentTags replaces the tag list for one agent. Empty list deletes.
func (s *Service) SetAgentTags(agentID string, tags []string) error {
	normalized := normalizeTags(tags)
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(normalized) == 0 {
		delete(s.tags, agentID)
	} else {
		s.tags[agentID] = normalized
	}
	return s.persistLocked()
}

// GetAllTags returns every agent's tags in one map — cheap because the
// whole store is in memory. Used to build filter chips + palette entries.
func (s *Service) GetAllTags() map[string][]string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[string][]string, len(s.tags))
	for id, list := range s.tags {
		copyList := make([]string, len(list))
		copy(copyList, list)
		out[id] = copyList
	}
	return out
}

// KnownTags returns the union of every unique tag used across all agents,
// sorted, for populating a filter dropdown.
func (s *Service) KnownTags() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	set := map[string]struct{}{}
	for _, list := range s.tags {
		for _, tag := range list {
			set[tag] = struct{}{}
		}
	}
	out := make([]string, 0, len(set))
	for tag := range set {
		out = append(out, tag)
	}
	sort.Strings(out)
	return out
}

// normalizeTags trims, lowercases (case-fold), drops empties, dedups, sorts.
func normalizeTags(tags []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(tags))
	for _, raw := range tags {
		t := strings.ToLower(strings.TrimSpace(raw))
		if t == "" {
			continue
		}
		if _, ok := seen[t]; ok {
			continue
		}
		seen[t] = struct{}{}
		out = append(out, t)
	}
	sort.Strings(out)
	return out
}
