package bloodhound

import (
	"errors"
	"path/filepath"
	"testing"
)

func TestValidate(t *testing.T) {
	valid := Config{ServerURL: "https://bh.example.com", TokenID: "id", TokenKey: "key"}
	if err := validate(valid); err != nil {
		t.Fatalf("validate(valid) = %v, want nil", err)
	}
	for _, tc := range []struct {
		name string
		cfg  Config
	}{
		{"empty url", Config{TokenID: "id", TokenKey: "key"}},
		{"bad scheme", Config{ServerURL: "ftp://x", TokenID: "id", TokenKey: "key"}},
		{"no host", Config{ServerURL: "https://", TokenID: "id", TokenKey: "key"}},
		{"missing token id", Config{ServerURL: "https://bh.example.com", TokenKey: "key"}},
		{"missing token key", Config{ServerURL: "https://bh.example.com", TokenID: "id"}},
	} {
		if err := validate(tc.cfg); err == nil {
			t.Errorf("%s: validate(%+v) = nil, want error", tc.name, tc.cfg)
		}
	}
}

func TestMasked(t *testing.T) {
	view := Masked(Config{
		ServerURL: "https://bh.example.com", TokenID: "id",
		TokenKey: "supersecret", InsecureTLS: true,
	})
	if view.TokenID != "id" || view.ServerURL != "https://bh.example.com" {
		t.Fatalf("unexpected view %+v", view)
	}
	if !view.HasTokenKey {
		t.Fatal("HasTokenKey = false, want true")
	}
	if !view.InsecureTLS {
		t.Fatal("InsecureTLS = false, want true")
	}
	for _, s := range []string{view.ServerURL, view.TokenID} {
		if s == "supersecret" {
			t.Fatal("token key leaked into view")
		}
	}
	if Masked(Config{}).HasTokenKey {
		t.Fatal("empty config HasTokenKey = true, want false")
	}
}

func TestConfigStoreRoundTrip(t *testing.T) {
	dir := t.TempDir()
	store := NewConfigStore(dir)
	if _, ok, err := store.Load(); ok || err != nil {
		t.Fatalf("Load on empty dir = (%v,%v), want (false,nil)", ok, err)
	}
	cfg := Config{ServerURL: "http://127.0.0.1:8080", TokenID: "id", TokenKey: "key"}
	if err := store.Save(cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, ok, err := NewConfigStore(filepath.Join(dir)).Load()
	if err != nil || !ok {
		t.Fatalf("Load: ok=%v err=%v, want true,nil", ok, err)
	}
	if got != cfg {
		t.Fatalf("round trip got %+v want %+v", got, cfg)
	}
	if err := store.Save(Config{ServerURL: "nope"}); !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("Save(invalid) = %v, want ErrNotConfigured", err)
	}
}

func TestConfigured(t *testing.T) {
	if (Config{}).configured() {
		t.Fatal("zero config configured()=true")
	}
	if !(Config{ServerURL: "https://x", TokenID: "a", TokenKey: "b"}).configured() {
		t.Fatal("full config configured()=false")
	}
}
