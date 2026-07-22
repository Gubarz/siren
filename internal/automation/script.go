package automation

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/grafana/sobek"
)

const (
	defaultAutomationTimeout = 5 * time.Minute
	maxAutomationTimeout     = 1 * time.Hour
	maxJSOutputSize          = 10 * 1024 * 1024
)

func (e *Engine) executeCommands(rule AutomationRule, target Target) (string, []string, error) {
	ctx, cancel := automationContext(e.ctx, rule.TimeoutSeconds)
	defer cancel()

	var output strings.Builder
	var commands []string
	var runErr error
	for index, command := range rule.Commands {
		command = renderAutomationCommand(command, target)
		if strings.TrimSpace(command) == "" {
			continue
		}
		commands = append(commands, command)
		if index > 0 && rule.DelaySeconds > 0 {
			select {
			case <-ctx.Done():
				runErr = ctx.Err()
			case <-time.After(time.Duration(rule.DelaySeconds) * time.Second):
			}
			if runErr != nil {
				break
			}
		}
		result, err := e.runAutomationCommand(ctx, target, command)
		appendAutomationCommandOutput(&output, command, result, err)
		if err != nil {
			runErr = err
			if !rule.ContinueOnError {
				break
			}
		}
	}
	return output.String(), commands, runErr
}

type jsExecution struct {
	engine    *Engine
	ctx       context.Context
	target    Target
	output    strings.Builder
	commands  []string
	truncated bool
}

func (je *jsExecution) appendLog(values ...interface{}) {
	if je.truncated {
		return
	}
	if je.output.Len() >= maxJSOutputSize {
		je.output.WriteString("\n... output truncated ...")
		je.truncated = true
		return
	}
	if je.output.Len() > 0 {
		je.output.WriteByte('\n')
	}
	for index, value := range values {
		if index > 0 {
			je.output.WriteByte(' ')
		}
		je.output.WriteString(fmt.Sprint(value))
	}
}

func (je *jsExecution) run(command string) (string, error) {
	if err := je.ctx.Err(); err != nil {
		return "", err
	}
	command = renderAutomationCommand(command, je.target)
	if strings.TrimSpace(command) == "" {
		return "", fmt.Errorf("command cannot be empty")
	}
	je.commands = append(je.commands, command)
	result, err := je.engine.runAutomationCommand(je.ctx, je.target, command)
	if !je.truncated {
		appendAutomationCommandOutput(&je.output, command, result, err)
		if je.output.Len() >= maxJSOutputSize {
			je.output.WriteString("\n... output truncated ...")
			je.truncated = true
		}
	}
	return result, err
}

func (je *jsExecution) sleep(milliseconds int64) error {
	if milliseconds < 0 {
		return fmt.Errorf("sleep duration cannot be negative")
	}
	timer := time.NewTimer(time.Duration(milliseconds) * time.Millisecond)
	defer timer.Stop()
	select {
	case <-je.ctx.Done():
		return je.ctx.Err()
	case <-timer.C:
		return nil
	}
}

func (je *jsExecution) listTags() []string {
	if je.engine.tags == nil || je.target.ID == "" {
		return nil
	}
	return je.engine.tags.GetAgentTags(je.target.ID)
}

func (je *jsExecution) addTags(values ...string) ([]string, error) {
	current := je.listTags()
	next := append(current, normalizeScriptTagArgs(values)...)
	return je.saveTags(next)
}

func (je *jsExecution) removeTags(values ...string) ([]string, error) {
	remove := tagSet(normalizeScriptTagArgs(values))
	current := je.listTags()
	next := make([]string, 0, len(current))
	for _, tag := range current {
		if _, ok := remove[strings.ToLower(strings.TrimSpace(tag))]; !ok {
			next = append(next, tag)
		}
	}
	return je.saveTags(next)
}

func (je *jsExecution) setTags(values ...string) ([]string, error) {
	return je.saveTags(normalizeScriptTagArgs(values))
}

