package automation

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

//go:embed starter-rules.json
var starterRulesData []byte

type ImportResult struct {
	Imported int      `json:"imported"`
	Skipped  int      `json:"skipped"`
	Errors   []string `json:"errors"`
}

func (e *Engine) ExportRules(includeSecrets bool) (string, error) {
	e.mu.RLock()
	rules := append([]AutomationRule{}, e.rules...)
	e.mu.RUnlock()
	if !includeSecrets {
		rules = scrubSecrets(rules)
	}
	data, err := json.MarshalIndent(map[string]any{"rules": rules}, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (e *Engine) ImportRules(payload string) (ImportResult, error) {
	result := ImportResult{}
	rules, err := parseRulesPayload(payload)
	if err != nil {
		return result, err
	}
	for _, rule := range rules {
		if _, err := e.importOne(rule); err != nil {
			result.Skipped++
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", rule.Name, err))
			continue
		}
		result.Imported++
	}
	return result, nil
}

func (e *Engine) StarterRules() ([]AutomationRule, error) {
	return parseRulesPayload(string(starterRulesData))
}

func (e *Engine) ImportStarterRule(id string) (AutomationRule, error) {
	starters, err := e.StarterRules()
	if err != nil {
		return AutomationRule{}, err
	}
	for _, rule := range starters {
		if rule.ID == id {
			return e.importOne(rule)
		}
	}
	return AutomationRule{}, fmt.Errorf("starter rule not found: %s", id)
}

func (e *Engine) importOne(rule AutomationRule) (AutomationRule, error) {
	rule.ID = ""
	rule.RunCount = 0
	rule.CreatedAt = 0
	rule.UpdatedAt = 0
	rule.Enabled = false
	if rule.ExecutionMode == "" {
		if len(rule.Commands) > 0 {
			rule.ExecutionMode = ExecutionModeCommands
		} else if rule.Script != "" {
			rule.ExecutionMode = ExecutionModeJavaScript
		}
	}
	return e.SaveRule(rule)
}

func parseRulesPayload(payload string) ([]AutomationRule, error) {
	trimmed := []byte(payload)
	if len(trimmed) == 0 {
		return nil, fmt.Errorf("empty payload")
	}
	var envelope struct {
		Rules []AutomationRule `json:"rules"`
	}
	if err := json.Unmarshal(trimmed, &envelope); err == nil && len(envelope.Rules) > 0 {
		return envelope.Rules, nil
	}
	var list []AutomationRule
	if err := json.Unmarshal(trimmed, &list); err == nil {
		return list, nil
	}
	var single AutomationRule
	if err := json.Unmarshal(trimmed, &single); err == nil && single.Name != "" {
		return []AutomationRule{single}, nil
	}
	return nil, fmt.Errorf("could not parse rules JSON")
}

func scrubSecrets(rules []AutomationRule) []AutomationRule {
	out := make([]AutomationRule, len(rules))
	for i, rule := range rules {
		out[i] = scrubRuleSecrets(rule)
	}
	return out
}

func scrubRuleSecrets(rule AutomationRule) AutomationRule {
	modified := false
	scrubbed := make([]ActionSpec, len(rule.Actions))
	copy(scrubbed, rule.Actions)
	for j, spec := range scrubbed {
		if spec.Type != "webhook" {
			continue
		}
		cfg := make(map[string]any, len(spec.Config))
		for k, v := range spec.Config {
			if k != "url" {
				cfg[k] = v
			}
		}
		scrubbed[j] = ActionSpec{Type: spec.Type, Config: cfg}
		modified = true
	}
	if modified {
		rule.Actions = scrubbed
	}
	return rule
}
