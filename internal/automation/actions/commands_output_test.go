package actions

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"siren/internal/automation"
)

type hugeOutputExec struct{ size int }

func (h hugeOutputExec) Execute(context.Context, string, string, string) (string, error) {
	return strings.Repeat("x", h.size), nil
}

// The run record is persisted with the automation state and the whole file is
// rewritten on every run, so command output has to be bounded the way the
// script action's output already is.
func TestCommandOutputIsBounded(t *testing.T) {
	var logged string
	rc := &automation.RunContext{
		Ctx:    context.Background(),
		Rule:   automation.AutomationRule{TimeoutSeconds: 30},
		Target: automation.Target{ID: "t1", Kind: "session"},
		Action: automation.ActionSpec{
			Type:   "commands",
			Config: map[string]any{"commands": []string{"ps", "ls"}},
		},
		Deps: automation.ActionDeps{Executor: hugeOutputExec{size: 8 << 20}},
		Log:  func(args ...any) { logged = fmt.Sprint(args...) },
	}

	if err := Commands().Execute(rc); err != nil {
		t.Fatalf("Execute() = %v, want nil", err)
	}
	if len(logged) > 12<<20 {
		t.Fatalf("accumulated %d bytes of command output, want it capped", len(logged))
	}
	if !strings.Contains(logged, "output truncated") {
		t.Fatal("capped output does not say it was truncated")
	}
}
