package agents

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func newRecord(id string) Record {
	return Record{
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

func TestObserveCreatesActiveRecords(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested")
	s := New(dir)

	before := time.Now().UnixMilli()
	if err := s.Observe([]Record{newRecord("sess-a"), newRecord("beacon-b")}); err != nil {
		t.Fatalf("Observe() error = %v", err)
	}
	after := time.Now().UnixMilli()

	list := s.List()
	if len(list) != 2 {
		t.Fatalf("List() len = %d, want 2", len(list))
	}
	for _, r := range list {
		if r.Status != "active" {
			t.Fatalf("%s Status = %q, want active", r.ID, r.Status)
		}
		if r.FirstSeen < before || r.FirstSeen > after {
			t.Fatalf("%s FirstSeen = %d, want within [%d, %d]", r.ID, r.FirstSeen, before, after)
		}
		if r.LastSeen != r.FirstSeen {
			t.Fatalf("%s LastSeen = %d, want FirstSeen %d", r.ID, r.LastSeen, r.FirstSeen)
		}
		if r.LostAt != 0 {
			t.Fatalf("%s LostAt = %d, want 0", r.ID, r.LostAt)
		}
		if r.Hostname != "host-a" || r.Username != "user-a" {
			t.Fatalf("%s identity = %q/%q, want host-a/user-a", r.ID, r.Hostname, r.Username)
		}
	}

	path := filepath.Join(dir, "gui-agents.json")
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat(%s) error = %v", path, err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Fatalf("file perm = %o, want 600", perm)
	}
	dirInfo, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("Stat(%s) error = %v", dir, err)
	}
	if perm := dirInfo.Mode().Perm(); perm != 0o700 {
		t.Fatalf("dir perm = %o, want 700", perm)
	}
}

func TestObserveUpdatesExistingAndKeepsLost(t *testing.T) {
	s := New(t.TempDir())
	if err := s.Observe([]Record{newRecord("sess-a")}); err != nil {
		t.Fatalf("Observe() error = %v", err)
	}
	first := s.List()[0]

	time.Sleep(5 * time.Millisecond)
	if err := s.MarkLost("sess-a"); err != nil {
		t.Fatalf("MarkLost() error = %v", err)
	}

	updated := newRecord("sess-a")
	updated.Name = "sess-a-renamed"
	updated.Hostname = "host-b"
	updated.Username = "user-b"
	updated.OS = "darwin"
	if err := s.Observe([]Record{updated}); err != nil {
		t.Fatalf("Observe() error = %v", err)
	}

	got := s.List()[0]
	if got.Status != "lost" {
		t.Fatalf("Status = %q, want lost", got.Status)
	}
	if got.FirstSeen != first.FirstSeen {
		t.Fatalf("FirstSeen = %d, want %d", got.FirstSeen, first.FirstSeen)
	}
	if got.LastSeen <= first.LastSeen {
		t.Fatalf("LastSeen = %d, want > %d", got.LastSeen, first.LastSeen)
	}
	if got.LostAt == 0 {
		t.Fatal("LostAt = 0, want preserved mark-lost timestamp")
	}
	if got.Name != "sess-a-renamed" || got.Hostname != "host-b" || got.Username != "user-b" || got.OS != "darwin" {
		t.Fatalf("identity not updated: %+v", got)
	}
}

func TestObserveDoesNotResurrectRemoved(t *testing.T) {
	s := New(t.TempDir())
	if err := s.Observe([]Record{newRecord("sess-a")}); err != nil {
		t.Fatalf("Observe() error = %v", err)
	}
	if err := s.Remove("sess-a"); err != nil {
		t.Fatalf("Remove() error = %v", err)
	}
	if err := s.Observe([]Record{newRecord("sess-a")}); err != nil {
		t.Fatalf("Observe() error = %v", err)
	}
	if got := s.List(); len(got) != 0 {
		t.Fatalf("List() len = %d, want 0 (removed stays removed)", len(got))
	}
}

