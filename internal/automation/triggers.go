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
	return registerIn(&e.registryMu, e.triggers, "trigger", t.Type(), t)
}

func (e *Engine) triggerByType(typ string) (Trigger, bool) {
	return lookupIn(&e.registryMu, e.triggers, typ)
}

func (e *Engine) TriggerSchemas() map[string][]FieldSpec {
	e.registryMu.RLock()
	defer e.registryMu.RUnlock()
	return collectSchemas(e.triggers)
}

// armedRule is one live trigger subscription. gen tells a re-armed rule apart
// from the subscription it replaced, so the superseded goroutine cannot tear
// down its replacement.
type armedRule struct {
	cancel context.CancelFunc
	gen    uint64
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
	e.armGen++
	gen := e.armGen
	e.armed[rule.ID] = &armedRule{cancel: cancel, gen: gen}
	e.armedMu.Unlock()
	cfg := triggerConfig(rule)
	go func() {
		defer e.disarmGeneration(rule.ID, gen)
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

// disarmRule cancels whatever subscription the rule currently holds. Use it for
// an explicit disarm; the arm goroutine uses disarmGeneration instead.
func (e *Engine) disarmRule(ruleID string) {
	e.armedMu.Lock()
	entry, ok := e.armed[ruleID]
	delete(e.armed, ruleID)
	e.armedMu.Unlock()
	if ok && entry.cancel != nil {
		entry.cancel()
	}
}

// disarmGeneration clears ruleID only while it still holds gen. Keying this on
// the rule ID alone let a superseded goroutine's deferred cleanup cancel the
// subscription that replaced it, which left the rule enabled but never firing.
func (e *Engine) disarmGeneration(ruleID string, gen uint64) {
	e.armedMu.Lock()
	entry, ok := e.armed[ruleID]
	current := ok && entry.gen == gen
	if current {
		delete(e.armed, ruleID)
	}
	e.armedMu.Unlock()
	if current && entry.cancel != nil {
		entry.cancel()
	}
}

func (e *Engine) disarmAll() {
	e.armedMu.Lock()
	cancels := make([]context.CancelFunc, 0, len(e.armed))
	for id, entry := range e.armed {
		delete(e.armed, id)
		if entry.cancel != nil {
			cancels = append(cancels, entry.cancel)
		}
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
	e.registryMu.RLock()
	defer e.registryMu.RUnlock()
	types := make([]string, 0, len(e.triggers))
	for typ := range e.triggers {
		types = append(types, typ)
	}
	sort.Strings(types)
	return types
}
