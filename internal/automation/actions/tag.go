package actions

import (
	"fmt"
	"strings"

	"siren/internal/automation"
)

type tag struct{}

func Tag() automation.Action { return tag{} }

func (tag) Type() string { return "tag" }

func (tag) ConfigSchema() []automation.FieldSpec {
	return []automation.FieldSpec{
		{Key: "add", Label: "Tags to add (comma-separated)", Type: "string"},
		{Key: "remove", Label: "Tags to remove (comma-separated)", Type: "string"},
	}
}

func (tag) Execute(rc *automation.RunContext) error {
	if rc.Deps.Tags == nil {
		return fmt.Errorf("tag: tag store unavailable")
	}
	if rc.Target.ID == "" {
		return fmt.Errorf("tag: target has no id")
	}
	current := rc.Deps.Tags.GetAgentTags(rc.Target.ID)
	add := splitTagList(renderTemplate(cfgString("add", rc.Action.Config), rc.Target))
	remove := splitTagList(renderTemplate(cfgString("remove", rc.Action.Config), rc.Target))
	next := applyTagEdits(current, add, remove)
	if err := rc.Deps.Tags.SetAgentTags(rc.Target.ID, next); err != nil {
		return err
	}
	rc.Log(fmt.Sprintf("tags: %v", next))
	return nil
}

func splitTagList(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func applyTagEdits(current, add, remove []string) []string {
	removeSet := make(map[string]struct{}, len(remove))
	for _, t := range remove {
		removeSet[strings.ToLower(t)] = struct{}{}
	}
	out := make([]string, 0, len(current)+len(add))
	for _, t := range current {
		if _, ok := removeSet[strings.ToLower(t)]; !ok {
			out = append(out, t)
		}
	}
	for _, t := range add {
		if _, ok := removeSet[strings.ToLower(t)]; !ok {
			out = append(out, t)
			removeSet[strings.ToLower(t)] = struct{}{}
		}
	}
	return out
}
