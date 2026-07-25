package bootstrap

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"sliver-gui/internal/automation"
	"sliver-gui/internal/bus"
	"sliver-gui/internal/envvars"
	"sliver-gui/internal/journal"
	automationstate "sliver-gui/internal/localstate/automation"
	"sliver-gui/internal/localstate/comments"
	"sliver-gui/internal/localstate/events"
	localjournal "sliver-gui/internal/localstate/journal"
	"sliver-gui/internal/localstate/tags"

	automationexec "sliver-gui/internal/sliver/automationexec"
	"sliver-gui/internal/sliver/beacons"
	"sliver-gui/internal/sliver/console"
	"sliver-gui/internal/sliver/rpc"

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
	Bus              bus.Bus
	Journal          *journal.Service
}

func resolveDataDir(deps Dependencies) string {
	if deps.DataDir != "" {
		return deps.DataDir
	}
	guiCfg := deps.GUIConfig
	if guiCfg == nil {
		guiCfg, _ = envvars.LoadGUIConfig(assets.GetRootAppDir())
	}
	dataDir, err := envvars.ResolveDataDir(guiCfg)
	if err != nil {
		dataDir = filepath.Join(os.TempDir(), fmt.Sprintf("sliver-gui-%d", os.Getpid()))
		_ = os.MkdirAll(dataDir, 0o700)
	}
	return dataDir
}

func NewShared(deps Dependencies) *SharedStack {
	deps.DataDir = resolveDataDir(deps)

	busImpl := bus.New()
	journalStore, err := localjournal.NewSQLiteStore(deps.DataDir)
	if err != nil {
		log.Printf("bootstrap: journal store unavailable, journal disabled: %v", err)
		journalStore = nil
	}
	journalSvc := journal.NewService(journalStore, busImpl)

	rpcClient := rpc.NewClient()
	rpcClient.JournalHook = rpc.NewJournalHook(journalSvc)
	con := console.New(rpcClient)
	beac := beacons.New(rpcClient, con)
	beac.SetJournal(journalSvc)
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
		Bus:              busImpl,
		Journal:          journalSvc,
	}
}
