package files

import (
	"os"
	"path/filepath"
	"testing"
)

func TestHistoryStore_AddAndQuery(t *testing.T) {
	tempDir, dbPath := setupHistoryTest(t)
	defer os.RemoveAll(tempDir)

	store := NewHistoryStore(dbPath)
	rec := DownloadRecord{
		SessionID: "session-1", RemotePath: "/tmp/test.txt",
		LocalPath: "/home/user/Downloads/test.txt", Size: 1024,
		IsDirectory: false, Status: "in_progress",
	}
	id := store.AddRecord(rec)
	if id == "" {
		t.Fatalf("expected non-empty ID")
	}
	history := store.GetHistory("session-1", "/tmp/test.txt")
	if len(history) != 1 {
		t.Fatalf("expected 1 record, got %d", len(history))
	}
	if history[0].Status != "in_progress" {
		t.Fatalf("expected status 'in_progress', got '%s'", history[0].Status)
	}
}

func TestHistoryStore_Update(t *testing.T) {
	tempDir, dbPath := setupHistoryTest(t)
	defer os.RemoveAll(tempDir)

	store := NewHistoryStore(dbPath)
	rec := DownloadRecord{
		SessionID: "session-1", RemotePath: "/tmp/test.txt",
		LocalPath: "/home/user/Downloads/test.txt", Size: 1024,
		IsDirectory: false, Status: "in_progress",
	}
	store.AddRecord(rec)

	history := store.GetHistory("session-1", "/tmp/test.txt")
	if len(history) != 1 {
		t.Fatalf("expected 1 record, got %d", len(history))
	}
	store.UpdateRecord(history[0].ID, "completed", 2048, "")
	history = store.GetHistory("session-1", "/tmp/test.txt")
	if history[0].Status != "completed" || history[0].Size != 2048 {
		t.Fatalf("expected status 'completed' and size 2048, got status '%s' size %d",
			history[0].Status, history[0].Size)
	}
}

func TestHistoryStore_Persistence(t *testing.T) {
	tempDir, dbPath := setupHistoryTest(t)
	defer os.RemoveAll(tempDir)

	store := NewHistoryStore(dbPath)
	store.AddRecord(DownloadRecord{
		SessionID: "session-1", RemotePath: "/tmp/persist.txt",
		LocalPath: "/tmp/test", Size: 512,
		IsDirectory: false, Status: "completed",
	})

	store2 := NewHistoryStore(dbPath)
	all := store2.GetAllHistory()
	if len(all) != 1 {
		t.Fatalf("expected 1 persisted record, got %d", len(all))
	}
}

func TestHistoryStore_Clear(t *testing.T) {
	tempDir, dbPath := setupHistoryTest(t)
	defer os.RemoveAll(tempDir)

	store := NewHistoryStore(dbPath)
	store.AddRecord(DownloadRecord{
		SessionID: "session-1", RemotePath: "/tmp/test.txt",
		LocalPath: "/tmp/a", Size: 1024,
		IsDirectory: false, Status: "completed",
	})
	store.AddRecord(DownloadRecord{
		SessionID: "session-1", RemotePath: "/tmp/folder",
		LocalPath: "/tmp/b", Size: 4096,
		IsDirectory: true, Status: "failed", Error: "RPC timeout",
		Timestamp: nowString(),
	})

	all := store.GetAllHistory()
	if len(all) != 2 {
		t.Fatalf("expected 2 records, got %d", len(all))
	}

	store.ClearHistory("session-1", "/tmp/test.txt")
	history := store.GetHistory("session-1", "/tmp/test.txt")
	if len(history) != 0 {
		t.Fatalf("expected 0 records after clearing /tmp/test.txt, got %d", len(history))
	}
	all = store.GetAllHistory()
	if len(all) != 1 {
		t.Fatalf("expected 1 record remaining, got %d", len(all))
	}

	store.ClearHistory("", "")
	all = store.GetAllHistory()
	if len(all) != 0 {
		t.Fatalf("expected 0 records after clearing all, got %d", len(all))
	}
}

func setupHistoryTest(t *testing.T) (string, string) {
	t.Helper()
	tempDir, err := os.MkdirTemp("", "download_history_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	return tempDir, filepath.Join(tempDir, "download_history.json")
}
