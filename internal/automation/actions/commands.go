package actions

import (
	"context"
	"fmt"
	"strings"
	"time"

	"siren/internal/automation"
)

type commands struct{}

func Commands() automation.Action { return commands{} }

func (commands) Type() string { return "commands" }

func (commands) ConfigSchema() []automation.FieldSpec {
	return []automation.FieldSpec{
		{Key: "commands", Label: "Commands (one per line)", Type: "string", Required: true},
		{Key: "delaySeconds", Label: "Delay between commands (s)", Type: "number"},
		{Key: "continueOnError", Label: "Continue on error", Type: "bool"},
	}
}

func (commands) Execute(rc *automation.RunContext) error {
	cfg := rc.Action.Config
	list := cfgStringList(cfg, "commands")
	delay := int(cfgFloat("delaySeconds", cfg, 0))
	continueOnError := cfgBool(cfg, "continueOnError")
	if len(list) == 0 {
		return nil
	}
	ctx, cancel := context.WithTimeout(rc.Ctx, timeoutFromRule(rc.Rule))
	defer cancel()
	return runCommandList(ctx, rc, list, delay, continueOnError)
}

func runCommandList(ctx context.Context, rc *automation.RunContext, list []string, delay int, continueOnErr bool) error {
	var output strings.Builder
	var ran []string
	var runErr error
	for index, command := range list {
		command = renderTemplate(command, rc.Target)
		command = strings.TrimSpace(command)
		if command == "" {
			continue
		}
		ran = append(ran, command)
		if index > 0 && delay > 0 {
			select {
			case <-ctx.Done():
				runErr = ctx.Err()
			case <-time.After(time.Duration(delay) * time.Second):
			}
			if runErr != nil {
				break
			}
		}
		result, err := rc.Deps.Executor.Execute(ctx, rc.Target.ID, rc.Target.Kind, command)
		appendCmdOutput(&output, command, result, err)
		if err != nil {
			runErr = err
			if !continueOnErr {
				break
			}
		}
		if output.Len() >= maxActionOutputSize {
			break
		}
	}
	if rc.Commands != nil {
		*rc.Commands = append(*rc.Commands, ran...)
	}
	if output.Len() > 0 {
		rc.Log(output.String())
	}
	return runErr
}

// maxActionOutputSize bounds what one action accumulates from command output.
// The run is persisted with the automation state and that file is rewritten on
// every run start and finish, so an unbounded result grows it without limit.
const maxActionOutputSize = 10 * 1024 * 1024

func appendCmdOutput(out *strings.Builder, command, result string, err error) {
	if out.Len() >= maxActionOutputSize {
		return
	}
	if out.Len() > 0 {
		out.WriteString("\n\n")
	}
	fmt.Fprintf(out, "$ %s", command)
	if result != "" {
		out.WriteByte('\n')
		out.WriteString(clipToBudget(result, maxActionOutputSize-out.Len()))
	}
	if err != nil {
		fmt.Fprintf(out, "\n[!] %v", err)
	}
}

// clipToBudget keeps the accumulated output within budget, marking it when the
// text had to be cut.
func clipToBudget(text string, budget int) string {
	if budget <= 0 {
		return ""
	}
	if len(text) <= budget {
		return text
	}
	return text[:budget] + "\n... output truncated ..."
}

func timeoutFromRule(rule automation.AutomationRule) time.Duration {
	const defaultTimeout = 5 * time.Minute
	const maxTimeout = 1 * time.Hour
	timeout := time.Duration(rule.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	if timeout > maxTimeout {
		timeout = maxTimeout
	}
	return timeout
}
