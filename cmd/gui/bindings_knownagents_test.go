package gui

import (
	"testing"
	"time"

	knownagents "siren/internal/localstate/agents"
)

func newKnownAgentRecord(id string) knownagents.Record {
	return knownagents.Record{
		ID:            id,
		Kind:          "session",
		Name:          id,
		Hostname:      "host-a",
		Username:      "user-a",
		OS:            "linux",
		Arch:          "amd64",
		RemoteAddress: "10.0.0.1:443",
		Transport:     "mtls",
	}
}

func TestListKnownAgentsMapsRecords(t *testing.T) {
	svc := knownagents.New(t.TempDir())
	if err := svc.Observe([]knownagents.Record{newKnownAgentRecord("sess-a")}); err != nil {
		t.Fatalf("Observe() error = %v", err)
	}
	time.Sleep(5 * time.Millisecond)
	if err := svc.Observe([]knownagents.Record{newKnownAgentRecord("beacon-b")}); err != nil {
		t.Fatalf("Observe() error = %v", err)
	}
	if err := svc.MarkLost("sess-a"); err != nil {
		t.Fatalf("MarkLost() error = %v", err)
	}

	views, err := listKnownAgents(svc)
	if err != nil {
		t.Fatalf("listKnownAgents() error = %v", err)
	}
	if len(views) != 2 {
		t.Fatalf("len = %d, want 2", len(views))
	}
	if views[0].ID != "beacon-b" || views[0].Status != "active" {
		t.Fatalf("views[0] = %+v, want beacon-b active (newest first)", views[0])
	}
	if views[1].ID != "sess-a" || views[1].Status != "lost" {
		t.Fatalf("views[1] = %+v, want sess-a lost", views[1])
	}

	got := views[1]
	if got.Kind != "session" || got.Name != "sess-a" || got.Hostname != "host-a" ||
		got.Username != "user-a" || got.OS != "linux" || got.Arch != "amd64" ||
		got.RemoteAddress != "10.0.0.1:443" || got.Transport != "mtls" {
		t.Fatalf("identity not mapped: %+v", got)
	}
	if got.FirstSeen == 0 || got.LastSeen == 0 || got.LostAt == 0 {
		t.Fatalf("timestamps not mapped: %+v", got)
	}
	if got.LastSeen < got.FirstSeen || got.LostAt < got.LastSeen {
		t.Fatalf("timestamps out of order: %+v", got)
	}
	if views[0].LostAt != 0 {
		t.Fatalf("active LostAt = %d, want 0", views[0].LostAt)
	}
}

func TestListKnownAgentsEmpty(t *testing.T) {
	views, err := listKnownAgents(knownagents.New(t.TempDir()))
	if err != nil {
		t.Fatalf("listKnownAgents() error = %v", err)
	}
	if views == nil {
		t.Fatal("views = nil, want non-nil empty slice")
	}
	if len(views) != 0 {
		t.Fatalf("len = %d, want 0", len(views))
	}
}

func TestListKnownAgentsNilService(t *testing.T) {
	views, err := listKnownAgents(nil)
	if err != nil {
		t.Fatalf("listKnownAgents(nil) error = %v", err)
	}
	if views == nil || len(views) != 0 {
		t.Fatalf("views = %#v, want non-nil empty slice", views)
	}
}

func TestRemoveKnownAgentDropsFromList(t *testing.T) {
	dir := t.TempDir()
	svc := knownagents.New(dir)
	if err := svc.Observe([]knownagents.Record{newKnownAgentRecord("sess-a")}); err != nil {
		t.Fatalf("Observe() error = %v", err)
	}

	if err := removeKnownAgent(svc, "sess-a"); err != nil {
		t.Fatalf("removeKnownAgent() error = %v", err)
	}
	views, err := listKnownAgents(svc)
	if err != nil {
		t.Fatalf("listKnownAgents() error = %v", err)
	}
	if len(views) != 0 {
		t.Fatalf("len = %d after remove, want 0", len(views))
	}

	reloaded := knownagents.New(dir)
	if got := reloaded.List(); len(got) != 0 {
		t.Fatalf("reloaded len = %d, want 0 (removal persisted)", len(got))
	}
}

func TestRemoveKnownAgentUnknownIsNoop(t *testing.T) {
	if err := removeKnownAgent(knownagents.New(t.TempDir()), "missing"); err != nil {
		t.Fatalf("removeKnownAgent(missing) error = %v, want nil", err)
	}
}

func TestRemoveKnownAgentNilService(t *testing.T) {
	if err := removeKnownAgent(nil, "missing"); err != nil {
		t.Fatalf("removeKnownAgent(nil) error = %v, want nil", err)
	}
}
