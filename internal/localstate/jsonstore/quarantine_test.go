package jsonstore

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func quarantinedNames(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var found []string
	for _, entry := range entries {
		if strings.Contains(entry.Name(), ".corrupt-") {
			found = append(found, entry.Name())
		}
	}
	return found
}

// Callers read a failed Load as empty state, so without quarantining the next
// Save replaces the only copy of whatever the file held.
func TestCorruptFileIsQuarantined(t *testing.T) {
	dir := t.TempDir()
	store := New[testPayload](dir, "corrupt-quarantine")
	path := store.Path()

	if err := os.WriteFile(path, []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, _, err := store.Load(); err == nil {
		t.Fatal("Load() on corrupt JSON = nil error, want an error")
	}

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("original file still at %s, want it moved aside", path)
	}
	names := quarantinedNames(t, dir)
	if len(names) != 1 {
		t.Fatalf("quarantined files = %v, want exactly one", names)
	}
	body, err := os.ReadFile(filepath.Join(dir, names[0]))
	if err != nil || string(body) != "{not json" {
		t.Fatalf("quarantined contents = %q (%v), want the original bytes", body, err)
	}

	// The store is usable again, and saving does not destroy the quarantined copy.
	if err := store.Save(testPayload{Name: "fresh"}); err != nil {
		t.Fatalf("Save() after quarantine = %v, want success", err)
	}
	if again := quarantinedNames(t, dir); len(again) != 1 {
		t.Fatalf("quarantined files = %v after Save, want the copy preserved", again)
	}
}

// A file that could not be read is not corrupt, so it must be left where it is.
func TestUnreadableFileIsNotQuarantined(t *testing.T) {
	dir := t.TempDir()
	store := New[testPayload](dir, "unreadable")

	if err := os.Mkdir(store.Path(), 0o700); err != nil {
		t.Fatal(err)
	}

	if _, _, err := store.Load(); err == nil {
		t.Fatal("Load() over a directory = nil error, want an error")
	}
	if _, err := os.Stat(store.Path()); err != nil {
		t.Fatalf("path was moved aside on a read failure: %v", err)
	}
	if names := quarantinedNames(t, dir); len(names) != 0 {
		t.Fatalf("quarantined files = %v, want none for a read failure", names)
	}
}
