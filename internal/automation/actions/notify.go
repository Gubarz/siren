package actions

import (
	"fmt"

	"siren/internal/automation"
)

type notify struct{}

func Notify() automation.Action { return notify{} }

func (notify) Type() string { return "notify" }

func (notify) ConfigSchema() []automation.FieldSpec {
	return []automation.FieldSpec{
		{Key: "severity", Label: "Severity", Type: "select", Options: []string{"info", "warning", "error"}, Default: "info"},
		{Key: "message", Label: "Message", Type: "string", Required: true},
	}
}

func (notify) Execute(rc *automation.RunContext) error {
	message := renderTemplate(cfgString("message", rc.Action.Config), rc.Target)
	if rc.Deps.Emitter == nil {
		rc.Log("notify:", message)
		return nil
	}
	rc.Deps.Emitter.Emit("automation-notify", map[string]any{
		"severity": cfgString("severity", rc.Action.Config),
		"message":  message,
		"rule":     rc.Rule.Name,
		"target":   rc.Target.Name,
	})
	rc.Log(fmt.Sprintf("notify: %s", message))
	return nil
}