func TestMarkLostPersistsAcrossReload(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)
	if err := s.Observe([]Record{newRecord("beacon-b")}); err != nil {
		t.Fatalf("Observe() error = %v", err)
	}
	time.Sleep(5 * time.Millisecond)
	if err := s.MarkLost("beacon-b"); err != nil {
		t.Fatalf("MarkLost() error = %v", err)
	}

	reloaded := New(dir)
	list := reloaded.List()
	if len(list) != 1 {
		t.Fatalf("List() len = %d, want 1", len(list))
	}
	if list[0].Status != "lost" {
		t.Fatalf("Status = %q, want lost", list[0].Status)
	}
	if list[0].LostAt == 0 {
		t.Fatal("LostAt = 0, want persisted timestamp")
	}
	if list[0].LastSeen == 0 {
		t.Fatal("LastSeen = 0, want persisted timestamp")
	}
}

func TestMarkLostUnknownIsNoop(t *testing.T) {
	s := New(t.TempDir())
	if err := s.MarkLost("missing"); err != nil {
		t.Fatalf("MarkLost() error = %v, want nil", err)
	}
	if got := s.List(); len(got) != 0 {
		t.Fatalf("List() len = %d, want 0", len(got))
	}
}

func TestRemoveHidesAndPersistsAcrossReload(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)
	if err := s.Observe([]Record{newRecord("sess-a")}); err != nil {
		t.Fatalf("Observe() error = %v", err)
	}
	if err := s.Remove("sess-a"); err != nil {
		t.Fatalf("Remove() error = %v", err)
	}
	if got := s.List(); len(got) != 0 {
		t.Fatalf("List() len = %d, want 0", len(got))
	}
	if err := s.MarkLost("sess-a"); err != nil {
		t.Fatalf("MarkLost() on removed error = %v, want nil", err)
	}

	reloaded := New(dir)
	if got := reloaded.List(); len(got) != 0 {
		t.Fatalf("reloaded List() len = %d, want 0", len(got))
	}
}

func TestRemoveUnknownIsNoop(t *testing.T) {
	s := New(t.TempDir())
	if err := s.Remove("missing"); err != nil {
		t.Fatalf("Remove() error = %v, want nil", err)
	}
}

func TestListSortsByLastSeenThenID(t *testing.T) {
	s := New(t.TempDir())
	if err := s.Observe([]Record{newRecord("sess-a")}); err != nil {
		t.Fatalf("Observe() error = %v", err)
	}
	time.Sleep(5 * time.Millisecond)
	if err := s.Observe([]Record{newRecord("beacon-b")}); err != nil {
		t.Fatalf("Observe() error = %v", err)
	}
	list := s.List()
	if len(list) != 2 || list[0].ID != "beacon-b" || list[1].ID != "sess-a" {
		t.Fatalf("List() order = %v, want [beacon-b sess-a] by LastSeen desc", list)
	}

	// Same timestamp → ID ascending.
	same := New(t.TempDir())
	if err := same.Observe([]Record{newRecord("sess-a"), newRecord("beacon-b")}); err != nil {
		t.Fatalf("Observe() error = %v", err)
	}
	list = same.List()
	if len(list) != 2 || list[0].ID != "beacon-b" || list[1].ID != "sess-a" {
		t.Fatalf("List() tie order = %v, want [beacon-b sess-a] by ID asc", list)
	}
}

func TestSetServerScopesFiles(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)

	s.SetServer("host-a", 31337)
	if err := s.Observe([]Record{newRecord("sess-a")}); err != nil {
		t.Fatalf("Observe() error = %v", err)
	}

	s.SetServer("host-b", 8888)
	if got := s.List(); len(got) != 0 {
		t.Fatalf("List() after switch len = %d, want 0", len(got))
	}
	if err := s.Observe([]Record{newRecord("beacon-b")}); err != nil {
		t.Fatalf("Observe() error = %v", err)
	}

	s.SetServer("host-a", 31337)
	list := s.List()
	if len(list) != 1 || list[0].ID != "sess-a" {
		t.Fatalf("List() for host-a = %v, want [sess-a]", list)
	}

	s.SetServer("host-b", 8888)
	list = s.List()
	if len(list) != 1 || list[0].ID != "beacon-b" {
		t.Fatalf("List() for host-b = %v, want [beacon-b]", list)
	}

	for _, name := range []string{"gui-agents-host-a_31337.json", "gui-agents-host-b_8888.json"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Fatalf("Stat(%s) error = %v", name, err)
		}
	}
}

