package files

import (
	"os"
	"path/filepath"
	"testing"
)

func TestHistoryStore(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "download_history_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "download_history.json")
	store := NewHistoryStore(dbPath)

	// Test Add & Query
	rec1 := DownloadRecord{
		SessionID:   "session-1",
		RemotePath:  "/tmp/test.txt",
		LocalPath:   "/home/user/Downloads/test.txt",
		Size:        1024,
		IsDirectory: false,
		Status:      "in_progress",
	}

	id1 := store.AddRecord(rec1)
	if id1 == "" {
		t.Fatalf("expected non-empty ID")
	}

	history := store.GetHistory("session-1", "/tmp/test.txt")
	if len(history) != 1 {
		t.Fatalf("expected 1 record, got %d", len(history))
	}
	if history[0].Status != "in_progress" {
		t.Fatalf("expected status 'in_progress', got '%s'", history[0].Status)
	}

	// Test Update
	store.UpdateRecord(id1, "completed", 2048, "")
	history = store.GetHistory("session-1", "/tmp/test.txt")
	if len(history) != 1 {
		t.Fatalf("expected 1 record, got %d", len(history))
	}
	if history[0].Status != "completed" || history[0].Size != 2048 {
		t.Fatalf("expected updated status 'completed' and size 2048, got status '%s' size %d", history[0].Status, history[0].Size)
	}

	// Test Persistence (reload from file)
	store2 := NewHistoryStore(dbPath)
	all := store2.GetAllHistory()
	if len(all) != 1 {
		t.Fatalf("expected 1 persisted record, got %d", len(all))
	}
	if all[0].ID != id1 {
		t.Fatalf("expected ID %s, got %s", id1, all[0].ID)
	}

	// Test Add second record for different path
	rec2 := DownloadRecord{
		SessionID:   "session-1",
		RemotePath:  "/tmp/folder",
		LocalPath:   "/home/user/Downloads/folder.tar",
		Size:        4096,
		IsDirectory: true,
		Status:      "failed",
		Error:       "RPC timeout",
		Timestamp:   nowString(),
	}
	store.AddRecord(rec2)

	all = store.GetAllHistory()
	if len(all) != 2 {
		t.Fatalf("expected 2 records in total, got %d", len(all))
	}

	// Test Clear by remote path
	store.ClearHistory("session-1", "/tmp/test.txt")
	history = store.GetHistory("session-1", "/tmp/test.txt")
	if len(history) != 0 {
		t.Fatalf("expected 0 records after clearing /tmp/test.txt, got %d", len(history))
	}
	all = store.GetAllHistory()
	if len(all) != 1 {
		t.Fatalf("expected 1 record remaining, got %d", len(all))
	}

	// Test Clear all
	store.ClearHistory("", "")
	all = store.GetAllHistory()
	if len(all) != 0 {
		t.Fatalf("expected 0 records after clearing all, got %d", len(all))
	}
}
