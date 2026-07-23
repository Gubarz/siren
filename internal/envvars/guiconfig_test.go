package envvars

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadGUIConfigEmptyDir(t *testing.T) {
	dir := t.TempDir()
	cfg, err := LoadGUIConfig(dir)
	if err != nil {
		t.Fatalf("LoadGUIConfig returned error: %v", err)
	}
	if cfg == nil {
		t.Fatal("expected non-nil config")
	}
	if cfg.DataDirOverride != "" || cfg.LogDirOverride != "" {
		t.Fatalf("expected empty overrides from non-existent file, got data=%q log=%q", cfg.DataDirOverride, cfg.LogDirOverride)
	}
}

func TestSaveAndLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	orig := &GUIConfig{
		DataDirOverride: "/custom/data",
		LogDirOverride:  "/custom/logs",
	}
	if err := SaveGUIConfig(dir, orig); err != nil {
		t.Fatalf("SaveGUIConfig failed: %v", err)
	}
	loaded, err := LoadGUIConfig(dir)
	if err != nil {
		t.Fatalf("LoadGUIConfig after save failed: %v", err)
	}
	if loaded.DataDirOverride != orig.DataDirOverride {
		t.Fatalf("data dir override: got %q, want %q", loaded.DataDirOverride, orig.DataDirOverride)
	}
	if loaded.LogDirOverride != orig.LogDirOverride {
		t.Fatalf("log dir override: got %q, want %q", loaded.LogDirOverride, orig.LogDirOverride)
	}
}

func TestSaveNilConfig(t *testing.T) {
	dir := t.TempDir()
	if err := SaveGUIConfig(dir, nil); err != nil {
		t.Fatalf("SaveGUIConfig with nil config failed: %v", err)
	}
	cfg, err := LoadGUIConfig(dir)
	if err != nil {
		t.Fatalf("LoadGUIConfig after nil save failed: %v", err)
	}
	if cfg.DataDirOverride != "" || cfg.LogDirOverride != "" {
		t.Fatalf("expected empty overrides after nil save")
	}
}

func TestLoadGUIConfigCorruptYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, guiSettingsFilename)
	os.WriteFile(path, []byte("{not: yaml: [[["), 0o600)
	_, err := LoadGUIConfig(dir)
	if err == nil {
		t.Fatal("expected error for corrupt YAML")
	}
}

func TestSaveGUIConfigUnwritable(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, guiSettingsFilename)
	os.WriteFile(path, []byte{}, 0o400)
	os.Chmod(dir, 0o500)
	defer os.Chmod(dir, 0o700)
	err := SaveGUIConfig(dir, &GUIConfig{DataDirOverride: "ignored"})
	if err == nil {
		t.Fatal("expected error when saving to unwritable path")
	}
}
