package bootstrap

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"sliver-gui/internal/automation"
	"sliver-gui/internal/automation/actions"
	"sliver-gui/internal/automation/triggers"
	"sliver-gui/internal/bus"
	"sliver-gui/internal/envvars"
	"sliver-gui/internal/journal"
	automationstate "sliver-gui/internal/localstate/automation"
	"sliver-gui/internal/localstate/casefile"
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
	DataDir   string
	GUIConfig *envvars.GUIConfig
	Emitter   automation.Emitter
}

type SharedStack struct {
	RPC        *rpc.Client
	Console    *console.Service
	Beacons    *beacons.Service
	Automation *automation.Engine
	CheckinPub *automationexec.CheckinPublisher
	LootWriter *automationexec.LootWriter
	Tags       *tags.Service
	Comments   *comments.Service
	Cases      *casefile.Service
	Events     *events.Store
	Bus        bus.Bus
	Journal    *journal.Service
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
	caseSvc := casefile.New(deps.DataDir)
	con.SetBus(busImpl)
	executor := automationexec.NewExecutor(con, beac)
	targets := automationexec.NewTargetProvider(rpcClient)
	lootWriter := automationexec.NewLootWriter(rpcClient)
	eng := automation.New(automation.Dependencies{
		Store: automationstate.New(deps.DataDir), Emitter: deps.Emitter,
		Executor: executor, Targets: targets, Tags: tagsSvc,
		Bus: busImpl, Journal: journalSvc, Cases: caseSvc,
		Loot: lootWriter,
	})
	registerBuiltinTriggers(eng, busImpl)
	registerBuiltinActions(eng)
	return &SharedStack{
		RPC: rpcClient, Console: con, Beacons: beac, Automation: eng,
		CheckinPub: automationexec.NewCheckinPublisher(rpcClient, busImpl),
		LootWriter: lootWriter,
		Tags: tagsSvc, Comments: commentsSvc, Cases: caseSvc,
		Events: eventsStore, Bus: busImpl, Journal: journalSvc,
	}
}

func registerBuiltinTriggers(eng *automation.Engine, b bus.Bus) {
	for _, t := range []automation.Trigger{
		triggers.Manual(),
		triggers.Interval(),
		triggers.SessionConnected(b),
		triggers.BeaconRegistered(b),
		triggers.BeaconCheckin(b),
		triggers.Cron(),
		triggers.TaskFinish(b),
		triggers.Keyword(b),
		triggers.FileDownload(b),
		triggers.Screenshot(b),
		triggers.PayloadBuild(b),
	} {
		if err := eng.RegisterTrigger(t); err != nil {
			log.Printf("bootstrap: register trigger: %v", err)
		}
	}
}

func registerBuiltinActions(eng *automation.Engine) {
	for _, a := range []automation.Action{
		actions.Commands(),
		actions.Script(),
		actions.Webhook(),
		actions.Notify(),
		actions.Tag(),
		actions.CaseAdd(),
	} {
		if err := eng.RegisterAction(a); err != nil {
			log.Printf("bootstrap: register action: %v", err)
		}
	}
}
