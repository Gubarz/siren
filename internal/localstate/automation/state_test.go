package automation

import (
	"context"
	"testing"

	"siren/internal/automation"
)

func TestJSONStore_Lifecycle(t *testing.T) {
	dir := t.TempDir()
	store := New(dir)
	ctx := context.Background()

	// Initial load should be empty state without error
	st, err := store.Load(ctx)
	if err != nil {
		t.Fatalf("unexpected load error: %v", err)
	}
	if st == nil || len(st.Rules) != 0 {
		t.Fatalf("expected empty state, got %+v", st)
	}

	// Save rules
	toSave := &automation.State{
		Rules: []automation.AutomationRule{
			{ID: "rule-1", Name: "Test Rule", Enabled: true},
		},
	}
	if err := store.Save(ctx, toSave); err != nil {
		t.Fatalf("save error: %v", err)
	}

	// Reload state
	loaded, err := store.Load(ctx)
	if err != nil {
		t.Fatalf("load error after save: %v", err)
	}
	if len(loaded.Rules) != 1 || loaded.Rules[0].ID != "rule-1" {
		t.Fatalf("unexpected loaded rules: %+v", loaded.Rules)
	}

	// SetServer switches scope
	store.SetServer("teamserver.local", 8888)
	st2, err := store.Load(ctx)
	if err != nil {
		t.Fatalf("load error on new server: %v", err)
	}
	if len(st2.Rules) != 0 {
		t.Fatalf("expected 0 rules on new server, got %d", len(st2.Rules))
	}
}
