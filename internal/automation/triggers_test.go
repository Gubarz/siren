package automation

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

type fakeTrigger struct {
	typ      string
	schema   []FieldSpec
	armCalls atomic.Int64
	lastCfg  map[string]any
	fireFn   func(FireEvent)
}

func (f *fakeTrigger) Type() string                  { return f.typ }
func (f *fakeTrigger) ConfigSchema() []FieldSpec     { return f.schema }
func (f *fakeTrigger) Arm(ctx context.Context, cfg map[string]any, fire func(FireEvent)) error {
	f.fireFn = fire
	f.lastCfg = cfg
	f.armCalls.Add(1)
	<-ctx.Done()
	return ctx.Err()
}

type memStore struct{ state *State }

func (m *memStore) Load(context.Context) (*State, error) {
	if m.state == nil {
		return &State{}, nil
	}
	return m.state, nil
}
func (m *memStore) Save(_ context.Context, s *State) error { m.state = s; return nil }
func (m *memStore) SetServer(string, uint32)               {}

type fakeExec struct{}

func (fakeExec) Execute(context.Context, string, string, string) (string, error) { return "", nil }

type fakeTargets struct {
	conn bool
	list []Target
}

func (t *fakeTargets) Connected() bool                                   { return t.conn }
func (t *fakeTargets) GetSessions(context.Context) ([]Target, error)     { return t.list, nil }
func (t *fakeTargets) GetBeacons(context.Context) ([]Target, error)      { return nil, nil }
func (t *fakeTargets) FindTarget(context.Context, string) (Target, error) { return Target{}, context.DeadlineExceeded }

type fakeAction struct{ typ string }

func (f *fakeAction) Type() string                    { return f.typ }
func (f *fakeAction) ConfigSchema() []FieldSpec       { return nil }
func (f *fakeAction) Execute(*RunContext) error       { return nil }

func newTestEngine(t *testing.T) *Engine {
	t.Helper()
	e := New(Dependencies{
		Store:    &memStore{},
		Executor: fakeExec{},
		Targets:  &fakeTargets{},
	})
	e.RegisterAction(&fakeAction{typ: "commands"})
	e.RegisterAction(&fakeAction{typ: "script"})
	e.Start(context.Background())
	return e
}

func TestRegisterTriggerRejectsDuplicates(t *testing.T) {
	e := newTestEngine(t)
	if err := e.RegisterTrigger(&fakeTrigger{typ: "x"}); err != nil {
		t.Fatal(err)
	}
	if err := e.RegisterTrigger(&fakeTrigger{typ: "x"}); err == nil {
		t.Fatal("duplicate registration must fail")
	}
}

func TestSaveRuleValidatesAgainstRegistry(t *testing.T) {
	e := newTestEngine(t)
	_ = e.RegisterTrigger(&fakeTrigger{typ: "manual"})
	if _, err := e.SaveRule(AutomationRule{Name: "bad", Trigger: "nope", Commands: []string{"ps"}}); err == nil {
		t.Fatal("unknown trigger must be rejected")
	}
	rule, err := e.SaveRule(AutomationRule{Name: "ok", Trigger: "manual", Commands: []string{"ps"}})
	if err != nil || rule.ID == "" {
		t.Fatalf("save: %v", err)
	}
}

func TestEnableArmsAndDisableDisarms(t *testing.T) {
	e := newTestEngine(t)
	tr := &fakeTrigger{typ: "fake"}
	_ = e.RegisterTrigger(tr)
	rule, _ := e.SaveRule(AutomationRule{Name: "r", Trigger: "fake", Commands: []string{"ps"}, Enabled: false})
	if tr.armCalls.Load() != 0 {
		t.Fatal("disabled rule must not arm")
	}
	if err := e.SetRuleEnabled(rule.ID, true); err != nil {
		t.Fatal(err)
	}
	waitArmed(t, tr, 1)
	if err := e.SetRuleEnabled(rule.ID, false); err != nil {
		t.Fatal(err)
	}
	assertNoArmedGoroutines(t, e)
}

func TestDeleteDisarms(t *testing.T) {
	e := newTestEngine(t)
	tr := &fakeTrigger{typ: "fake"}
	_ = e.RegisterTrigger(tr)
	rule, _ := e.SaveRule(AutomationRule{Name: "r", Trigger: "fake", Commands: []string{"ps"}, Enabled: true})
	waitArmed(t, tr, 1)
	if err := e.DeleteRule(rule.ID); err != nil {
		t.Fatal(err)
	}
	assertNoArmedGoroutines(t, e)
}

func TestSetServerDisarmsAll(t *testing.T) {
	e := newTestEngine(t)
	tr := &fakeTrigger{typ: "fake"}
	_ = e.RegisterTrigger(tr)
	_, _ = e.SaveRule(AutomationRule{Name: "r", Trigger: "fake", Commands: []string{"ps"}, Enabled: true})
	waitArmed(t, tr, 1)
	e.SetServer("other", 31337)
	assertNoArmedGoroutines(t, e)
}

func TestStartArmsEnabledRules(t *testing.T) {
	e := newTestEngine(t)
	tr := &fakeTrigger{typ: "fake"}
	_ = e.RegisterTrigger(tr)
	_, _ = e.SaveRule(AutomationRule{Name: "r", Trigger: "fake", Commands: []string{"ps"}, Enabled: true})
	waitArmed(t, tr, 1)
}

func TestFireCallbackQueuesRun(t *testing.T) {
	e := newTestEngine(t)
	e.targets.(*fakeTargets).conn = true
	e.targets.(*fakeTargets).list = []Target{{ID: "t1", Kind: "session"}}

	tr := &fakeTrigger{typ: "fake"}
	_ = e.RegisterTrigger(tr)
	_, _ = e.SaveRule(AutomationRule{Name: "r", Trigger: "fake", Commands: []string{"ps"}, Enabled: true})
	waitArmed(t, tr, 1)
	if tr.fireFn == nil {
		t.Fatal("trigger not armed")
	}
	tr.fireFn(FireEvent{Target: &Target{ID: "t1", Kind: "session"}})
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		history, _ := e.GetHistory()
		if len(history) == 1 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("fire callback did not queue a run")
}

func waitArmed(t *testing.T, tr *fakeTrigger, want int64) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if tr.armCalls.Load() == want {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("arm calls: %d, want %d", tr.armCalls.Load(), want)
}

func assertNoArmedGoroutines(t *testing.T, e *Engine) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if e.armedCount() == 0 {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("still armed: %d", e.armedCount())
}
