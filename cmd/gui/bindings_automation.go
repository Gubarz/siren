package gui

import (
	"fmt"
	"os"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"sliver-gui/internal/automation"
	"sort"
)

type AutomationCapabilitySpec struct {
	Type   string                  `json:"type"`
	Schema []automation.FieldSpec  `json:"schema"`
}

type AutomationCapabilities struct {
	Triggers []AutomationCapabilitySpec `json:"triggers"`
	Actions  []AutomationCapabilitySpec `json:"actions"`
}

func (a *App) ListAutomationCapabilities() (AutomationCapabilities, error) {
	capabilities := AutomationCapabilities{}
	for typ, schema := range a.Automation.TriggerSchemas() {
		capabilities.Triggers = append(capabilities.Triggers, AutomationCapabilitySpec{Type: typ, Schema: schema})
	}
	for typ, schema := range a.Automation.ActionSchemas() {
		capabilities.Actions = append(capabilities.Actions, AutomationCapabilitySpec{Type: typ, Schema: schema})
	}
	sort.Slice(capabilities.Triggers, func(i, j int) bool { return capabilities.Triggers[i].Type < capabilities.Triggers[j].Type })
	sort.Slice(capabilities.Actions, func(i, j int) bool { return capabilities.Actions[i].Type < capabilities.Actions[j].Type })
	return capabilities, nil
}

// ---- Automation ----

func (a *App) ListAutomationRules() ([]automation.AutomationRule, error) {
	return a.Automation.ListRules()
}

func (a *App) SaveAutomationRule(rule automation.AutomationRule) (automation.AutomationRule, error) {
	return a.Automation.SaveRule(rule)
}

func (a *App) DeleteAutomationRule(id string) error {
	return a.Automation.DeleteRule(id)
}

func (a *App) SetAutomationRuleEnabled(id string, enabled bool) error {
	return a.Automation.SetRuleEnabled(id, enabled)
}

func (a *App) RunAutomationRule(id, targetID string) error {
	return a.Automation.RunRule(id, targetID)
}

func (a *App) GetAutomationHistory() ([]automation.AutomationRun, error) {
	return a.Automation.GetHistory()
}

func (a *App) ClearAutomationHistory() error {
	return a.Automation.ClearHistory()
}

func (a *App) ExportAutomationRules(includeSecrets bool) (string, error) {
	raw, err := a.Automation.ExportRules(includeSecrets)
	if err != nil {
		return "", err
	}
	path, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "Export Automation Rules",
		DefaultFilename: fmt.Sprintf("sliver-automation-%s.json", time.Now().Format("2006-01-02")),
		Filters: []runtime.FileFilter{{
			DisplayName: "JSON files (*.json)",
			Pattern:     "*.json",
		}},
	})
	if err != nil || path == "" {
		return path, err
	}
	return path, os.WriteFile(path, []byte(raw), 0o600)
}

func (a *App) ImportAutomationRules(payload string) (automation.ImportResult, error) {
	return a.Automation.ImportRules(payload)
}

func (a *App) GetStarterAutomationRules() ([]automation.AutomationRule, error) {
	return a.Automation.StarterRules()
}

func (a *App) ImportStarterAutomationRule(id string) (automation.AutomationRule, error) {
	return a.Automation.ImportStarterRule(id)
}
