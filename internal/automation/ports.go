package automation

import "context"

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

type EventHandler func(trigger string, target Target)

type EventSource interface {
	Start(ctx context.Context, handler EventHandler)
	Stop()
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

type Dependencies struct {
	Store    StateStore
	Emitter  Emitter
	Executor CommandExecutor
	Targets  TargetProvider
	Events   EventSource
	Tags     AgentTagStore
}
