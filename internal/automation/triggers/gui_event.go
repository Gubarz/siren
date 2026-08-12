package triggers

import (
	"context"
	"path/filepath"

	"siren/internal/automation"
	"siren/internal/bus"
)

func FileDownload(b bus.Bus) automation.Trigger {
	return newGUIEvent("file-download", "gui.file-downloaded", "remotePath", b)
}

func Screenshot(b bus.Bus) automation.Trigger {
	return newGUIEvent("screenshot", "gui.screenshot-taken", "", b)
}

func PayloadBuild(b bus.Bus) automation.Trigger {
	return newGUIEvent("payload-build", "gui.payload-built", "name", b)
}

type guiEventTrigger struct {
	triggerType, busType, globKey string
	b                             bus.Bus
}

func newGUIEvent(triggerType, busType, globKey string, b bus.Bus) automation.Trigger {
	return &guiEventTrigger{triggerType: triggerType, busType: busType, globKey: globKey, b: b}
}

func (t *guiEventTrigger) Type() string { return t.triggerType }

func (t *guiEventTrigger) ConfigSchema() []automation.FieldSpec {
	if t.globKey == "" {
		return nil
	}
	return []automation.FieldSpec{{
		Key: "glob", Label: "Path/name glob (optional)", Type: "string",
	}}
}

func (t *guiEventTrigger) Arm(ctx context.Context, cfg map[string]any, fire func(automation.FireEvent)) error {
	glob, _ := cfg["glob"].(string)
	if glob != "" {
		if _, err := filepath.Match(glob, ""); err != nil {
			return err
		}
	}
	unsub := t.b.Subscribe([]string{t.busType}, func(ev bus.Event) {
		payload, ok := ev.Payload.(map[string]any)
		if !ok {
			return
		}
		if glob != "" && t.globKey != "" {
			if ok, _ := filepath.Match(glob, str(payload, t.globKey)); !ok {
				return
			}
		}
		var target *automation.Target
		if id := str(payload, "sessionID"); id != "" {
			target = &automation.Target{ID: id, Kind: "session"}
		}
		fire(automation.FireEvent{Target: target, Data: payload})
	})
	defer unsub()
	<-ctx.Done()
	return ctx.Err()
}