func (je *jsExecution) saveTags(values []string) ([]string, error) {
	if je.engine.tags == nil {
		return nil, fmt.Errorf("tag store is unavailable")
	}
	if je.target.ID == "" {
		return nil, fmt.Errorf("target has no id")
	}
	if err := je.engine.tags.SetAgentTags(je.target.ID, values); err != nil {
		return nil, err
	}
	updated := je.engine.tags.GetAgentTags(je.target.ID)
	je.engine.emit("agent-tags-updated", je.target.ID)
	return updated, nil
}

func (je *jsExecution) setupVM(vm *sobek.Runtime, trigger string) error {
	targetValue := map[string]string{
		"id": je.target.ID, "name": je.target.Name, "hostname": je.target.Hostname,
		"username": je.target.Username, "os": je.target.OS, "arch": je.target.Arch, "kind": je.target.Kind,
	}
	if err := vm.Set("target", targetValue); err != nil {
		return err
	}
	if err := vm.Set("trigger", map[string]string{"type": trigger}); err != nil {
		return err
	}
	tags := map[string]interface{}{
		"add": je.addTags, "remove": je.removeTags, "set": je.setTags, "list": je.listTags,
	}
	if err := vm.Set("sliver", map[string]interface{}{
		"run": je.run, "log": je.appendLog, "sleep": je.sleep, "tags": tags,
	}); err != nil {
		return err
	}
	console := vm.NewObject()
	if err := console.Set("log", je.appendLog); err != nil {
		return err
	}
	return vm.Set("console", console)
}

func (e *Engine) executeJavaScript(
	rule AutomationRule,
	trigger string,
	target Target,
) (string, []string, error) {
	ctx, cancel := automationContext(e.ctx, rule.TimeoutSeconds)
	defer cancel()

	je := &jsExecution{
		engine: e,
		ctx:    ctx,
		target: target,
	}

	vm := sobek.New()
	vm.SetMaxCallStackSize(2048)
	vm.SetFieldNameMapper(sobek.TagFieldNameMapper("json", true))

	if err := je.setupVM(vm, trigger); err != nil {
		return "", nil, err
	}

	interruptDone := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			vm.Interrupt(ctx.Err())
		case <-interruptDone:
		}
	}()
	result, err := vm.RunString(rule.Script)
	close(interruptDone)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return je.output.String(), je.commands, fmt.Errorf(
				"JavaScript exceeded %s timeout",
				automationTimeout(rule.TimeoutSeconds),
			)
		}
		return je.output.String(), je.commands, fmt.Errorf("JavaScript: %w", err)
	}
	if result != nil && !sobek.IsUndefined(result) && !sobek.IsNull(result) {
		je.appendLog("Result:", result.Export())
	}
	return je.output.String(), je.commands, nil
}

func normalizeScriptTagArgs(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		for _, part := range strings.Split(value, ",") {
			if tag := strings.TrimSpace(part); tag != "" {
				out = append(out, tag)
			}
		}
	}
	return out
}

func tagSet(values []string) map[string]struct{} {
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		if tag := strings.ToLower(strings.TrimSpace(value)); tag != "" {
			set[tag] = struct{}{}
		}
	}
	return set
}

func (e *Engine) runAutomationCommand(
	ctx context.Context,
	target Target,
	command string,
) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	return e.executor.Execute(ctx, target.ID, target.Kind, command)
}

func automationContext(parent context.Context, timeoutSeconds int) (context.Context, context.CancelFunc) {
	if parent == nil {
		parent = context.Background()
	}
	return context.WithTimeout(parent, automationTimeout(timeoutSeconds))
}

func automationTimeout(timeoutSeconds int) time.Duration {
	timeout := time.Duration(timeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = defaultAutomationTimeout
	}
	if timeout > maxAutomationTimeout {
		timeout = maxAutomationTimeout
	}
	return timeout
}

func appendAutomationCommandOutput(output *strings.Builder, command, result string, err error) {
	if output.Len() > 0 {
		output.WriteString("\n\n")
	}
	fmt.Fprintf(output, "$ %s", command)
	if result != "" {
		output.WriteByte('\n')
		output.WriteString(result)
	}
	if err != nil {
		fmt.Fprintf(output, "\n[!] %v", err)
	}
}
