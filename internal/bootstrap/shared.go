package bootstrap

import (
	"fmt"
	"os"
	"path/filepath"

	"sliver-gui/internal/automation"
	automationstate "sliver-gui/internal/localstate/automation"
	"sliver-gui/internal/localstate/comments"
	"sliver-gui/internal/localstate/events"
	"sliver-gui/internal/localstate/tags"

	automationexec "sliver-gui/internal/sliver/automationexec"
	"sliver-gui/internal/sliver/beacons"
	"sliver-gui/internal/sliver/console"
	"sliver-gui/internal/sliver/rpc"

	"sliver-gui/internal/envvars"

	"github.com/bishopfox/sliver/client/assets"
)

type Dependencies struct {
	DataDir     string
	GUIConfig   *envvars.GUIConfig
	Emitter     automation.Emitter
	StartEvents bool
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
	guiCfg := deps.GUIConfig
	if guiCfg == nil {
		guiCfg, _ = envvars.LoadGUIConfig(assets.GetRootAppDir())
	}
	if deps.DataDir == "" {
		dataDir, err := envvars.ResolveDataDir(guiCfg)
		if err != nil {
			dataDir = filepath.Join(os.TempDir(), fmt.Sprintf("sliver-gui-%d", os.Getpid()))
			_ = os.MkdirAll(dataDir, 0o700)
		}
		deps.DataDir = dataDir
	}

	rpcClient := rpc.NewClient()
	con := console.New(rpcClient)
	beac := beacons.New(rpcClient, con)
	tagsSvc := tags.New(deps.DataDir)
	commentsSvc := comments.New(deps.DataDir)
	eventsStore := events.New(deps.DataDir)

	store := automationstate.New(deps.DataDir)
	executor := automationexec.NewExecutor(con, beac)
	targets := automationexec.NewTargetProvider(rpcClient)

	var automationEvents *automationexec.EventSource
	if deps.StartEvents {
		automationEvents = automationexec.NewEventSource(rpcClient)
	} else {
		automationEvents = automationexec.NewEventSource(nil)
	}

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
