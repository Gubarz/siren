package triggers

import (
	"context"

	"sliver-gui/internal/automation"
	"sliver-gui/internal/bus"
	"sliver-gui/internal/journal"
)

func TaskFinish(b bus.Bus) automation.Trigger {
	schema := []automation.FieldSpec{
		{Key: "verb", Label: "Journal verb", Type: "string", Default: "BeaconTaskResult"},
		{Key: "targetID", Label: "Target ID (optional)", Type: "string"},
	}
	return &taskFinish{schema: schema, b: b}
}

type taskFinish struct {
	schema []automation.FieldSpec
	b      bus.Bus
}

func (t *taskFinish) Type() string                        { return "task-finish" }
func (t *taskFinish) ConfigSchema() []automation.FieldSpec { return t.schema }

func (t *taskFinish) Arm(ctx context.Context, cfg map[string]any, fire func(automation.FireEvent)) error {
	verb, _ := cfg["verb"].(string)
	if verb == "" {
		verb = "BeaconTaskResult"
	}
	targetFilter, _ := cfg["targetID"].(string)
	unsub := t.b.Subscribe([]string{"journal.action-recorded"}, func(ev bus.Event) {
		entry, ok := ev.Payload.(journal.Entry)
		if !ok || entry.Verb != verb {
			return
		}
		if targetFilter != "" && entry.TargetID != targetFilter {
			return
		}
		fire(automation.FireEvent{
			Target: &automation.Target{
				ID: entry.TargetID, Kind: entry.TargetKind, Hostname: entry.Hostname,
			},
			Data: map[string]any{"entry": entry},
		})
	})
	defer unsub()
	<-ctx.Done()
	return ctx.Err()
}
