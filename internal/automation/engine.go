package automation

import (
	"context"
	"log"
	"sync"
	"time"
)

const (
	automationHistoryLimit = 500
	beaconPollInterval     = 5 * time.Second
)

type Engine struct {
	store    StateStore
	emitter  Emitter
	executor CommandExecutor
	targets  TargetProvider
	events   EventSource
	tags     AgentTagStore
	ctx      context.Context

	mu             sync.RWMutex
	rules          []AutomationRule
	history        []AutomationRun
	running        map[string]bool
	activeByRule   map[string]int
	lastRun        map[string]time.Time
	lastInterval   map[string]time.Time
	beaconCheckins map[string]int64
	beaconsPrimed  bool
}

func New(deps Dependencies) *Engine {
	e := &Engine{
		store:          deps.Store,
		emitter:        deps.Emitter,
		executor:       deps.Executor,
		targets:        deps.Targets,
		events:         deps.Events,
		tags:           deps.Tags,
		running:        map[string]bool{},
		activeByRule:   map[string]int{},
		lastRun:        map[string]time.Time{},
		lastInterval:   map[string]time.Time{},
		beaconCheckins: map[string]int64{},
	}
	if state, err := deps.Store.Load(context.Background()); err != nil {
		log.Printf("automation: could not load state: %v", err)
	} else if state != nil {
		e.rules = state.Rules
		e.history = state.History
	}
	return e
}

func (e *Engine) SetEmitter(emitter Emitter) {
	e.emitter = emitter
}

func (e *Engine) Start(ctx context.Context) {
	e.ctx = ctx
	e.events.Start(ctx, func(trigger string, target Target) {
		if trigger == "beacon-registered" && target.LastCheckin > 0 {
			e.mu.Lock()
			e.beaconCheckins[target.ID] = target.LastCheckin
			e.mu.Unlock()
		}
		e.dispatchTrigger(trigger, target)
	})
	go func() {
		ticker := time.NewTicker(beaconPollInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case now := <-ticker.C:
				e.tick(now)
			}
		}
	}()
}

func (e *Engine) SetServer(host string, port uint32) {
	e.store.SetServer(host, port)
	e.mu.Lock()
	defer e.mu.Unlock()
	e.rules = nil
	e.history = nil
	if state, err := e.store.Load(context.Background()); err != nil {
		log.Printf("automation: could not load state for %s:%d: %v", host, port, err)
	} else if state != nil {
		e.rules = state.Rules
		e.history = state.History
	}
}

func (e *Engine) persistLocked() error {
	return e.store.Save(context.Background(), &State{Rules: e.rules, History: e.history})
}

func (e *Engine) tick(now time.Time) {
	if !e.targets.Connected() {
		return
	}
	e.pollBeaconCheckins()

	e.mu.RLock()
	rules := append([]AutomationRule(nil), e.rules...)
	e.mu.RUnlock()
	for _, rule := range rules {
		if !rule.Enabled || rule.Trigger != "interval" {
			continue
		}
		interval := time.Duration(rule.IntervalSeconds) * time.Second
		if interval < 10*time.Second {
			interval = 10 * time.Second
		}
		e.mu.Lock()
		last := e.lastInterval[rule.ID]
		if last.IsZero() {
			e.lastInterval[rule.ID] = now
			e.mu.Unlock()
			continue
		}
		if now.Sub(last) < interval {
			e.mu.Unlock()
			continue
		}
		e.lastInterval[rule.ID] = now
		e.mu.Unlock()
		e.dispatchRule(rule, "interval", nil)
	}
}

func (e *Engine) pollBeaconCheckins() {
	beacons, err := e.targets.GetBeacons(context.Background())
	if err != nil {
		return
	}
	e.mu.RLock()
	previous := make(map[string]int64, len(e.beaconCheckins))
	for id, checkin := range e.beaconCheckins {
		previous[id] = checkin
	}
	primed := e.beaconsPrimed
	e.mu.RUnlock()

	current := make(map[string]int64, len(beacons))
	for _, beacon := range beacons {
		current[beacon.ID] = beacon.LastCheckin
		if primed && previous[beacon.ID] != 0 && beacon.LastCheckin > previous[beacon.ID] {
			e.dispatchTrigger("beacon-checkin", beacon)
		}
	}
	e.mu.Lock()
	e.beaconCheckins = current
	e.beaconsPrimed = true
	e.mu.Unlock()
}

func (e *Engine) emit(name string, payload any) {
	if e.emitter != nil {
		e.emitter.Emit(name, payload)
	}
}

func (e *Engine) ruleByIDLocked(id string) *AutomationRule {
	for index := range e.rules {
		if e.rules[index].ID == id {
			return &e.rules[index]
		}
	}
	return nil
}
