package automation

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gubarz/revils/store"

	"siren/internal/bus"
)

type ActionSpec struct {
	Type   string         `json:"type"`
	Config map[string]any `json:"config,omitempty"`
}

type ActionResult struct {
	Type       string `json:"type"`
	Status     string `json:"status"`
	Output     string `json:"output,omitempty"`
	Error      string `json:"error,omitempty"`
	DurationMs int64  `json:"durationMs"`
}

type (
	RecordQuerier interface {
		Query(f store.Filter) ([]store.RecordRow, error)
	}
	HTTPDoer interface {
		Do(req *http.Request) (*http.Response, error)
	}
	CaseAppender interface {
		AppendNote(ctx context.Context, caseRef, markdown string) error
	}
	LootItem struct {
		Name string `json:"name"`
		Type string `json:"type"`
	}
	LootWriter interface {
		Add(ctx context.Context, name, lootType string, data []byte) error
		List(ctx context.Context) ([]LootItem, error)
	}
)

type ActionDeps struct {
	Executor  CommandExecutor
	Tags      AgentTagStore
	Emitter   Emitter
	Bus       bus.Bus
	Records   RecordQuerier
	HTTP      HTTPDoer
	Cases     CaseAppender
	Loot      LootWriter
	Collector CollectorStarter
}

type RunContext struct {
	Ctx         context.Context
	Rule        AutomationRule
	Trigger     string
	Target      Target
	RunID       string
	StageID     string
	Action      ActionSpec
	Log         func(...any)
	Commands    *[]string
	OutputSoFar func() string
	Deps        ActionDeps
}

type Action interface {
	Type() string
	ConfigSchema() []FieldSpec
	Execute(rc *RunContext) error
}

func (e *Engine) RegisterAction(a Action) error {
	return registerIn(&e.registryMu, e.actions, "action", a.Type(), a)
}

func (e *Engine) actionByType(typ string) (Action, bool) {
	return lookupIn(&e.registryMu, e.actions, typ)
}

func (e *Engine) ActionSchemas() map[string][]FieldSpec {
	e.registryMu.RLock()
	defer e.registryMu.RUnlock()
	return collectSchemas(e.actions)
}

func migrateRule(rule *AutomationRule) {
	if len(rule.Actions) > 0 {
		return
	}
	switch automationExecutionMode(*rule) {
	case ExecutionModeJavaScript:
		if rule.Script == "" {
			return
		}
		rule.Actions = []ActionSpec{{
			Type:   "script",
			Config: map[string]any{"source": rule.Script},
		}}
	default:
		commands := compactCommands(rule.Commands)
		if len(commands) == 0 {
			return
		}
		rule.Actions = []ActionSpec{{
			Type: "commands",
			Config: map[string]any{
				"commands":        commands,
				"delaySeconds":    float64(rule.DelaySeconds),
				"continueOnError": rule.ContinueOnError,
			},
		}}
	}
}

func (e *Engine) executeAction(rc *RunContext, spec ActionSpec) (result ActionResult) {
	result.Type = spec.Type
	result.Status = "ok"
	start := time.Now()
	defer func() {
		result.DurationMs = time.Since(start).Milliseconds()
		if r := recover(); r != nil {
			result.Status = "error"
			result.Error = fmt.Sprintf("panic: %v", r)
		}
	}()
	action, ok := e.actionByType(spec.Type)
	if !ok {
		result.Status = "error"
		result.Error = fmt.Sprintf("unregistered action %q", spec.Type)
		return result
	}
	if err := action.Execute(rc); err != nil {
		result.Status = "error"
		result.Error = err.Error()
	}
	return result
}

func (e *Engine) actionDeps() ActionDeps {
	return ActionDeps{
		Executor:  e.executor,
		Tags:      e.tags,
		Emitter:   e.emitter,
		Bus:       e.bus,
		Records:   e.records,
		HTTP:      e.httpClient(),
		Cases:     e.cases,
		Loot:      e.loot,
		Collector: e.collector(),
	}
}

// SetCollector wires the BloodHound collection starter. Called once at
// startup after the engine is constructed (the runner needs sliver services
// that do not exist at bootstrap time).
func (e *Engine) SetCollector(c CollectorStarter) {
	e.collectorPtr.Store(&c)
}

func (e *Engine) collector() CollectorStarter {
	if p := e.collectorPtr.Load(); p != nil {
		return *p
	}
	return nil
}

func (e *Engine) httpClient() HTTPDoer {
	if e.http != nil {
		return e.http
	}
	return &http.Client{Timeout: 15 * time.Second}
}
