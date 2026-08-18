package jsonstore

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// ScopedStore manages atomic JSON file persistence scoped optionally per teamserver.
type ScopedStore[T any] struct {
	mu     sync.RWMutex
	dir    string
	prefix string
	path   string
}

// New creates a ScopedStore targeting the default path <dir>/<prefix>.json.
func New[T any](rootDir, prefix string) *ScopedStore[T] {
	return &ScopedStore[T]{
		dir:    rootDir,
		prefix: prefix,
		path:   filepath.Join(rootDir, prefix+".json"),
	}
}

// SetServer scopes the store's file path to <dir>/<prefix>-<host>_<port>.json.
func (s *ScopedStore[T]) SetServer(host string, port uint32) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.path = filepath.Join(s.dir, fmt.Sprintf("%s-%s_%d.json", s.prefix, host, port))
}

// Path returns the current file path.
func (s *ScopedStore[T]) Path() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.path
}

// Load reads and unmarshals the JSON file. If the file does not exist,
// it returns (zero, false, nil).
func (s *ScopedStore[T]) Load() (T, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.loadLocked()
}

func (s *ScopedStore[T]) loadLocked() (T, bool, error) {
	var zero T
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return zero, false, nil
	}
	if err != nil {
		return zero, false, err
	}
	var val T
	if err := json.Unmarshal(data, &val); err != nil {
		return zero, false, err
	}
	return val, true, nil
}

// Save marshals the value with indentation and atomically writes it to disk.
func (s *ScopedStore[T]) Save(val T) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.saveLocked(val)
}

func (s *ScopedStore[T]) saveLocked(val T) error {
	data, err := json.MarshalIndent(val, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}
