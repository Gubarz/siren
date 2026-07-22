package automation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"sliver-gui/internal/automation"
)

type JSONStore struct {
	mu   sync.Mutex
	dir  string
	path string
}

const stateFilename = "gui-automation.json"

func New(rootDir string) *JSONStore {
	return &JSONStore{
		dir:  rootDir,
		path: filepath.Join(rootDir, stateFilename),
	}
}

func (s *JSONStore) Load(_ context.Context) (*automation.State, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.loadLocked()
}

func (s *JSONStore) loadLocked() (*automation.State, error) {
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return &automation.State{}, nil
	}
	if err != nil {
		return nil, err
	}
	var state automation.State
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}
	return &state, nil
}

func (s *JSONStore) Save(_ context.Context, state *automation.State) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.persistLocked(state)
}

func (s *JSONStore) persistLocked(state *automation.State) error {
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

func (s *JSONStore) SetServer(host string, port uint32) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.path = filepath.Join(s.dir, fmt.Sprintf("gui-automation-%s_%d.json", host, port))
}
