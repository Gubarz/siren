package automation

import (
	"context"
	"sync"
	"testing"
	"time"
)

// overlapStore widens the window around a persist so a test can tell whether
// SetServer swapped the store path while a save was in flight.
type overlapStore struct {
	memStore

	mu         sync.Mutex
	armed      bool
	saving     bool
	overlapped bool
}

func (s *overlapStore) arm() {
	s.mu.Lock()
	s.armed = true
	s.mu.Unlock()
}

func (s *overlapStore) isSaving() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.saving
}

func (s *overlapStore) didOverlap() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.overlapped
}

func (s *overlapStore) Save(ctx context.Context, state *State) error {
	s.mu.Lock()
	if !s.armed {
		s.mu.Unlock()
		return s.memStore.Save(ctx, state)
	}
	s.saving = true
	s.mu.Unlock()

	time.Sleep(80 * time.Millisecond)

	s.mu.Lock()
	s.saving = false
	s.mu.Unlock()
	return nil
}

func (s *overlapStore) SetServer(string, uint32) {
	s.mu.Lock()
	if s.saving {
		s.overlapped = true
	}
	s.mu.Unlock()
	time.Sleep(80 * time.Millisecond)
}

// Switching teamservers must not happen while a run is persisting, or that run
// lands in the new server's file.
func TestSetServerDoesNotSwapThePathDuringAPersist(t *testing.T) {
	st := &overlapStore{}
	e := New(Dependencies{Store: st, Executor: fakeExec{}, Targets: &fakeTargets{}})
	e.RegisterAction(&fakeAction{typ: "commands"})
	_ = e.RegisterTrigger(&fakeTrigger{typ: "manual"})
	e.Start(context.Background())

	rule, err := e.SaveRule(AutomationRule{Name: "r", Trigger: "manual", Commands: []string{"ps"}})
	if err != nil {
		t.Fatal(err)
	}
	st.arm()

	e.dispatchRule(rule, "manual", &Target{ID: "t1", Kind: "session"})

	deadline := time.Now().Add(2 * time.Second)
	for !st.isSaving() {
		if time.Now().After(deadline) {
			t.Fatal("the run never started persisting")
		}
		time.Sleep(5 * time.Millisecond)
	}

	e.SetServer("other", 31337)

	if st.didOverlap() {
		t.Fatal("SetServer swapped the store path while a persist was in flight")
	}
}
