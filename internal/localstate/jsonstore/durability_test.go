package jsonstore

import (
	"os"
	"path/filepath"
	"testing"
)

// The save path used to be a fixed "<path>.tmp", shared by every writer. A
// directory left there (or a second process mid-write) made every later save
// fail outright.
func TestSaveSurvivesAnObstacleAtTheOldTmpPath(t *testing.T) {
	dir := t.TempDir()
	store := New[testPayload](dir, "test-store")

	if err := os.Mkdir(store.Path()+".tmp", 0o700); err != nil {
		t.Fatal(err)
	}

	want := testPayload{Name: "siren", Count: 7}
	if err := store.Save(want); err != nil {
		t.Fatalf("Save() = %v, want success", err)
	}

	got, ok, err := store.Load()
	if err != nil || !ok {
		t.Fatalf("Load() ok=%v err=%v, want the saved value", ok, err)
	}
	if got.Name != want.Name || got.Count != want.Count {
		t.Fatalf("Load() = %+v, want %+v", got, want)
	}
}

// A save leaves nothing behind next to the real file.
func TestSaveLeavesNoTempFile(t *testing.T) {
	dir := t.TempDir()
	store := New[testPayload](dir, "test-store")

	if err := store.Save(testPayload{Name: "siren"}); err != nil {
		t.Fatal(err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.Name() != filepath.Base(store.Path()) {
			t.Fatalf("unexpected leftover %q after Save", entry.Name())
		}
	}
}
