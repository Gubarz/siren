package automation

import (
	"context"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"siren/internal/bus"
)

const automationHistoryLimit = 500

type Engine struct {
	store    StateStore
	emitter  Emitter
	executor CommandExecutor
	targets  TargetProvider
	tags     AgentTagStore
	bus      bus.Bus
	records  RecordQuerier
	http     HTTPDoer
	cases    CaseAppender
	loot     LootWriter
	ctx      context.Context

	// Three locks, and no two of them are ever held at once: each is taken and
	// released before the next is acquired, so there is no lock order to get
	// wrong. Keep it that way.
	//
	//	registryMu guards the trigger and action registries. They are written
	//	           once at startup and read on every fire and rule validation.
	//	armedMu    guards the live trigger subscriptions and the generation
	//	           counter that tells a re-armed rule from the one it replaced.
	//	mu         guards the rules, their run history, and the counters that
	//	           decide whether a rule may fire again.
	registryMu sync.RWMutex
	triggers   map[string]Trigger
	actions    map[string]Action

	armedMu sync.Mutex
	armed   map[string]*armedRule
	armGen  uint64

	mu           sync.RWMutex
	rules        []AutomationRule
	history      []AutomationRun
	running      map[string]bool
	activeByRule map[string]int
	lastRun      map[string]time.Time

	// collectorPtr is wired once after construction, so it needs no lock.
	collectorPtr atomic.Pointer[CollectorStarter]
}

func New(deps Dependencies) *Engine {
	e := &Engine{
		store:        deps.Store,
		emitter:      deps.Emitter,
		executor:     deps.Executor,
		targets:      deps.Targets,
		tags:         deps.Tags,
		bus:          deps.Bus,
		records:      deps.Records,
		http:         deps.HTTP,
		cases:        deps.Cases,
		loot:         deps.Loot,
		triggers:     map[string]Trigger{},
		actions:      map[string]Action{},
		armed:        map[string]*armedRule{},
		running:      map[string]bool{},
		activeByRule: map[string]int{},
		lastRun:      map[string]time.Time{},
	}
	if state, err := deps.Store.Load(context.Background()); err != nil {
		log.Printf("automation: could not load state: %v", err)
	} else if state != nil {
		e.rules = state.Rules
		e.history = state.History
		for i := range e.rules {
			migrateRule(&e.rules[i])
		}
	}
	return e
}

func (e *Engine) SetEmitter(emitter Emitter) {
	e.emitter = emitter
}

func (e *Engine) Start(ctx context.Context) {
	e.ctx = ctx
	e.mu.RLock()
	rules := append([]AutomationRule(nil), e.rules...)
	e.mu.RUnlock()
	for _, rule := range rules {
		if rule.Enabled {
			e.armRule(rule)
		}
	}
}

func (e *Engine) SetServer(host string, port uint32) {
	e.disarmAll()
	// The write lock is taken before the path swap, not after. persistLocked
	// runs under it, so swapping first let an in-flight run write the previous
	// server's rules and history into the new server's file.
	e.mu.Lock()
	defer e.mu.Unlock()
	e.store.SetServer(host, port)
	e.rules = nil
	e.history = nil
	if state, err := e.store.Load(context.Background()); err != nil {
		log.Printf("automation: could not load state for %s:%d: %v", host, port, err)
	} else if state != nil {
		e.rules = state.Rules
		e.history = state.History
		for i := range e.rules {
			migrateRule(&e.rules[i])
		}
	}
}

func (e *Engine) persistLocked() error {
	return e.store.Save(context.Background(), &State{Rules: e.rules, History: e.history})
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
