package jsonstore

import (
	"os"
	"path/filepath"
	"testing"
)

type testPayload struct {
	Name  string   `json:"name"`
	Count int      `json:"count"`
	Items []string `json:"items"`
}

func TestScopedStore_LoadSave(t *testing.T) {
	dir := t.TempDir()
	store := New[testPayload](dir, "test-store")

	// Missing file returns false, nil
	val, ok, err := store.Load()
	if err != nil {
		t.Fatalf("unexpected error on missing file: %v", err)
	}
	if ok {
		t.Fatalf("expected ok=false for missing file, got true")
	}
	if val.Name != "" {
		t.Fatalf("expected zero value, got %+v", val)
	}

	// Save payload
	data := testPayload{
		Name:  "siren",
		Count: 42,
		Items: []string{"a", "b", "c"},
	}
	if err := store.Save(data); err != nil {
		t.Fatalf("failed to save: %v", err)
	}

	// Load saved payload
	val, ok, err = store.Load()
	if err != nil {
		t.Fatalf("unexpected error on load: %v", err)
	}
	if !ok {
		t.Fatalf("expected ok=true for existing file")
	}
	if val.Name != "siren" || val.Count != 42 || len(val.Items) != 3 {
		t.Fatalf("loaded payload mismatch: %+v", val)
	}
}

func TestScopedStore_SetServer(t *testing.T) {
	dir := t.TempDir()
	store := New[testPayload](dir, "test-store")

	expectedDefault := filepath.Join(dir, "test-store.json")
	if store.Path() != expectedDefault {
		t.Fatalf("expected path %q, got %q", expectedDefault, store.Path())
	}

	store.SetServer("192.168.1.100", 31337)
	expectedScoped := filepath.Join(dir, "test-store-192.168.1.100_31337.json")
	if store.Path() != expectedScoped {
		t.Fatalf("expected scoped path %q, got %q", expectedScoped, store.Path())
	}

	// Save to scoped server file
	if err := store.Save(testPayload{Name: "server1"}); err != nil {
		t.Fatalf("failed to save scoped: %v", err)
	}

	// Switch server
	store.SetServer("10.0.0.1", 8888)
	_, ok, err := store.Load()
	if err != nil || ok {
		t.Fatalf("expected missing file for second server, ok=%v err=%v", ok, err)
	}

	// Switch back
	store.SetServer("192.168.1.100", 31337)
	val, ok, err := store.Load()
	if err != nil || !ok || val.Name != "server1" {
		t.Fatalf("failed to reload server1 payload: ok=%v, err=%v, val=%+v", ok, err, val)
	}
}

func TestScopedStore_CorruptJSON(t *testing.T) {
	dir := t.TempDir()
	store := New[testPayload](dir, "corrupt-test")

	if err := os.WriteFile(store.Path(), []byte("{invalid json"), 0o600); err != nil {
		t.Fatalf("failed to write corrupt file: %v", err)
	}

	_, _, err := store.Load()
	if err == nil {
		t.Fatalf("expected error on corrupt JSON, got nil")
	}
}
