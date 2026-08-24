package bootstrap

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"siren/internal/automation"
	"siren/internal/automation/actions"
	"siren/internal/automation/triggers"
	"siren/internal/bus"
	"siren/internal/envvars"
	"siren/internal/journal"
	automationstate "siren/internal/localstate/automation"
	"siren/internal/localstate/casefile"
	"siren/internal/localstate/comments"
	"siren/internal/localstate/events"
	localjournal "siren/internal/localstate/journal"
	"siren/internal/localstate/tags"

	automationexec "siren/internal/sliver/automationexec"
	"siren/internal/sliver/beacons"
	"siren/internal/sliver/console"
	"siren/internal/sliver/rpc"

	"github.com/bishopfox/sliver/client/assets"
)

type Dependencies struct {
	DataDir   string
	GUIConfig *envvars.GUIConfig
	Emitter   automation.Emitter
}

type SharedStack struct {
	DataDir    string
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
		dataDir = filepath.Join(os.TempDir(), fmt.Sprintf("siren-%d", os.Getpid()))
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
		DataDir: deps.DataDir,
		RPC:     rpcClient, Console: con, Beacons: beac, Automation: eng,
		CheckinPub: automationexec.NewCheckinPublisher(rpcClient, busImpl),
		LootWriter: lootWriter,
		Tags:       tagsSvc, Comments: commentsSvc, Cases: caseSvc,
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
		actions.BloodHoundCollect(),
	} {
		if err := eng.RegisterAction(a); err != nil {
			log.Printf("bootstrap: register action: %v", err)
		}
	}
}
