package automation

import (
	"path/filepath"
	"strings"

	"github.com/bishopfox/sliver/protobuf/clientpb"
)

type automationTarget struct {
	ID       string
	Name     string
	Hostname string
	Username string
	OS       string
	Arch     string
	Kind     string
}

func targetFromSession(session *clientpb.Session) automationTarget {
	return automationTarget{
		ID: session.ID, Name: session.Name, Hostname: session.Hostname,
		Username: session.Username, OS: session.OS, Arch: session.Arch, Kind: "session",
	}
}

func targetFromBeacon(beacon *clientpb.Beacon) automationTarget {
	return automationTarget{
		ID: beacon.ID, Name: beacon.Name, Hostname: beacon.Hostname,
		Username: beacon.Username, OS: beacon.OS, Arch: beacon.Arch, Kind: "beacon",
	}
}

func displayTargetName(target automationTarget) string {
	if target.Name != "" {
		return target.Name
	}
	if target.Hostname != "" {
		return target.Hostname
	}
	return target.ID
}

func matchesAutomationRule(rule AutomationRule, target automationTarget) bool {
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

func renderAutomationCommand(command string, target automationTarget) string {
	replacer := strings.NewReplacer(
		"{{id}}", target.ID,
		"{{name}}", target.Name,
		"{{hostname}}", target.Hostname,
		"{{username}}", target.Username,
		"{{os}}", target.OS,
		"{{arch}}", target.Arch,
		"{{kind}}", target.Kind,
	)
	return replacer.Replace(command)
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
