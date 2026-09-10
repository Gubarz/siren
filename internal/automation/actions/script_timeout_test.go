package actions

import (
	"context"
	"testing"
	"time"

	"siren/internal/automation"
)

// vm.Interrupt only takes effect at interpreter checkpoints, so it cannot
// preempt a host call like sliver.sleep. Without the script's own deadline
// reaching that call, the sleep runs to completion and the run never finalizes.
func TestScriptCannotSleepPastItsTimeout(t *testing.T) {
	rc := &automation.RunContext{
		Ctx:    context.Background(),
		Rule:   automation.AutomationRule{TimeoutSeconds: 1},
		Target: automation.Target{ID: "t1", Kind: "session"},
		Action: automation.ActionSpec{
			Type:   "script",
			Config: map[string]any{"source": "sliver.sleep(600000);"},
		},
	}

	done := make(chan error, 1)
	go func() { done <- Script().Execute(rc) }()

	select {
	case err := <-done:
		if err == nil {
			t.Fatal("script slept past its rule timeout and reported success")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("script outlived its rule timeout; the sleep was not bounded")
	}
}
