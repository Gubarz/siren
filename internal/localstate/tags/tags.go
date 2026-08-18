// Package tags stores per-entity operator tags and colors. Sliver's
// teamserver has no tag RPC — this is a client-side persistence layer scoped
// per teamserver profile (host+port), so tags don't leak between operators
// sharing the same GUI install.
package tags

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	"siren/internal/localstate/jsonstore"
)

const persistPrefix = "gui-agent-tags"

type Service struct {
	mu     sync.RWMutex
	store  *jsonstore.ScopedStore[persisted]
	tags   map[string][]string
	colors map[string]string
}

type persisted struct {
	Tags   map[string][]string `json:"tags"`
	Colors map[string]string   `json:"colors,omitempty"`
}

// RowColorNames is the closed palette the GUI offers for agent rows. The
// set is closed (not free-form) so a tampered config file can't smuggle
// arbitrary CSS into the table renderer.
var RowColorNames = []string{"red", "orange", "yellow", "green", "blue", "purple", "pink", "gray"}

func validRowColor(color string) bool {
	for _, name := range RowColorNames {
		if name == color {
			return true
		}
	}
	return false
}

func entityKey(entityType, entityID string) string {
	entityType = strings.ToLower(strings.TrimSpace(entityType))
	entityID = strings.TrimSpace(entityID)
	if entityType == "" || entityID == "" {
		return ""
	}
	return entityType + ":" + entityID
}

func legacyAgentKey(agentID string) string {
	return strings.TrimSpace(agentID)
}

func isAgentEntityKey(key string) (string, bool) {
	agentID, ok := strings.CutPrefix(key, "agent:")
	return agentID, ok && agentID != ""
}

func New(rootDir string) *Service {
	s := &Service{
		store:  jsonstore.New[persisted](rootDir, persistPrefix),
		tags:   map[string][]string{},
		colors: map[string]string{},
	}
	_ = s.load()
	return s
}

func (s *Service) Close() {}

// SetServer scopes persistence per teamserver, matching the automation
// engine's pattern. Missing files start empty on next Set.
func (s *Service) SetServer(host string, port uint32) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.store.SetServer(host, port)
	s.tags = map[string][]string{}
	s.colors = map[string]string{}
	_ = s.loadLocked()
}

func (s *Service) load() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.loadLocked()
}

func (s *Service) loadLocked() error {
	p, ok, err := s.store.Load()
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}
	if p.Tags != nil {
		s.tags = p.Tags
	}
	if p.Colors != nil {
		s.colors = p.Colors
	}
	return nil
}

func (s *Service) persistLocked() error {
	return s.store.Save(persisted{Tags: s.tags, Colors: s.colors})
}

// GetEntityTags returns the tag list for one entity, empty slice if none.
func (s *Service) GetEntityTags(entityType, entityID string) []string {
	key := entityKey(entityType, entityID)
	s.mu.RLock()
	defer s.mu.RUnlock()
	list := s.tags[key]
	if len(list) == 0 && strings.EqualFold(strings.TrimSpace(entityType), "agent") {
		list = s.tags[legacyAgentKey(entityID)]
	}
	out := make([]string, len(list))
	copy(out, list)
	return out
}

// SetEntityTags replaces the tag list for one entity. Empty list deletes.
func (s *Service) SetEntityTags(entityType, entityID string, tags []string) error {
	key := entityKey(entityType, entityID)
	if key == "" {
		return fmt.Errorf("entity type and id are required")
	}
	normalized := normalizeTags(tags)
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(normalized) == 0 {
		delete(s.tags, key)
	} else {
		s.tags[key] = normalized
	}
	if strings.HasPrefix(key, "agent:") {
		delete(s.tags, legacyAgentKey(entityID))
	}
	return s.persistLocked()
}

// GetEntityColor returns the assigned color for one entity.
func (s *Service) GetEntityColor(entityType, entityID string) string {
	key := entityKey(entityType, entityID)
	s.mu.RLock()
	defer s.mu.RUnlock()
	color := s.colors[key]
	if color == "" && strings.EqualFold(strings.TrimSpace(entityType), "agent") {
		color = s.colors[legacyAgentKey(entityID)]
	}
	return color
}

// SetEntityColor assigns a color from RowColorNames to one entity. An empty
// color clears the assignment.
func (s *Service) SetEntityColor(entityType, entityID, color string) error {
	key := entityKey(entityType, entityID)
	if key == "" {
		return fmt.Errorf("entity type and id are required")
	}
	color = strings.ToLower(strings.TrimSpace(color))
	s.mu.Lock()
	defer s.mu.Unlock()
	if color == "" {
		delete(s.colors, key)
		if strings.HasPrefix(key, "agent:") {
			delete(s.colors, legacyAgentKey(entityID))
		}
		return s.persistLocked()
	}
	if !validRowColor(color) {
		return fmt.Errorf("unknown row color %q", color)
	}
	if s.colors == nil {
		s.colors = map[string]string{}
	}
	s.colors[key] = color
	if strings.HasPrefix(key, "agent:") {
		delete(s.colors, legacyAgentKey(entityID))
	}
	return s.persistLocked()
}

// GetAllEntityColors returns every entity's color.
func (s *Service) GetAllEntityColors() map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[string]string, len(s.colors))
	for id, color := range s.colors {
		out[id] = color
	}
	return out
}

// GetAllEntityTags returns every entity's tags — cheap because the whole
// store is in memory. Used to build filter chips + palette entries.
func (s *Service) GetAllEntityTags() map[string][]string {
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

// GetAgentTags returns the tag list for one agent, empty slice if none.
func (s *Service) GetAgentTags(agentID string) []string {
	return s.GetEntityTags("agent", agentID)
}

// SetAgentTags replaces the tag list for one agent. Empty list deletes.
func (s *Service) SetAgentTags(agentID string, tags []string) error {
	return s.SetEntityTags("agent", agentID, tags)
}

// SetAgentColor assigns a row color from RowColorNames to one agent. An
// empty color clears the assignment.
func (s *Service) SetAgentColor(agentID, color string) error {
	return s.SetEntityColor("agent", agentID, color)
}

// GetAllColors returns every agent's row color in one map, mirroring
// GetAllTags for the table renderer.
func (s *Service) GetAllColors() map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := map[string]string{}
	for key, color := range s.colors {
		if agentID, ok := isAgentEntityKey(key); ok {
			out[agentID] = color
		} else if !strings.Contains(key, ":") {
			out[key] = color
		}
	}
	return out
}

// GetAllTags returns every agent's tags in one map.
func (s *Service) GetAllTags() map[string][]string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := map[string][]string{}
	for key, list := range s.tags {
		agentID, ok := isAgentEntityKey(key)
		if !ok && !strings.Contains(key, ":") {
			agentID = key
			ok = true
		}
		if !ok {
			continue
		}
		copyList := make([]string, len(list))
		copy(copyList, list)
		out[agentID] = copyList
	}
	return out
}

// KnownTags returns the union of every unique tag used across all entities,
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
