package automation

import (
	"context"

	"siren/internal/bus"
)

type Emitter interface {
	Emit(name string, payload any)
}

type CommandExecutor interface {
	Execute(ctx context.Context, targetID string, targetKind string, command string) (string, error)
}

type TargetProvider interface {
	Connected() bool
	GetSessions(ctx context.Context) ([]Target, error)
	GetBeacons(ctx context.Context) ([]Target, error)
	FindTarget(ctx context.Context, targetID string) (Target, error)
}

type StateStore interface {
	Load(ctx context.Context) (*State, error)
	Save(ctx context.Context, state *State) error
	SetServer(host string, port uint32)
}

type AgentTagStore interface {
	GetAgentTags(agentID string) []string
	SetAgentTags(agentID string, tags []string) error
}

// CollectorRequest carries one BloodHound collection request across the
// automation boundary (the concrete runner type lives outside this package).
type CollectorRequest struct {
	Collector      string
	Methods        []string
	Flags          []string
	Domain         string
	TimeoutSeconds int
	Ingest         bool
	Loot           bool
}

// CollectorProgress is a snapshot of a running collection.
type CollectorProgress struct {
	Stage string
	Error string
}

// CollectorStarter launches BloodHound collections and reports progress.
// Implementations are wired in after the engine is constructed (the runner
// needs sliver services), so the field is settable post-New.
type CollectorStarter interface {
	StartCollection(ctx context.Context, agentID, agentKind, agentOS string, req CollectorRequest) (string, error)
	CollectionState(ctx context.Context, id string) (CollectorProgress, bool)
}

type Dependencies struct {
	Store     StateStore
	Emitter   Emitter
	Executor  CommandExecutor
	Targets   TargetProvider
	Tags      AgentTagStore
	Bus       bus.Bus
	Journal   JournalQuerier
	HTTP      HTTPDoer
	Cases     CaseAppender
	Loot      LootWriter
	Collector CollectorStarter
}
