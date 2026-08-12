package actions

import (
	"fmt"
	"strings"

	"siren/internal/automation"
)

type caseAdd struct{}

func CaseAdd() automation.Action { return caseAdd{} }

func (caseAdd) Type() string { return "case-add" }

func (caseAdd) ConfigSchema() []automation.FieldSpec {
	return []automation.FieldSpec{
		{Key: "case", Label: "Case name or ID", Type: "string", Required: true},
		{Key: "itemType", Label: "Item type", Type: "select",
			Options: []string{"run-summary", "command-list", "output-excerpt"}, Default: "run-summary"},
	}
}

func (caseAdd) Execute(rc *automation.RunContext) error {
	if rc.Deps.Cases == nil {
		return fmt.Errorf("case-add: case service unavailable")
	}
	caseRef := cfgString("case", rc.Action.Config)
	if caseRef == "" {
		return fmt.Errorf("case-add: case is required")
	}
	note := renderCaseNote(rc, cfgString("itemType", rc.Action.Config))
	if err := rc.Deps.Cases.AppendNote(rc.Ctx, caseRef, note); err != nil {
		return err
	}
	rc.Log(fmt.Sprintf("case %s: appended %s", caseRef, cfgString("itemType", rc.Action.Config)))
	return nil
}

func renderCaseNote(rc *automation.RunContext, itemType string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "### Automation run — %s\n", rc.Rule.Name)
	fmt.Fprintf(&b, "- Trigger: `%s` · Target: `%s` (%s)\n", rc.Trigger, rc.Target.Name, rc.Target.ID)
	switch itemType {
	case "command-list":
		if rc.Commands != nil && len(*rc.Commands) > 0 {
			b.WriteString("```\n" + strings.Join(*rc.Commands, "\n") + "\n```\n")
		}
	case "output-excerpt":
		if tail := tailString(rc, 4096); tail != "" {
			b.WriteString("```\n" + tail + "\n```\n")
		}
	default:
		fmt.Fprintf(&b, "- Run ID: `%s`\n", rc.RunID)
	}
	return b.String()
}
