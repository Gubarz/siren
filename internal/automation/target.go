package automation

import (
	"path/filepath"
	"strings"
)

type Target struct {
	ID          string
	Name        string
	Hostname    string
	Username    string
	OS          string
	Arch        string
	Kind        string
	LastCheckin int64
}

func displayTargetName(target Target) string {
	if target.Name != "" {
		return target.Name
	}
	if target.Hostname != "" {
		return target.Hostname
	}
	return target.ID
}

func matchesAutomationRule(rule AutomationRule, target Target) bool {
	if rule.TargetKind != "" && rule.TargetKind != "any" && rule.TargetKind != target.Kind {
		return false
	}
	return matchAutomationPattern(target.OS, rule.Filter.OS) &&
		matchAutomationPattern(target.Arch, rule.Filter.Arch) &&
		matchAutomationPattern(target.Hostname, rule.Filter.Hostname) &&
		matchAutomationPattern(target.Username, rule.Filter.Username) &&
		matchAutomationPattern(target.Name, rule.Filter.Name)
}

func matchAutomationPattern(value, patterns string) bool {
	patterns = strings.TrimSpace(patterns)
	if patterns == "" || patterns == "*" {
		return true
	}
	value = strings.ToLower(value)
	for _, candidate := range strings.Split(patterns, ",") {
		candidate = strings.ToLower(strings.TrimSpace(candidate))
		if candidate == "" {
			continue
		}
		if matched, err := filepath.Match(candidate, value); err == nil && matched {
			return true
		}
	}
	return false
}

const (
	ExecutionModeJavaScript = "javascript"
	ExecutionModeCommands   = "commands"
)

func automationExecutionMode(rule AutomationRule) string {
	if rule.ExecutionMode == ExecutionModeJavaScript {
		return ExecutionModeJavaScript
	}
	return ExecutionModeCommands
}
