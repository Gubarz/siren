package gui

import (
	"context"
	"sync"

	"github.com/bishopfox/sliver/client/assets"

	"sliver-gui/internal/bootstrap"
	"sliver-gui/internal/localstate/casefile"
	"sliver-gui/internal/sliver/agents"
	"sliver-gui/internal/sliver/armory"
	"sliver-gui/internal/sliver/builders"
	"sliver-gui/internal/sliver/catalog"
	"sliver-gui/internal/sliver/clientlog"
	"sliver-gui/internal/sliver/console"
	"sliver-gui/internal/sliver/crack"
	"sliver-gui/internal/sliver/discovery"
	"sliver-gui/internal/sliver/env"
	"sliver-gui/internal/sliver/extensions"
	"sliver-gui/internal/sliver/files"
	"sliver-gui/internal/sliver/health"
	"sliver-gui/internal/sliver/hosts"
	"sliver-gui/internal/sliver/implants"
	"sliver-gui/internal/sliver/listeners"
	"sliver-gui/internal/sliver/loot"
	"sliver-gui/internal/sliver/memfiles"
	"sliver-gui/internal/sliver/monitor"
	"sliver-gui/internal/sliver/pivots"
	"sliver-gui/internal/sliver/procs"
	"sliver-gui/internal/sliver/registry"
	"sliver-gui/internal/sliver/server"
	"sliver-gui/internal/sliver/services"
	"sliver-gui/internal/sliver/shells"
	"sliver-gui/internal/sliver/staging"
	"sliver-gui/internal/sliver/tunneling"
	"sliver-gui/internal/sliver/websites"
	"sliver-gui/internal/sliver/wireguard"
)

// App is the Wails-facing composition root. Keep construction and service
// ownership here; exported bridge methods belong in the domain-oriented
// bindings_*.go files.
type App struct {
	ctx          context.Context
	cancel       context.CancelFunc
	connectionMu sync.Mutex
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
	Cases      *casefile.Service
	Health     *health.Service
	Env        *env.Service
}

func NewApp() *App {
	configureDefaultArmory()
	shared := bootstrap.NewShared(bootstrap.Dependencies{
		DataDir: "", // let NewShared resolve via envvars
	})

	tun := tunneling.New(shared.RPC)
	app := &App{
		SharedStack: shared,
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
		Cases:       casefile.New(assets.GetRootAppDir()),
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
