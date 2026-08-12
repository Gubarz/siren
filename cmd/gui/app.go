package gui

import (
	"context"
	"sync"

	"github.com/bishopfox/sliver/client/assets"
	"github.com/wailsapp/wails/v3/pkg/application"

	"siren/internal/bootstrap"
	"siren/internal/sliver/agents"
	"siren/internal/sliver/armory"
	"siren/internal/sliver/builders"
	"siren/internal/sliver/catalog"
	"siren/internal/sliver/clientlog"
	"siren/internal/sliver/console"
	"siren/internal/sliver/crack"
	"siren/internal/sliver/discovery"
	"siren/internal/sliver/env"
	"siren/internal/sliver/extensions"
	"siren/internal/sliver/files"
	"siren/internal/sliver/health"
	"siren/internal/sliver/hosts"
	"siren/internal/sliver/implants"
	"siren/internal/sliver/listeners"
	"siren/internal/sliver/loot"
	"siren/internal/sliver/memfiles"
	"siren/internal/sliver/monitor"
	"siren/internal/sliver/pivots"
	"siren/internal/sliver/procs"
	"siren/internal/sliver/registry"
	"siren/internal/sliver/server"
	"siren/internal/sliver/services"
	"siren/internal/sliver/shells"
	"siren/internal/sliver/staging"
	"siren/internal/sliver/tunneling"
	"siren/internal/sliver/websites"
	"siren/internal/sliver/wireguard"
	"siren/internal/wailsadapter"
)

// App is the Wails-facing composition root. Keep construction and service
// ownership here; exported bridge methods belong in the domain-oriented
// bindings_*.go files.
type App struct {
	ctx          context.Context
	cancel       context.CancelFunc
	connectionMu sync.Mutex
	wails        *application.App
	window       *application.WebviewWindow
	bridge       *wailsadapter.Bridge
	*bootstrap.SharedStack

	Catalog    *catalog.Service
	Agents     *agents.Service
	Armory     *armory.Service
	Implants   *implants.Service
	Listeners  *listeners.Service
	Files      *files.Service
	Procs      *procs.Service
	Registry   *registry.Service
	Shells     *shells.Service
	Pivots     *pivots.Service
	Services   *services.Service
	Tunneling  *tunneling.Service
	Loot       *loot.Service
	Server     *server.Service
	Monitor    *monitor.Service
	Extensions *extensions.Service
	Memfiles   *memfiles.Service
	WireGuard  *wireguard.Service
	Crack      *crack.Service
	Builders   *builders.Service
	Discovery  *discovery.Service
	ClientLog  *clientlog.Service
	Websites   *websites.Service
	Staging    *staging.Service
	Hosts      *hosts.Service
	Health     *health.Service
	Env        *env.Service
}

func NewApp(wailsApp *application.App, window *application.WebviewWindow) *App {
	configureDefaultArmory()
	shared := bootstrap.NewShared(bootstrap.Dependencies{
		DataDir: "", // let NewShared resolve via envvars
	})

	tun := tunneling.New(shared.RPC)
	app := &App{
		SharedStack: shared,
		wails:       wailsApp,
		window:      window,
		bridge:      wailsadapter.New(wailsApp),
		Catalog:     catalog.New(shared.Console),
		Agents:      agents.New(shared.RPC, shared.Console),
		Armory:      armory.New(shared.Console),
		Implants:    implants.New(shared.RPC),
		Listeners:   listeners.New(shared.RPC),
		Files:       files.New(shared.RPC),
		Procs:       procs.New(shared.RPC),
		Registry:    registry.New(shared.RPC),
		Shells:      shells.New(shared.RPC, shared.Console),
		Pivots:      pivots.New(shared.RPC),
		Services:    services.New(shared.RPC),
		Tunneling:   tun,
		Loot:        loot.New(shared.RPC),
		Server:      server.New(shared.RPC),
		Discovery:   discovery.New(shared.Console, shared.Beacons),
		ClientLog:   clientlog.New(shared.RPC),
		Websites:    websites.New(shared.RPC),
		Staging:     staging.New(shared.RPC),
		Hosts:       hosts.New(shared.RPC),
		Monitor:     monitor.New(shared.RPC),
		Extensions:  extensions.New(shared.RPC),
		Memfiles:    memfiles.New(shared.RPC),
		WireGuard:   wireguard.New(shared.RPC),
		Crack:       crack.New(shared.RPC),
		Builders:    builders.New(shared.RPC),
		Env:         env.New(shared.RPC),
	}
	app.Files.SetBus(shared.Bus)
	app.Procs.SetBus(shared.Bus)
	app.Implants.SetBus(shared.Bus)
	app.Builders.SetBus(shared.Bus)
	app.Loot.SetBus(shared.Bus)
	shared.LootWriter.SetService(app.Loot)
	app.Console.SetRoutedCommandHandler(func(sessionID, line string) console.RoutedCommandResult {
		result := app.Tunneling.HandleConsoleTunnelCommand(sessionID, line)
		return console.RoutedCommandResult{
			Handled: result.Handled,
			Output:  result.Output,
			Refresh: result.Refresh,
		}
	})
	app.Health = health.New(shared.RPC, tun)
	return app
}

func configureDefaultArmory() {
	const (
		publicKey = "RWSBpxpRWDrD7Fe+VvRE3c2VEDC2NK80rlNCj+BX0gz44Xw07r6KQD9L"
		repoURL   = "https://api.github.com/repos/sliverarmory/armory/releases"
	)

	if assets.DefaultArmoryPublicKey == "" {
		assets.DefaultArmoryPublicKey = publicKey
	}
	if assets.DefaultArmoryRepoURL == "" {
		assets.DefaultArmoryRepoURL = repoURL
	}
	assets.DefaultArmoryConfig.PublicKey = assets.DefaultArmoryPublicKey
	assets.DefaultArmoryConfig.RepoURL = assets.DefaultArmoryRepoURL
}
