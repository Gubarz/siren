package automation

import (
	"context"
	"sort"
	"strings"
	"testing"
)

type memoryTagStore struct {
	tags map[string][]string
}

func (s *memoryTagStore) GetAgentTags(agentID string) []string {
	out := append([]string(nil), s.tags[agentID]...)
	return out
}

func (s *memoryTagStore) SetAgentTags(agentID string, values []string) error {
	normalized := normalizeTestTags(values)
	if len(normalized) == 0 {
		delete(s.tags, agentID)
		return nil
	}
	s.tags[agentID] = normalized
	return nil
}

type noopEmitter struct{}

func (noopEmitter) Emit(string, any) {}

func TestJavaScriptCanManageGUITags(t *testing.T) {
	store := &memoryTagStore{tags: map[string][]string{}}
	engine := &Engine{tags: store, emitter: noopEmitter{}}
	rule := AutomationRule{
		Name:          "tag test",
		ExecutionMode: ExecutionModeJavaScript,
		Script: `
const first = sliver.tags.add('windows, x64');
sliver.log(first.join('|'));
const second = sliver.tags.remove('x64');
sliver.log(second.join('|'));
const finalTags = sliver.tags.set(second.join(','), 'checked');
sliver.log(finalTags.join('|'));
`,
	}
	target := Target{ID: "session-1", Kind: "session", OS: "windows", Arch: "amd64"}

	output, commands, err := engine.executeJavaScript(context.Background(), rule, "manual", target)
	if err != nil {
		t.Fatalf("executeJavaScript() error = %v", err)
	}
	if len(commands) != 0 {
		t.Fatalf("expected no sliver commands, got %v", commands)
	}

	got := strings.Join(store.GetAgentTags("session-1"), "|")
	if got != "checked|windows" {
		t.Fatalf("tags = %q, want checked|windows\noutput:\n%s", got, output)
	}
	if !strings.Contains(output, "checked|windows") {
		t.Fatalf("output did not include final tags:\n%s", output)
	}
}

func TestStarterRulesParseAndCompile(t *testing.T) {
	engine := &Engine{triggers: map[string]Trigger{
		"manual":            triggersStub{typ: "manual"},
		"interval":          triggersStub{typ: "interval"},
		"session-connected": triggersStub{typ: "session-connected"},
		"beacon-registered": triggersStub{typ: "beacon-registered"},
		"beacon-checkin":    triggersStub{typ: "beacon-checkin"},
	}, actions: map[string]Action{
		"commands": actionStub{typ: "commands"},
		"script":   actionStub{typ: "script"},
	}}
	rules, err := engine.StarterRules()
	if err != nil {
		t.Fatalf("StarterRules() error = %v", err)
	}
	if len(rules) == 0 {
		t.Fatal("StarterRules() returned no rules")
	}
	for _, rule := range rules {
		migrateRule(&rule)
		if err := engine.validateAutomationRule(rule); err != nil {
			t.Fatalf("starter %q is invalid: %v", rule.Name, err)
		}
	}
}

type triggersStub struct {
	typ    string
	schema []FieldSpec
}

func (t triggersStub) Type() string                  { return t.typ }
func (t triggersStub) ConfigSchema() []FieldSpec     { return t.schema }
func (t triggersStub) Arm(ctx context.Context, cfg map[string]any, fire func(FireEvent)) error { <-ctx.Done(); return ctx.Err() }

var _ Trigger = triggersStub{}

type actionStub struct {
	typ    string
	schema []FieldSpec
}

func (a actionStub) Type() string                { return a.typ }
func (a actionStub) ConfigSchema() []FieldSpec   { return a.schema }
func (a actionStub) Execute(*RunContext) error   { return nil }

var _ Action = actionStub{}


func normalizeTestTags(values []string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, value := range values {
		for _, part := range strings.Split(value, ",") {
			tag := strings.ToLower(strings.TrimSpace(part))
			if tag == "" {
				continue
			}
			if _, ok := seen[tag]; ok {
				continue
			}
			seen[tag] = struct{}{}
			out = append(out, tag)
		}
	}
	sort.Strings(out)
	return out
}