func TestCorruptFileIsSkipped(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "gui-agents.json"), []byte("{ not json"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	s := New(dir)
	if got := s.List(); len(got) != 0 {
		t.Fatalf("List() len = %d, want 0 for corrupt file", len(got))
	}

	if err := s.Observe([]Record{newRecord("sess-a")}); err != nil {
		t.Fatalf("Observe() after corrupt load error = %v", err)
	}
	if got := s.List(); len(got) != 1 {
		t.Fatalf("List() len = %d, want 1 after recovery", len(got))
	}

	scoped := filepath.Join(dir, "gui-agents-host-a_1.json")
	if err := os.WriteFile(scoped, []byte("garbage"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	s.SetServer("host-a", 1)
	if got := s.List(); len(got) != 0 {
		t.Fatalf("List() len = %d, want 0 for corrupt scoped file", len(got))
	}
}

type fakeClock struct {
	now time.Time
}

func newFakeClock() *fakeClock {
	return &fakeClock{now: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
}

func (c *fakeClock) Now() time.Time { return c.now }

func (c *fakeClock) Advance(d time.Duration) { c.now = c.now.Add(d) }

func storedByID(t *testing.T, path, id string) Record {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", path, err)
	}
	var stored []Record
	if err := json.Unmarshal(data, &stored); err != nil {
		t.Fatalf("Unmarshal(%s) error = %v", path, err)
	}
	for _, r := range stored {
		if r.ID == id {
			return r
		}
	}
	t.Fatalf("record %s not found in %s", id, path)
	return Record{}
}

func TestObserveThrottlesUnchangedLastSeen(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)
	clock := newFakeClock()
	s.nowFn = clock.Now
	path := filepath.Join(dir, "gui-agents.json")

	if err := s.Observe([]Record{newRecord("sess-a")}); err != nil {
		t.Fatalf("Observe() error = %v", err)
	}
	first, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	clock.Advance(time.Second)
	if err := s.Observe([]Record{newRecord("sess-a")}); err != nil {
		t.Fatalf("Observe() error = %v", err)
	}

	second, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if !bytes.Equal(first, second) {
		t.Fatal("persisted file changed for LastSeen-only movement within throttle window")
	}
	if got, want := s.List()[0].LastSeen, clock.Now().UnixMilli(); got != want {
		t.Fatalf("List() LastSeen = %d, want %d", got, want)
	}
}

func TestObservePersistsLastSeenAfterThrottleWindow(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)
	clock := newFakeClock()
	s.nowFn = clock.Now

	if err := s.Observe([]Record{newRecord("sess-a")}); err != nil {
		t.Fatalf("Observe() error = %v", err)
	}

	clock.Advance(30 * time.Second)
	if err := s.Observe([]Record{newRecord("sess-a")}); err != nil {
		t.Fatalf("Observe() error = %v", err)
	}

	stored := storedByID(t, filepath.Join(dir, "gui-agents.json"), "sess-a")
	if want := clock.Now().UnixMilli(); stored.LastSeen != want {
		t.Fatalf("stored LastSeen = %d, want %d after throttle window", stored.LastSeen, want)
	}
}

func TestObservePersistsIdentityChangeImmediately(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)
	clock := newFakeClock()
	s.nowFn = clock.Now

	if err := s.Observe([]Record{newRecord("sess-a")}); err != nil {
		t.Fatalf("Observe() error = %v", err)
	}

	clock.Advance(time.Second)
	updated := newRecord("sess-a")
	updated.Hostname = "host-b"
	if err := s.Observe([]Record{updated}); err != nil {
		t.Fatalf("Observe() error = %v", err)
	}

	stored := storedByID(t, filepath.Join(dir, "gui-agents.json"), "sess-a")
	if stored.Hostname != "host-b" {
		t.Fatalf("stored Hostname = %q, want host-b", stored.Hostname)
	}
	if want := clock.Now().UnixMilli(); stored.LastSeen != want {
		t.Fatalf("stored LastSeen = %d, want %d", stored.LastSeen, want)
	}
}
