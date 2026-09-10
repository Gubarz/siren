// Package downloadhistory persists the operator's file download history.
//
// Sliver's teamserver has no download-history RPC, so this is client-side state
// kept next to the other local stores. It is not scoped per teamserver: the file
// name is unchanged from the original implementation so an existing history
// keeps working.
package downloadhistory

import (
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"siren/internal/localstate/jsonstore"
)

const (
	persistPrefix = "download_history"
	maxRecords    = 1000
)

// Record is one entry in the download history.
type Record struct {
	ID          string `json:"id"`
	SessionID   string `json:"sessionID"`
	RemotePath  string `json:"remotePath"`
	LocalPath   string `json:"localPath"`
	Size        int64  `json:"size"`
	IsDirectory bool   `json:"isDirectory"`
	Timestamp   string `json:"timestamp"`
	Status      string `json:"status"` // "completed", "failed", "in_progress"
	Error       string `json:"error,omitempty"`
}

// Store persists the history in <dir>/download_history.json.
//
// Each operation reads and rewrites the file instead of caching records in
// memory. The GUI and cmd/headless share a data directory, and a cached copy
// would let one process save over what the other just wrote.
type Store struct {
	mu    sync.Mutex
	store *jsonstore.ScopedStore[[]Record]
}

// New builds a store rooted at dir.
func New(dir string) *Store {
	return &Store{store: jsonstore.New[[]Record](dir, persistPrefix)}
}

// Path returns the file the history is written to.
func (s *Store) Path() string { return s.store.Path() }

// Add prepends rec, trims the history to the newest maxRecords entries, and
// returns the record ID. A record arriving without an ID or timestamp is given
// both.
//
// A history that cannot be read is treated as empty: jsonstore moves an
// undecodable file aside first, so the write replaces it rather than failing
// forever.
func (s *Store) Add(rec Record) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if rec.ID == "" {
		rec.ID = fmt.Sprintf("%d", time.Now().UnixNano())
	}
	if strings.TrimSpace(rec.Timestamp) == "" {
		rec.Timestamp = time.Now().UTC().Format(time.RFC3339Nano)
	}

	records, _ := s.loadLocked()
	records = append([]Record{rec}, records...)
	if len(records) > maxRecords {
		records = records[:maxRecords]
	}
	return rec.ID, s.store.Save(records)
}

// Update applies the terminal state of a download to its record. An unknown ID
// writes nothing.
func (s *Store) Update(id, status string, size int64, errStr string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	records, err := s.loadLocked()
	if err != nil {
		return err
	}
	for i := range records {
		if records[i].ID != id {
			continue
		}
		records[i].Status = status
		if size > 0 {
			records[i].Size = size
		}
		records[i].Error = errStr
		return s.store.Save(records)
	}
	return nil
}

// All returns every record, newest first.
func (s *Store) All() ([]Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	records, err := s.loadLocked()
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return []Record{}, nil
	}
	out := make([]Record, len(records))
	copy(out, records)
	return out, nil
}

// Get returns the records for a session and remote path. An empty argument
// matches everything, so Get("", "") returns the whole history.
func (s *Store) Get(sessionID, remotePath string) ([]Record, error) {
	records, err := s.All()
	if err != nil {
		return nil, err
	}
	cleanRemote := cleanPath(remotePath)
	if sessionID == "" && remotePath == "" {
		return records, nil
	}

	out := make([]Record, 0, len(records))
	for _, rec := range records {
		if sessionID != "" && rec.SessionID != sessionID {
			continue
		}
		if remotePath != "" && cleanPath(rec.RemotePath) != cleanRemote {
			continue
		}
		out = append(out, rec)
	}
	return out, nil
}

// Clear drops the records for a session and remote path. Both empty clears the
// whole history.
func (s *Store) Clear(sessionID, remotePath string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if sessionID == "" && remotePath == "" {
		return s.store.Save([]Record{})
	}

	records, err := s.loadLocked()
	if err != nil {
		return err
	}
	cleanRemote := cleanPath(remotePath)
	kept := make([]Record, 0, len(records))
	for _, rec := range records {
		matchSession := sessionID == "" || rec.SessionID == sessionID
		matchRemote := remotePath == "" || cleanPath(rec.RemotePath) == cleanRemote
		if !matchSession || !matchRemote {
			kept = append(kept, rec)
		}
	}
	return s.store.Save(kept)
}

func (s *Store) loadLocked() ([]Record, error) {
	records, _, err := s.store.Load()
	return records, err
}

// cleanPath normalises a remote path for comparison. The path belongs to the
// target's filesystem, so it is compared case-insensitively.
func cleanPath(p string) string {
	if p == "" {
		return ""
	}
	return strings.ToLower(filepath.Clean(p))
}
