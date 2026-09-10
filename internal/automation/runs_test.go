package automation

import (
	"context"
	"testing"
	"time"
)

type panicEmitter struct{}

func (panicEmitter) Emit(string, any) { panic("emitter exploded") }

type panicStore struct{ memStore }

func (p *panicStore) Save(context.Context, *State) error { panic("store exploded") }

// waitForNoRunning polls until the engine has released its per-rule bookkeeping,
// which is what lets a rule fire again after a run dies.
func waitForNoRunning(t *testing.T, e *Engine) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		e.mu.RLock()
		running := len(e.running)
		e.mu.RUnlock()
		if running == 0 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("run bookkeeping was never released after a panic")
}

// Emitting a run happens on the goroutine this package started, outside
// executeAction's recover, so a panic there would otherwise end the process.
func TestRunPanicIsRecoveredAndReleasesTheRule(t *testing.T) {
	e := newTestEngine(t)
	_ = e.RegisterTrigger(&fakeTrigger{typ: "manual"})
	rule, err := e.SaveRule(AutomationRule{Name: "r", Trigger: "manual", Commands: []string{"ps"}})
	if err != nil {
		t.Fatal(err)
	}
	e.emitter = panicEmitter{}

	e.dispatchRule(rule, "manual", &Target{ID: "t1", Kind: "session"})

	waitForNoRunning(t, e)
}

// persistLocked runs while e.mu is held, so a panic from the store must not
// leave the mutex locked; every later engine call would block forever.
func TestPersistPanicDoesNotHoldTheEngineMutex(t *testing.T) {
	e := newTestEngine(t)
	_ = e.RegisterTrigger(&fakeTrigger{typ: "manual"})
	rule, err := e.SaveRule(AutomationRule{Name: "r", Trigger: "manual", Commands: []string{"ps"}})
	if err != nil {
		t.Fatal(err)
	}
	e.store = &panicStore{}

	func() {
		defer func() { _ = recover() }()
		e.storeRun(AutomationRun{ID: "run-1", RuleID: rule.ID})
	}()

	acquired := make(chan int, 1)
	go func() {
		e.mu.RLock()
		rules := len(e.rules)
		e.mu.RUnlock()
		acquired <- rules
	}()
	select {
	case <-acquired:
	case <-time.After(2 * time.Second):
		t.Fatal("e.mu is still held after a panic while persisting")
	}
}
