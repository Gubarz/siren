package automation

import (
	"context"
	"sync"
	"testing"
	"time"
)

// ctxRecordingTrigger keeps every context it was armed with, so a test can tell
// whether the current subscription is still alive.
type ctxRecordingTrigger struct {
	typ string

	mu       sync.Mutex
	contexts []context.Context
}

func (f *ctxRecordingTrigger) Type() string              { return f.typ }
func (f *ctxRecordingTrigger) ConfigSchema() []FieldSpec { return nil }

func (f *ctxRecordingTrigger) Arm(ctx context.Context, _ map[string]any, _ func(FireEvent)) error {
	f.mu.Lock()
	f.contexts = append(f.contexts, ctx)
	f.mu.Unlock()
	<-ctx.Done()
	return ctx.Err()
}

func (f *ctxRecordingTrigger) armCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.contexts)
}

func (f *ctxRecordingTrigger) latestContext() context.Context {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.contexts) == 0 {
		return nil
	}
	return f.contexts[len(f.contexts)-1]
}

func waitForArmCount(t *testing.T, tr *ctxRecordingTrigger, want int) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if tr.armCount() >= want {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %d arm calls, got %d", want, tr.armCount())
}

// Saving an enabled rule re-arms it, so the subscription being replaced has to
// shut down without touching the one that replaced it.
func TestRearmingKeepsTheNewSubscriptionAlive(t *testing.T) {
	e := newTestEngine(t)
	tr := &ctxRecordingTrigger{typ: "rec"}
	_ = e.RegisterTrigger(tr)

	rule, err := e.SaveRule(AutomationRule{Name: "r", Trigger: "rec", Commands: []string{"ps"}, Enabled: false})
	if err != nil {
		t.Fatal(err)
	}
	if err := e.SetRuleEnabled(rule.ID, true); err != nil {
		t.Fatal(err)
	}
	waitForArmCount(t, tr, 1)

	rule.Enabled = true
	if _, err := e.SaveRule(rule); err != nil {
		t.Fatal(err)
	}
	waitForArmCount(t, tr, 2)

	// The superseded goroutine wakes as soon as its own context is canceled and
	// then runs its deferred cleanup. Give that cleanup time to cancel the wrong
	// subscription, which is what a rule-ID-keyed disarm did.
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if err := tr.latestContext().Err(); err != nil {
			t.Fatalf("re-arming canceled the new subscription: %v", err)
		}
		time.Sleep(10 * time.Millisecond)
	}
	if got := e.armedCount(); got != 1 {
		t.Fatalf("armedCount() = %d, want 1", got)
	}
}
