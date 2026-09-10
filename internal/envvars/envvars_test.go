package envvars

import (
	"testing"
)

func TestResolveDataDirDefaultsToSliverRoot(t *testing.T) {
	t.Setenv("SLIVER_CLIENT_ROOT_DIR", "")
	dir, err := ResolveDataDir(nil)
	if err != nil {
		t.Fatalf("ResolveDataDir returned error: %v", err)
	}
	if dir == "" {
		t.Fatal("expected non-empty data dir")
	}
}

func TestResolveDataDirUsesSliverClientRootDir(t *testing.T) {
	assertResolveEnv(t, "SLIVER_CLIENT_ROOT_DIR", ResolveDataDir)
}

func TestResolveDataDirUsesGuiDataDirEnvVar(t *testing.T) {
	assertResolveEnv(t, "SLIVER_GUI_DATA_DIR", ResolveDataDir)
}

func TestResolveDataDirGuiOverrideWinsOverEnvVar(t *testing.T) {
	assertGuiOverrideWins(t, "SLIVER_GUI_DATA_DIR", ResolveDataDir,
		func(cfg *GUIConfig, dir string) { cfg.DataDirOverride = dir })
}

func TestResolveDataDirInvalidPathReturnsError(t *testing.T) {
	t.Setenv("SLIVER_GUI_DATA_DIR", "/nonexistent/should/fail/here/please")
	_, err := ResolveDataDir(nil)
	if err == nil {
		t.Fatal("expected error for non-creatable directory")
	}
}

func TestResolveLogDirDefaultsToSliverLogs(t *testing.T) {
	t.Setenv("SLIVER_CLIENT_ROOT_DIR", t.TempDir())
	dir, err := ResolveLogDir(nil)
	if err != nil {
		t.Fatalf("ResolveLogDir returned error: %v", err)
	}
	if dir == "" {
		t.Fatal("expected non-empty log dir")
	}
}

func TestResolveLogDirUsesEnvVar(t *testing.T) {
	assertResolveEnv(t, "SLIVER_GUI_LOG_DIR", ResolveLogDir)
}

func TestResolveLogDirGuiOverrideWins(t *testing.T) {
	assertGuiOverrideWins(t, "SLIVER_GUI_LOG_DIR", ResolveLogDir,
		func(cfg *GUIConfig, dir string) { cfg.LogDirOverride = dir })
}

func TestGetEnvInfoCollectsActiveVars(t *testing.T) {
	t.Setenv("SLIVER_CLIENT_ROOT_DIR", "/tmp/test-sliver")
	t.Setenv("SLIVER_NO_UPDATE_CHECK", "1")
	info := GetEnvInfo(nil)
	foundClientRoot := false
	foundNoUpdate := false
	for _, ev := range info.ActiveVars {
		if ev.Name == "SLIVER_CLIENT_ROOT_DIR" {
			if ev.Value != "/tmp/test-sliver" || !ev.Set {
				t.Fatalf("SLIVER_CLIENT_ROOT_DIR: value=%q set=%v", ev.Value, ev.Set)
			}
			foundClientRoot = true
		}
		if ev.Name == "SLIVER_NO_UPDATE_CHECK" {
			if ev.Value != "1" || !ev.Set {
				t.Fatalf("SLIVER_NO_UPDATE_CHECK: value=%q set=%v", ev.Value, ev.Set)
			}
			foundNoUpdate = true
		}
	}
	if !foundClientRoot || !foundNoUpdate {
		t.Fatal("missing expected env vars in ActiveVars")
	}
}

func TestBuildPassthroughEnvForwardsOnlyListedVars(t *testing.T) {
	t.Setenv("SLIVER_CLIENT_ROOT_DIR", "/custom")
	t.Setenv("SECRET_TOKEN", "leaked")
	env := BuildPassthroughEnv("TERM=xterm-256color")

	gotClientRoot := false
	gotSecret := false
	for _, e := range env {
		if e == "SLIVER_CLIENT_ROOT_DIR=/custom" {
			gotClientRoot = true
		}
		if e == "SECRET_TOKEN=leaked" {
			gotSecret = true
		}
	}
	if !gotClientRoot {
		t.Fatal("SLIVER_CLIENT_ROOT_DIR not forwarded")
	}
	if gotSecret {
		t.Fatal("SECRET_TOKEN should not be forwarded")
	}
}

func TestBuildPassthroughEnvIncludesExtra(t *testing.T) {
	t.Setenv("HOME", "/home/test")
	env := BuildPassthroughEnv("TERM=xterm-256color", "COLORTERM=truecolor")
	gotTerm := false
	gotColorTerm := false
	for _, e := range env {
		if e == "TERM=xterm-256color" {
			gotTerm = true
		}
		if e == "COLORTERM=truecolor" {
			gotColorTerm = true
		}
	}
	if !gotTerm || !gotColorTerm {
		t.Fatal("extra vars not included in passthrough env")
	}
}

// assertResolveEnv points envVar at a temp dir and asserts resolve returns it.
func assertResolveEnv(t *testing.T, envVar string, resolve func(*GUIConfig) (string, error)) {
	t.Helper()
	expected := t.TempDir()
	t.Setenv(envVar, expected)
	dir, err := resolve(nil)
	if err != nil {
		t.Fatalf("resolve with %s: %v", envVar, err)
	}
	if dir != expected {
		t.Fatalf("resolve with %s: got %q, want %q", envVar, dir, expected)
	}
}

// assertGuiOverrideWins asserts the GUI override beats the environment value.
func assertGuiOverrideWins(t *testing.T, envVar string, resolve func(*GUIConfig) (string, error), set func(*GUIConfig, string)) {
	t.Helper()
	envDir := t.TempDir()
	guiDir := t.TempDir()
	t.Setenv(envVar, envDir)
	cfg := &GUIConfig{}
	set(cfg, guiDir)
	dir, err := resolve(cfg)
	if err != nil {
		t.Fatalf("resolve with %s override: %v", envVar, err)
	}
	if dir != guiDir {
		t.Fatalf("resolve with %s override: got %q, want %q", envVar, dir, guiDir)
	}
}
