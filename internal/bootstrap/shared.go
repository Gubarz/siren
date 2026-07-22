package bootstrap

import (
	automationstate "sliver-gui/internal/localstate/automation"

	"sliver-gui/internal/automation"
	"sliver-gui/internal/localstate/comments"
	"sliver-gui/internal/localstate/events"
	"sliver-gui/internal/localstate/tags"
	automationexec "sliver-gui/internal/sliver/automationexec"
	"sliver-gui/internal/sliver/beacons"
	"sliver-gui/internal/sliver/console"
	"sliver-gui/internal/sliver/rpc"
)

type Dependencies struct {
	DataDir string
	Emitter automation.Emitter
}

type SharedStack struct {
	RPC              *rpc.Client
	Console          *console.Service
	Beacons          *beacons.Service
	Automation       *automation.Engine
	AutomationEvents *automationexec.EventSource
	Tags             *tags.Service
	Comments         *comments.Service
	Events           *events.Store
}

func NewShared(deps Dependencies) *SharedStack {
	rpcClient := rpc.NewClient()
	con := console.New(rpcClient)
	beac := beacons.New(rpcClient, con)
	tagsSvc := tags.New()
	commentsSvc := comments.New()
	eventsStore := events.New()

	store := automationstate.New(deps.DataDir)
	executor := automationexec.NewExecutor(con, beac)
	targets := automationexec.NewTargetProvider(rpcClient)
	automationEvents := automationexec.NewEventSource()

	eng := automation.New(automation.Dependencies{
		Store:    store,
		Emitter:  deps.Emitter,
		Executor: executor,
		Targets:  targets,
		Events:   automationEvents,
		Tags:     tagsSvc,
	})

	return &SharedStack{
		RPC:              rpcClient,
		Console:          con,
		Beacons:          beac,
		Automation:       eng,
		AutomationEvents: automationEvents,
		Tags:             tagsSvc,
		Comments:         commentsSvc,
		Events:           eventsStore,
	}
}
