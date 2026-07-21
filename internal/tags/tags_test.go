package tags

import (
	"path/filepath"
	"testing"
)

func TestSetAgentTagsNormalizesDeduplicatesAndSorts(t *testing.T) {
	s := newTestService(t)

	if err := s.SetAgentTags("agent-1", []string{" Prod ", "dev", "", "prod", "DEV"}); err != nil {
		t.Fatalf("SetAgentTags returned error: %v", err)
	}

	got := s.GetAgentTags("agent-1")
	want := []string{"dev", "prod"}
	assertStringSlice(t, got, want)
}

func TestSetAgentTagsDeletesEmptyLists(t *testing.T) {
	s := newTestService(t)

	if err := s.SetAgentTags("agent-1", []string{"prod"}); err != nil {
		t.Fatalf("SetAgentTags returned error: %v", err)
	}
	if err := s.SetAgentTags("agent-1", []string{" ", ""}); err != nil {
		t.Fatalf("SetAgentTags returned error: %v", err)
	}

	if got := s.GetAgentTags("agent-1"); len(got) != 0 {
		t.Fatalf("GetAgentTags() = %v, want empty", got)
	}
	if _, ok := s.GetAllTags()["agent-1"]; ok {
		t.Fatal("GetAllTags() retained an agent with no tags")
	}
}

func TestTagsReturnedByServiceAreCopies(t *testing.T) {
	s := newTestService(t)

	if err := s.SetAgentTags("agent-1", []string{"prod"}); err != nil {
		t.Fatalf("SetAgentTags returned error: %v", err)
	}

	tags := s.GetAgentTags("agent-1")
	tags[0] = "changed"
	allTags := s.GetAllTags()
	allTags["agent-1"][0] = "changed-again"

	assertStringSlice(t, s.GetAgentTags("agent-1"), []string{"prod"})
}

func TestKnownTagsReturnsUniqueSortedTagsAcrossAgents(t *testing.T) {
	s := newTestService(t)

	if err := s.SetAgentTags("agent-1", []string{"prod", "dev"}); err != nil {
		t.Fatalf("SetAgentTags returned error: %v", err)
	}
	if err := s.SetAgentTags("agent-2", []string{"qa", "prod"}); err != nil {
		t.Fatalf("SetAgentTags returned error: %v", err)
	}

	assertStringSlice(t, s.KnownTags(), []string{"dev", "prod", "qa"})
}

func TestTagsPersistAndLoad(t *testing.T) {
	s := newTestService(t)

	if err := s.SetAgentTags("agent-1", []string{"prod"}); err != nil {
		t.Fatalf("SetAgentTags returned error: %v", err)
	}

	loaded := &Service{path: s.path, tags: map[string][]string{}}
	if err := loaded.load(); err != nil {
		t.Fatalf("load returned error: %v", err)
	}

	assertStringSlice(t, loaded.GetAgentTags("agent-1"), []string{"prod"})
}

func TestSetAgentColorRejectsUnknownColors(t *testing.T) {
	s := newTestService(t)

	if err := s.SetAgentColor("agent-1", "chartreuse"); err == nil {
		t.Fatal("SetAgentColor accepted a color outside the palette")
	}
	if err := s.SetAgentColor("agent-1", " Blue "); err != nil {
		t.Fatalf("SetAgentColor returned error: %v", err)
	}
	if got := s.GetAllColors()["agent-1"]; got != "blue" {
		t.Fatalf("GetAllColors()[agent-1] = %q, want %q", got, "blue")
	}
}

func TestSetAgentColorEmptyClears(t *testing.T) {
	s := newTestService(t)

	if err := s.SetAgentColor("agent-1", "red"); err != nil {
		t.Fatalf("SetAgentColor returned error: %v", err)
	}
	if err := s.SetAgentColor("agent-1", ""); err != nil {
		t.Fatalf("SetAgentColor(clear) returned error: %v", err)
	}
	if _, ok := s.GetAllColors()["agent-1"]; ok {
		t.Fatal("GetAllColors() retained a cleared color")
	}
}

func TestColorsPersistAndLoad(t *testing.T) {
	s := newTestService(t)

	if err := s.SetAgentColor("agent-1", "purple"); err != nil {
		t.Fatalf("SetAgentColor returned error: %v", err)
	}

	loaded := &Service{path: s.path, tags: map[string][]string{}}
	if err := loaded.load(); err != nil {
		t.Fatalf("load returned error: %v", err)
	}
	if got := loaded.GetAllColors()["agent-1"]; got != "purple" {
		t.Fatalf("loaded color = %q, want %q", got, "purple")
	}
}

func newTestService(t *testing.T) *Service {
	t.Helper()
	return &Service{
		path: filepath.Join(t.TempDir(), "tags.json"),
		tags: map[string][]string{},
	}
}

func assertStringSlice(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("len(%v) = %d, want %d", got, len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("slice[%d] = %q, want %q (full slice %v)", i, got[i], want[i], got)
		}
	}
}
