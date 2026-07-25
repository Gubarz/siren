package gui

import (
	"fmt"
	"os"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"sliver-gui/internal/automation"
)

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

func (a *App) ExportAutomationRules() (string, error) {
	raw, err := a.Automation.ExportRules()
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
