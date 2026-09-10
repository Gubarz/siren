package jsonstore

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
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
		// Callers treat a failed load as empty state, so the next save would
		// overwrite the only copy. Move it aside first, and keep reporting the
		// error so the failure stays visible.
		Quarantine(s.path)
		return zero, false, err
	}
	return val, true, nil
}

// Quarantine moves a file that could not be decoded out of the way as
// <path>.corrupt-<nanos>, so a later save cannot destroy it. It returns the new
// name, or "" if the file could not be moved.
func Quarantine(path string) string {
	dest := fmt.Sprintf("%s.corrupt-%d", path, time.Now().UnixNano())
	if err := os.Rename(path, dest); err != nil {
		log.Printf("jsonstore: could not quarantine %s: %v", path, err)
		return ""
	}
	log.Printf("jsonstore: %s was not valid JSON; moved to %s", path, dest)
	return dest
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
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	// A fixed "<path>.tmp" is shared by every writer, so a leftover temp file,
	// a stale directory, or the second process that shares the data dir (the
	// GUI and cmd/headless) collides with it. CreateTemp also supplies the
	// 0600 mode this needs.
	tmp, err := os.CreateTemp(dir, filepath.Base(s.path)+".*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	// Flush the contents before the rename. Without this a crash can leave the
	// new name pointing at a file whose data never reached the disk.
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpName, s.path); err != nil {
		return err
	}
	syncDir(dir)
	return nil
}

// syncDir flushes the directory entry so the rename itself survives a crash.
// Platforms that will not open a directory for this (Windows does not) are
// skipped: the contents are already synced by the time it is called.
func syncDir(dir string) {
	d, err := os.Open(dir)
	if err != nil {
		return
	}
	_ = d.Sync()
	_ = d.Close()
}
