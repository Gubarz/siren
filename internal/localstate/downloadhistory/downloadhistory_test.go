package downloadhistory

import (
	"os"
	"path/filepath"
	"testing"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	return New(t.TempDir())
}

// The file name is unchanged from the hand-rolled store this replaced, so an
// existing download history keeps being read.
func TestPathKeepsTheOriginalFileName(t *testing.T) {
	dir := t.TempDir()

	if got, want := New(dir).Path(), filepath.Join(dir, "download_history.json"); got != want {
		t.Errorf("Path() = %q, want %q", got, want)
	}
}

func TestAddAndQuery(t *testing.T) {
	store := newTestStore(t)

	id, err := store.Add(Record{
		SessionID: "session-1", RemotePath: "/tmp/test.txt",
		LocalPath: "/home/user/Downloads/test.txt", Size: 1024,
		Status: "in_progress",
	})
	if err != nil {
		t.Fatalf("Add() = %v, want success", err)
	}
	if id == "" {
		t.Fatal("Add() returned an empty ID")
	}

	history, err := store.Get("session-1", "/tmp/test.txt")
	if err != nil {
		t.Fatalf("Get() = %v, want success", err)
	}
	if len(history) != 1 {
		t.Fatalf("Get() returned %d records, want 1", len(history))
	}
	if history[0].Status != "in_progress" {
		t.Errorf("Status = %q, want in_progress", history[0].Status)
	}
}

// A caller that does not supply an ID or timestamp still gets usable ones.
func TestAddStampsMissingFields(t *testing.T) {
	store := newTestStore(t)

	id, err := store.Add(Record{SessionID: "session-1", RemotePath: "/tmp/a"})
	if err != nil {
		t.Fatalf("Add() = %v, want success", err)
	}

	history, err := store.Get("session-1", "/tmp/a")
	if err != nil {
		t.Fatalf("Get() = %v, want success", err)
	}
	if history[0].ID != id {
		t.Errorf("stored ID = %q, want the returned %q", history[0].ID, id)
	}
	if history[0].Timestamp == "" {
		t.Error("Timestamp was not stamped")
	}
}

// Remote paths come from the target's filesystem, so lookups are case
// insensitive and ignore redundant separators.
func TestGetMatchesRemotePathInsensitively(t *testing.T) {
	store := newTestStore(t)
	if _, err := store.Add(Record{SessionID: "session-1", RemotePath: "/tmp/Test.txt"}); err != nil {
		t.Fatalf("Add() = %v, want success", err)
	}

	history, err := store.Get("session-1", "/tmp/./test.txt")
	if err != nil {
		t.Fatalf("Get() = %v, want success", err)
	}
	if len(history) != 1 {
		t.Fatalf("Get() returned %d records, want 1", len(history))
	}
}

func TestUpdate(t *testing.T) {
	store := newTestStore(t)
	id, err := store.Add(Record{SessionID: "session-1", RemotePath: "/tmp/test.txt", Status: "in_progress"})
	if err != nil {
		t.Fatalf("Add() = %v, want success", err)
	}

	if err := store.Update(id, "completed", 2048, ""); err != nil {
		t.Fatalf("Update() = %v, want success", err)
	}

	history, err := store.Get("session-1", "/tmp/test.txt")
	if err != nil {
		t.Fatalf("Get() = %v, want success", err)
	}
	if history[0].Status != "completed" || history[0].Size != 2048 {
		t.Fatalf("record = %+v, want completed at 2048", history[0])
	}
}

