package actions

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/grafana/sobek"

	"sliver-gui/internal/automation"
)

const maxJSOutputSize = 10 * 1024 * 1024

type script struct{}

func Script() automation.Action { return script{} }

func (script) Type() string { return "script" }

func (script) ConfigSchema() []automation.FieldSpec {
	return []automation.FieldSpec{
		{Key: "source", Label: "JavaScript source", Type: "string", Required: true},
	}
}

func (script) Execute(rc *automation.RunContext) error {
	source := cfgString("source", rc.Action.Config)
	if source == "" {
		return fmt.Errorf("script: source is required")
	}
	timeout := timeoutFromRule(rc.Rule)
	ctx, cancel := context.WithTimeout(rc.Ctx, timeout)
	defer cancel()

	je := newJSExec(rc)
	vm := sobek.New()
	vm.SetMaxCallStackSize(2048)
	vm.SetFieldNameMapper(sobek.TagFieldNameMapper("json", true))

	if err := je.setupVM(vm); err != nil {
		return err
	}

	interruptDone := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			vm.Interrupt(ctx.Err())
		case <-interruptDone:
		}
	}()
	result, err := vm.RunString(source)
	close(interruptDone)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return fmt.Errorf("JavaScript exceeded %s timeout", timeout)
		}
		return fmt.Errorf("JavaScript: %w", err)
	}
	if result != nil && !sobek.IsUndefined(result) && !sobek.IsNull(result) {
		je.log("Result:", result.Export())
	}
	if je.output.Len() > 0 {
		rc.Log(je.output.String())
	}
	if rc.Commands != nil {
		*rc.Commands = append(*rc.Commands, je.commands...)
	}
	return nil
}

type jsExec struct {
	rc       *automation.RunContext
	output   strings.Builder
	commands []string
	trunc    bool
	vm       *sobek.Runtime
}

func newJSExec(rc *automation.RunContext) *jsExec {
	return &jsExec{rc: rc}
}

func (je *jsExec) setupVM(vm *sobek.Runtime) error {
	je.vm = vm
	targetValue := map[string]string{
		"id": je.rc.Target.ID, "name": je.rc.Target.Name, "hostname": je.rc.Target.Hostname,
		"username": je.rc.Target.Username, "os": je.rc.Target.OS, "arch": je.rc.Target.Arch, "kind": je.rc.Target.Kind,
	}
	if err := vm.Set("target", targetValue); err != nil {
		return err
	}
	if err := vm.Set("trigger", map[string]string{"type": je.rc.Trigger}); err != nil {
		return err
	}
	tags := map[string]interface{}{
		"add": je.addTags, "remove": je.removeTags, "set": je.setTags, "list": je.listTags,
	}
	sliverObj := map[string]interface{}{
		"run": je.run, "log": je.log, "sleep": je.sleep, "tags": tags,
	}
	sliverVal := vm.ToValue(sliverObj)
	if err := vm.Set("sliver", sliverVal); err != nil {
		return err
	}
	if err := je.extendAPI(vm, sliverVal); err != nil {
		return err
	}
	console := vm.NewObject()
	if err := console.Set("log", func(args ...interface{}) {
		je.log("console:", fmt.Sprint(args...))
	}); err != nil {
		return err
	}
	return vm.Set("console", console)
}

func (je *jsExec) log(values ...interface{}) {
	if je.trunc {
		return
	}
	if je.output.Len() >= maxJSOutputSize {
		je.output.WriteString("\n... output truncated ...")
		je.trunc = true
		return
	}
	if je.output.Len() > 0 {
		je.output.WriteByte('\n')
	}
	for index, value := range values {
		if index > 0 {
			je.output.WriteByte(' ')
		}
		fmt.Fprint(&je.output, value)
	}
}

func (je *jsExec) run(command string) (string, error) {
	if err := je.rc.Ctx.Err(); err != nil {
		return "", err
	}
	command = renderTemplate(command, je.rc.Target)
	command = strings.TrimSpace(command)
	if command == "" {
		return "", fmt.Errorf("command cannot be empty")
	}
	je.commands = append(je.commands, command)
	result, err := je.rc.Deps.Executor.Execute(je.rc.Ctx, je.rc.Target.ID, je.rc.Target.Kind, command)
	if !je.trunc {
		appendCmdOutput(&je.output, command, result, err)
		if je.output.Len() >= maxJSOutputSize {
			je.output.WriteString("\n... output truncated ...")
			je.trunc = true
		}
	}
	return result, err
}

func (je *jsExec) sleep(milliseconds int64) error {
	if milliseconds < 0 {
		return fmt.Errorf("sleep duration cannot be negative")
	}
	timer := time.NewTimer(time.Duration(milliseconds) * time.Millisecond)
	defer timer.Stop()
	select {
	case <-je.rc.Ctx.Done():
		return je.rc.Ctx.Err()
	case <-timer.C:
		return nil
	}
}

func (je *jsExec) listTags() []string {
	if je.rc.Deps.Tags == nil || je.rc.Target.ID == "" {
		return nil
	}
	return je.rc.Deps.Tags.GetAgentTags(je.rc.Target.ID)
}

func (je *jsExec) addTags(values ...string) ([]string, error) {
	current := je.listTags()
	next := append(current, normalizeTags(values)...)
	return je.saveTags(next)
}

func (je *jsExec) removeTags(values ...string) ([]string, error) {
	remove := tagSet(normalizeTags(values))
	current := je.listTags()
	next := make([]string, 0, len(current))
	for _, tag := range current {
		if _, ok := remove[strings.ToLower(strings.TrimSpace(tag))]; !ok {
			next = append(next, tag)
		}
	}
	return je.saveTags(next)
}

func (je *jsExec) setTags(values ...string) ([]string, error) {
	return je.saveTags(normalizeTags(values))
}

func (je *jsExec) saveTags(values []string) ([]string, error) {
	if je.rc.Deps.Tags == nil {
		return nil, fmt.Errorf("tag store is unavailable")
	}
	if je.rc.Target.ID == "" {
		return nil, fmt.Errorf("target has no id")
	}
	if err := je.rc.Deps.Tags.SetAgentTags(je.rc.Target.ID, values); err != nil {
		return nil, err
	}
	updated := je.rc.Deps.Tags.GetAgentTags(je.rc.Target.ID)
	if je.rc.Deps.Emitter != nil {
		je.rc.Deps.Emitter.Emit("agent-tags-updated", je.rc.Target.ID)
	}
	return updated, nil
}

func normalizeTags(values []string) []string {
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
