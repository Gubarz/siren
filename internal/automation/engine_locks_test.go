package automation

import (
	"context"
	"sync"
	"testing"
)

// lockedStore is a StateStore that can be saved from several goroutines. The
// shared memStore cannot, and using it here would report a race in the test
// rather than in the engine.
type lockedStore struct {
	mu    sync.Mutex
	state *State
}

func (s *lockedStore) Load(context.Context) (*State, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.state == nil {
		return &State{}, nil
	}
	return s.state, nil
}

func (s *lockedStore) Save(_ context.Context, state *State) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state = state
	return nil
}

func (s *lockedStore) SetServer(string, uint32) {}

// quietTrigger blocks until it is disarmed and records nothing, so arming it
// concurrently cannot itself introduce a race. fakeTrigger writes to its own
// fields in Arm, which would.
type quietTrigger struct{ typ string }

func (q *quietTrigger) Type() string              { return q.typ }
func (q *quietTrigger) ConfigSchema() []FieldSpec { return nil }

func (q *quietTrigger) Arm(ctx context.Context, _ map[string]any, _ func(FireEvent)) error {
	<-ctx.Done()
	return ctx.Err()
}

// The engine takes three locks and never holds two at once. This drives the
// paths behind each of them from several goroutines at the same time, so a
// change that starts nesting them, or that reads guarded state without taking
// the lock, has to argue with the race detector.
//
//	registryMu  SaveRule validates against it, TriggerSchemas and ActionSchemas read it
//	armedMu     SetRuleEnabled arms and disarms
//	mu          SaveRule, ListRules and GetHistory
func TestEngineHandlesConcurrentUse(t *testing.T) {
	e := New(Dependencies{
		Store:    &lockedStore{},
		Executor: fakeExec{},
		Targets:  &fakeTargets{},
	})
	if err := e.RegisterTrigger(&quietTrigger{typ: "manual"}); err != nil {
		t.Fatal(err)
	}
	if err := e.RegisterAction(&fakeAction{typ: "commands"}); err != nil {
		t.Fatal(err)
	}
	e.SetCollector(nil)
	e.Start(context.Background())

	rule, err := e.SaveRule(AutomationRule{Name: "r", Trigger: "manual", Commands: []string{"ps"}})
	if err != nil {
		t.Fatal(err)
	}

	const workers, rounds = 8, 25
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for round := 0; round < rounds; round++ {
				hammerEngine(t, e, rule.ID, round)
			}
		}()
	}
	wg.Wait()
}

// hammerEngine exercises the paths behind all three engine locks once.
func hammerEngine(t *testing.T, e *Engine, ruleID string, round int) {
	t.Helper()
	updated := AutomationRule{ID: ruleID, Name: "r", Trigger: "manual", Commands: []string{"ps"}}
	if _, err := e.SaveRule(updated); err != nil {
		t.Errorf("SaveRule: %v", err)
		return
	}
	if err := e.SetRuleEnabled(ruleID, round%2 == 0); err != nil {
		t.Errorf("SetRuleEnabled: %v", err)
		return
	}
	if _, err := e.ListRules(); err != nil {
		t.Errorf("ListRules: %v", err)
		return
	}
	if _, err := e.GetHistory(); err != nil {
		t.Errorf("GetHistory: %v", err)
		return
	}
	_ = e.TriggerSchemas()
	_ = e.ActionSchemas()
	e.SetCollector(nil)
	_ = e.collector()
}