// A zero size means the transfer never reported one, so it must not overwrite
// the size already recorded.
func TestUpdateKeepsSizeWhenNoneIsGiven(t *testing.T) {
	store := newTestStore(t)
	id, err := store.Add(Record{SessionID: "session-1", RemotePath: "/tmp/a", Size: 512})
	if err != nil {
		t.Fatalf("Add() = %v, want success", err)
	}

	if err := store.Update(id, "failed", 0, "RPC timeout"); err != nil {
		t.Fatalf("Update() = %v, want success", err)
	}

	history, err := store.Get("session-1", "/tmp/a")
	if err != nil {
		t.Fatalf("Get() = %v, want success", err)
	}
	if history[0].Size != 512 {
		t.Errorf("Size = %d, want the existing 512", history[0].Size)
	}
	if history[0].Error != "RPC timeout" {
		t.Errorf("Error = %q, want the recorded failure", history[0].Error)
	}
}

func TestUpdateUnknownIDWritesNothing(t *testing.T) {
	store := newTestStore(t)
	if _, err := store.Add(Record{SessionID: "session-1", RemotePath: "/tmp/a"}); err != nil {
		t.Fatalf("Add() = %v, want success", err)
	}

	if err := store.Update("not-a-record", "completed", 10, ""); err != nil {
		t.Fatalf("Update() = %v, want success", err)
	}

	all, err := store.All()
	if err != nil {
		t.Fatalf("All() = %v, want success", err)
	}
	if len(all) != 1 || all[0].Status != "" {
		t.Fatalf("records = %+v, want the untouched record", all)
	}
}

func TestHistorySurvivesANewStore(t *testing.T) {
	dir := t.TempDir()
	if _, err := New(dir).Add(Record{SessionID: "session-1", RemotePath: "/tmp/persist.txt", Status: "completed"}); err != nil {
		t.Fatalf("Add() = %v, want success", err)
	}

	all, err := New(dir).All()
	if err != nil {
		t.Fatalf("All() = %v, want success", err)
	}
	if len(all) != 1 {
		t.Fatalf("All() returned %d records, want the persisted one", len(all))
	}
}

func TestClear(t *testing.T) {
	store := newTestStore(t)
	for _, rec := range []Record{
		{SessionID: "session-1", RemotePath: "/tmp/test.txt"},
		{SessionID: "session-1", RemotePath: "/tmp/folder", IsDirectory: true, Status: "failed"},
	} {
		if _, err := store.Add(rec); err != nil {
			t.Fatalf("Add() = %v, want success", err)
		}
	}

	if err := store.Clear("session-1", "/tmp/test.txt"); err != nil {
		t.Fatalf("Clear() = %v, want success", err)
	}
	history, err := store.Get("session-1", "/tmp/test.txt")
	if err != nil {
		t.Fatalf("Get() = %v, want success", err)
	}
	if len(history) != 0 {
		t.Errorf("Get() returned %d records after clearing one path, want 0", len(history))
	}

	if err := store.Clear("", ""); err != nil {
		t.Fatalf("Clear() = %v, want success", err)
	}
	if all, err := store.All(); err != nil || len(all) != 0 {
		t.Errorf("All() = %+v (err %v), want an empty history", all, err)
	}
}

// The hand-rolled store this replaced wrote to a fixed "<path>.tmp", so anything
// left at that name made every later save fail and the record was lost.
// jsonstore creates a unique temporary file, so the same obstacle is harmless.
func TestHistorySurvivesAnObstacleAtTheTempPath(t *testing.T) {
	store := newTestStore(t)
	if _, err := store.Add(Record{SessionID: "session-1", RemotePath: "/tmp/a.txt"}); err != nil {
		t.Fatalf("Add() = %v, want success", err)
	}

	if err := os.Mkdir(store.Path()+".tmp", 0o700); err != nil {
		t.Fatal(err)
	}

	if _, err := store.Add(Record{SessionID: "session-1", RemotePath: "/tmp/b.txt"}); err != nil {
		t.Fatalf("Add() = %v, want success", err)
	}

	all, err := store.All()
	if err != nil {
		t.Fatalf("All() = %v, want success", err)
	}
	if len(all) != 2 {
		t.Errorf("All() returned %d records, want 2: a save was lost", len(all))
	}
}
