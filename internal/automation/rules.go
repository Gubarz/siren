package automation

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/grafana/sobek"
)

type AutomationFilter struct {
	OS       string `json:"os"`
	Arch     string `json:"arch"`
	Hostname string `json:"hostname"`
	Username string `json:"username"`
	Name     string `json:"name"`
}

type AutomationRule struct {
	ID              string           `json:"id"`
	Name            string           `json:"name"`
	Description     string           `json:"description"`
	Enabled         bool             `json:"enabled"`
	Trigger         string           `json:"trigger"`
	TargetKind      string           `json:"targetKind"`
	Filter          AutomationFilter `json:"filter"`
	ExecutionMode   string           `json:"executionMode"`
	Commands        []string         `json:"commands"`
	Script          string           `json:"script"`
	TimeoutSeconds  int              `json:"timeoutSeconds"`
	ContinueOnError bool             `json:"continueOnError"`
	DelaySeconds    int              `json:"delaySeconds"`
	CooldownSeconds int              `json:"cooldownSeconds"`
	IntervalSeconds int              `json:"intervalSeconds"`
	MaxRuns         int              `json:"maxRuns"`
	RunCount        int              `json:"runCount"`
	CreatedAt       int64            `json:"createdAt"`
	UpdatedAt       int64            `json:"updatedAt"`
}

type AutomationRun struct {
	ID         string   `json:"id"`
	RuleID     string   `json:"ruleId"`
	RuleName   string   `json:"ruleName"`
	Trigger    string   `json:"trigger"`
	TargetID   string   `json:"targetId"`
	TargetName string   `json:"targetName"`
	TargetKind string   `json:"targetKind"`
	Commands   []string `json:"commands"`
	Output     string   `json:"output"`
	Error      string   `json:"error"`
	Status     string   `json:"status"`
	StartedAt  int64    `json:"startedAt"`
	FinishedAt int64    `json:"finishedAt"`
}

type automationState struct {
	Rules   []AutomationRule `json:"rules"`
	History []AutomationRun  `json:"history"`
}

func (e *Engine) ListRules() ([]AutomationRule, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	rules := append([]AutomationRule{}, e.rules...)
	sort.SliceStable(rules, func(i, j int) bool {
		return strings.ToLower(rules[i].Name) < strings.ToLower(rules[j].Name)
	})
	return rules, nil
}

func (e *Engine) SaveRule(rule AutomationRule) (AutomationRule, error) {
	rule.Name = strings.TrimSpace(rule.Name)
	rule.Description = strings.TrimSpace(rule.Description)
	rule.TargetKind = strings.TrimSpace(rule.TargetKind)
	rule.ExecutionMode = automationExecutionMode(rule)
	if rule.ExecutionMode == ExecutionModeCommands {
		rule.Commands = compactCommands(rule.Commands)
	}
	if err := validateAutomationRule(rule); err != nil {
		return AutomationRule{}, err
	}
	now := time.Now().UnixMilli()
	e.mu.Lock()
	existing := e.ruleByIDLocked(rule.ID)
	if existing == nil {
		rule.ID = uuid.NewString()
		rule.CreatedAt = now
		rule.RunCount = 0
		e.rules = append(e.rules, rule)
	} else {
		rule.CreatedAt = existing.CreatedAt
		rule.RunCount = existing.RunCount
		*existing = rule
	}
	rule.UpdatedAt = now
	if current := e.ruleByIDLocked(rule.ID); current != nil {
		current.UpdatedAt = now
		rule = *current
	}
	if err := e.persistLocked(); err != nil {
		e.mu.Unlock()
		return AutomationRule{}, err
	}
	e.mu.Unlock()
	e.emit("automation-updated", rule)
	return rule, nil
}

func (e *Engine) DeleteRule(id string) error {
	e.mu.Lock()
	for index := range e.rules {
		if e.rules[index].ID == id {
			e.rules = append(e.rules[:index], e.rules[index+1:]...)
			err := e.persistLocked()
			e.mu.Unlock()
			if err == nil {
				e.emit("automation-updated", map[string]string{"deleted": id})
			}
			return err
		}
	}
	e.mu.Unlock()
	return fmt.Errorf("automation rule not found: %s", id)
}

func (e *Engine) SetRuleEnabled(id string, enabled bool) error {
	e.mu.Lock()
	rule := e.ruleByIDLocked(id)
	if rule == nil {
		e.mu.Unlock()
		return fmt.Errorf("automation rule not found: %s", id)
	}
	rule.Enabled = enabled
	rule.UpdatedAt = time.Now().UnixMilli()
	saved := *rule
	err := e.persistLocked()
	e.mu.Unlock()
	if err == nil {
		e.emit("automation-updated", saved)
	}
	return err
}

func (e *Engine) RunRule(id, targetID string) error {
	e.mu.RLock()
	rule := e.ruleByIDLocked(id)
	if rule == nil {
		e.mu.RUnlock()
		return fmt.Errorf("automation rule not found: %s", id)
	}
	copyRule := *rule
	e.mu.RUnlock()
	if targetID == "" {
		targets := e.currentTargets()
		matched := 0
		for _, target := range targets {
			if matchesAutomationRule(copyRule, target) {
				matched++
				e.queueRun(copyRule, "manual", target)
			}
		}
		if matched == 0 {
			return fmt.Errorf("no current targets match this rule")
		}
		return nil
	}

	var target automationTarget
	session, beacon, err := e.console.FindTarget(targetID)
	if err != nil {
		return err
	}
	if session != nil {
		target = targetFromSession(session)
	} else {
		target = targetFromBeacon(beacon)
	}
	if !matchesAutomationRule(copyRule, target) {
		return fmt.Errorf("target does not match this rule's filters")
	}
	e.queueRun(copyRule, "manual", target)
	return nil
}

func (e *Engine) GetHistory() ([]AutomationRun, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return append([]AutomationRun{}, e.history...), nil
}

func (e *Engine) ClearHistory() error {
	e.mu.Lock()
	e.history = nil
	err := e.persistLocked()
	e.mu.Unlock()
	if err == nil {
		e.emit("automation-run", map[string]bool{"cleared": true})
	}
	return err
}

func validateAutomationRule(rule AutomationRule) error {
	rule.Name = strings.TrimSpace(rule.Name)
	if rule.Name == "" {
		return fmt.Errorf("rule name is required")
	}
	switch rule.Trigger {
	case "session-connected", "beacon-registered", "beacon-checkin", "interval", "manual":
	default:
		return fmt.Errorf("unsupported trigger %q", rule.Trigger)
	}
	switch rule.TargetKind {
	case "", "any", "session", "beacon":
	default:
		return fmt.Errorf("unsupported target kind %q", rule.TargetKind)
	}
	switch automationExecutionMode(rule) {
	case ExecutionModeJavaScript:
		if strings.TrimSpace(rule.Script) == "" {
			return fmt.Errorf("JavaScript source is required")
		}
		if _, err := sobek.Compile(rule.Name+".js", rule.Script, true); err != nil {
			return fmt.Errorf("JavaScript: %w", err)
		}
		return nil
	case ExecutionModeCommands:
		for _, command := range rule.Commands {
			if strings.TrimSpace(command) != "" {
				return nil
			}
		}
		return fmt.Errorf("at least one command is required")
	default:
		return fmt.Errorf("unsupported execution mode %q", rule.ExecutionMode)
	}
}

func compactCommands(commands []string) []string {
	result := make([]string, 0, len(commands))
	for _, command := range commands {
		if command = strings.TrimSpace(command); command != "" {
			result = append(result, command)
		}
	}
	return result
}
