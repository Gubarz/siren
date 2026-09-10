package automation

import (
	"context"
	"log"
	"sort"
)

type FieldSpec struct {
	Key      string   `json:"key"`
	Label    string   `json:"label"`
	Type     string   `json:"type"`
	Required bool     `json:"required,omitempty"`
	Options  []string `json:"options,omitempty"`
	Default  any      `json:"default,omitempty"`
}

type FireEvent struct {
	Target *Target
	Data   map[string]any
}

type Trigger interface {
	Type() string
	ConfigSchema() []FieldSpec
	Arm(ctx context.Context, cfg map[string]any, fire func(FireEvent)) error
}

type configValidator interface {
	Validate(cfg map[string]any) error
}

func (e *Engine) RegisterTrigger(t Trigger) error {
	return registerIn(&e.triggersMu, e.triggers, "trigger", t.Type(), t)
}

func (e *Engine) triggerByType(typ string) (Trigger, bool) {
	return lookupIn(&e.triggersMu, e.triggers, typ)
}

func (e *Engine) TriggerSchemas() map[string][]FieldSpec {
	e.triggersMu.RLock()
	defer e.triggersMu.RUnlock()
	return collectSchemas(e.triggers)
}

func (e *Engine) armRule(rule AutomationRule) {
	if e.ctx == nil {
		return
	}
	trigger, ok := e.triggerByType(rule.Trigger)
	if !ok {
		return
	}
	e.disarmRule(rule.ID)
	ctx, cancel := context.WithCancel(e.ctx)
	e.armedMu.Lock()
	e.armed[rule.ID] = cancel
	e.armedMu.Unlock()
	cfg := triggerConfig(rule)
	go func() {
		defer e.disarmRule(rule.ID)
		_ = trigger.Arm(ctx, cfg, func(fe FireEvent) {
			defer func() {
				if r := recover(); r != nil {
					e.logf("trigger %s panicked in fire: %v", rule.Trigger, r)
				}
			}()
			e.fireRule(rule, fe)
		})
	}()
}

func (e *Engine) disarmRule(ruleID string) {
	e.armedMu.Lock()
	cancel, ok := e.armed[ruleID]
	delete(e.armed, ruleID)
	e.armedMu.Unlock()
	if ok && cancel != nil {
		cancel()
	}
}

func (e *Engine) disarmAll() {
	e.armedMu.Lock()
	cancels := make([]context.CancelFunc, 0, len(e.armed))
	for id, cancel := range e.armed {
		delete(e.armed, id)
		cancels = append(cancels, cancel)
	}
	e.armedMu.Unlock()
	for _, cancel := range cancels {
		cancel()
	}
}

func (e *Engine) armedCount() int {
	e.armedMu.Lock()
	defer e.armedMu.Unlock()
	return len(e.armed)
}

func (e *Engine) fireRule(rule AutomationRule, fe FireEvent) {
	if fe.Target != nil {
		if matchesAutomationRule(rule, *fe.Target) {
			e.queueRun(rule, rule.Trigger, *fe.Target)
		}
		return
	}
	e.dispatchRule(rule, rule.Trigger, nil)
}

func triggerConfig(rule AutomationRule) map[string]any {
	cfg := make(map[string]any, len(rule.TriggerConfig)+1)
	for k, v := range rule.TriggerConfig {
		cfg[k] = v
	}
	if _, ok := cfg["intervalSeconds"]; !ok && rule.IntervalSeconds > 0 {
		cfg["intervalSeconds"] = float64(rule.IntervalSeconds)
	}
	return cfg
}

func (e *Engine) logf(format string, args ...any) {
	log.Printf("automation: "+format, args...)
}

func sortedTriggerTypes(e *Engine) []string {
	e.triggersMu.RLock()
	defer e.triggersMu.RUnlock()
	types := make([]string, 0, len(e.triggers))
	for typ := range e.triggers {
		types = append(types, typ)
	}
	sort.Strings(types)
	return types
}
