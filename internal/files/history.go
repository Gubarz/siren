package files

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/bishopfox/sliver/client/assets"
)

const maxHistoryStored = 1000

type DownloadRecord struct {
	ID          string    `json:"id"`
	SessionID   string    `json:"sessionID"`
	RemotePath  string    `json:"remotePath"`
	LocalPath   string    `json:"localPath"`
	Size        int64     `json:"size"`
	IsDirectory bool      `json:"isDirectory"`
	Timestamp   time.Time `json:"timestamp"`
	Status      string    `json:"status"` // "completed", "failed", "in_progress"
	Error       string    `json:"error,omitempty"`
}

type HistoryStore struct {
	mu      sync.Mutex
	records []DownloadRecord
	path    string
}

func NewHistoryStore(customPath ...string) *HistoryStore {
	storePath := ""
	if len(customPath) > 0 && customPath[0] != "" {
		storePath = customPath[0]
	} else {
		root := assets.GetRootAppDir()
		if root != "" {
			storePath = filepath.Join(root, "download_history.json")
		}
	}
	s := &HistoryStore{
		path: storePath,
	}
	s.load()
	return s
}

func (s *HistoryStore) AddRecord(rec DownloadRecord) string {
	s.mu.Lock()
	defer s.mu.Unlock()

	if rec.ID == "" {
		rec.ID = fmt.Sprintf("%d", time.Now().UnixNano())
	}
	if rec.Timestamp.IsZero() {
		rec.Timestamp = time.Now()
	}

	s.records = append([]DownloadRecord{rec}, s.records...)
	if len(s.records) > maxHistoryStored {
		s.records = s.records[:maxHistoryStored]
	}

	s.persistLocked()
	return rec.ID
}

func (s *HistoryStore) UpdateRecord(id string, status string, size int64, errStr string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range s.records {
		if s.records[i].ID == id {
			s.records[i].Status = status
			if size > 0 {
				s.records[i].Size = size
			}
			s.records[i].Error = errStr
			break
		}
	}
	s.persistLocked()
}

func (s *HistoryStore) GetHistory(sessionID, remotePath string) []DownloadRecord {
	s.mu.Lock()
	defer s.mu.Unlock()

	var out []DownloadRecord
	cleanRemote := cleanPathForCompare(remotePath)

	for _, rec := range s.records {
		if sessionID != "" && rec.SessionID != sessionID {
			continue
		}
		if remotePath != "" && cleanPathForCompare(rec.RemotePath) != cleanRemote {
			continue
		}
		out = append(out, rec)
	}

	if out == nil {
		return []DownloadRecord{}
	}
	return out
}

func (s *HistoryStore) GetAllHistory() []DownloadRecord {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.records == nil {
		return []DownloadRecord{}
	}
	out := make([]DownloadRecord, len(s.records))
	copy(out, s.records)
	return out
}

func (s *HistoryStore) ClearHistory(sessionID, remotePath string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if sessionID == "" && remotePath == "" {
		s.records = nil
		s.persistLocked()
		return
	}

	cleanRemote := cleanPathForCompare(remotePath)
	var kept []DownloadRecord
	for _, rec := range s.records {
		matchSession := sessionID == "" || rec.SessionID == sessionID
		matchRemote := remotePath == "" || cleanPathForCompare(rec.RemotePath) == cleanRemote

		if !(matchSession && matchRemote) {
			kept = append(kept, rec)
		}
	}
	s.records = kept
	s.persistLocked()
}

func cleanPathForCompare(p string) string {
	if p == "" {
		return ""
	}
	return strings.ToLower(filepath.Clean(p))
}

func (s *HistoryStore) load() {
	if s.path == "" {
		return
	}
	data, err := os.ReadFile(s.path)
	if err != nil {
		return
	}
	if err := json.Unmarshal(data, &s.records); err != nil {
		log.Printf("download_history: could not decode: %v", err)
		s.records = nil
	}
}

func (s *HistoryStore) persistLocked() {
	if s.path == "" {
		return
	}
	data, err := json.Marshal(s.records)
	if err != nil {
		log.Printf("download_history: could not encode: %v", err)
		return
	}
	dir := filepath.Dir(s.path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			log.Printf("download_history: could not create dir: %v", err)
			return
		}
	}
	temp := s.path + ".tmp"
	if err := os.WriteFile(temp, data, 0o600); err != nil {
		log.Printf("download_history: could not write: %v", err)
		return
	}
	if err := os.Rename(temp, s.path); err != nil {
		log.Printf("download_history: could not rename: %v", err)
	}
}
