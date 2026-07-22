package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/bishopfox/sliver/client/assets"
	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/wailsapp/wails/v2/pkg/runtime"

	"sliver-gui/internal/agents"
	"sliver-gui/internal/armory"
	"sliver-gui/internal/automation"
	"sliver-gui/internal/beacons"
	"sliver-gui/internal/bootstrap"
	"sliver-gui/internal/builders"
	"sliver-gui/internal/buildinfo"
	"sliver-gui/internal/casefile"
	"sliver-gui/internal/casereport"
	"sliver-gui/internal/catalog"
	"sliver-gui/internal/clientlog"
	"sliver-gui/internal/comments"
	"sliver-gui/internal/console"
	"sliver-gui/internal/crack"
	"sliver-gui/internal/discovery"
	"sliver-gui/internal/events"
	"sliver-gui/internal/extensions"
	"sliver-gui/internal/files"
	"sliver-gui/internal/health"
	"sliver-gui/internal/hosts"
	"sliver-gui/internal/implants"
	"sliver-gui/internal/listeners"
	"sliver-gui/internal/loot"
	"sliver-gui/internal/memfiles"
	"sliver-gui/internal/monitor"
	"sliver-gui/internal/pivots"
	"sliver-gui/internal/procs"
	"sliver-gui/internal/registry"
	"sliver-gui/internal/rpc"
	"sliver-gui/internal/server"
	"sliver-gui/internal/services"
	uishellcode "sliver-gui/internal/shellcode"
	"sliver-gui/internal/shells"
	"sliver-gui/internal/staging"
	"sliver-gui/internal/theme"
	"sliver-gui/internal/tunneling"
	"sliver-gui/internal/wailsadapter"
	"sliver-gui/internal/websites"
	"sliver-gui/internal/wireguard"
)

type App struct {
	ctx    context.Context
	cancel context.CancelFunc
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
}

func NewApp() *App {
	configureDefaultArmory()
	shared := bootstrap.NewShared(bootstrap.Dependencies{
		DataDir: assets.GetRootAppDir(),
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
		Cases:       casefile.New(),
		Monitor:     monitor.New(shared.RPC),
		Extensions:  extensions.New(shared.RPC),
		Memfiles:    memfiles.New(shared.RPC),
		WireGuard:   wireguard.New(shared.RPC),
		Crack:       crack.New(shared.RPC),
		Builders:    builders.New(shared.RPC),
	}
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

func (a *App) GetBuildInfo() buildinfo.Info {
	return buildinfo.Get()
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

func (a *App) startup(ctx context.Context) {
	a.ctx, a.cancel = context.WithCancel(ctx)

	emitter := wailsadapter.New(ctx)
	a.Console.SetEmitter(emitter)
	a.Automation.SetEmitter(emitter)

	a.Implants.SetCtx(ctx)
	a.Files.SetCtx(ctx)
	a.Loot.SetCtx(ctx)
	a.Shells.SetCtx(ctx)
	a.Discovery.SetCtx(ctx)
	a.Staging.SetCtx(ctx)

	a.Automation.Start(ctx)

	go func() {
		time.Sleep(300 * time.Millisecond)
		runtime.WindowShow(ctx)
	}()
}

func (a *App) shutdown(context.Context) {
	preserveLiveResources := a.shouldPreserveLiveResourcesOnShutdown()
	if preserveLiveResources {
		log.Printf("shutdown: dev build detected; preserving live tunnels, shells, and consoles")
	} else {
		//
		// Tear down live tunnels/shells FIRST while the RPC is still up. Sliver's
		// server has historically panicked when the client's gRPC connection drops
		// mid-tunnel (socks/portfwd goroutines writing into a closed stream), so
		// we close them cleanly through the RPC before killing the connection.
		if a.Tunneling != nil {
			a.Tunneling.Close()
		}
		if a.Shells != nil {
			a.Shells.Close()
		}
		if a.Console != nil {
			a.Console.CloseSubprocs()
		}
	}
	if a.Crack != nil {
		a.Crack.Close()
	}
	if a.Monitor != nil {
		a.Monitor.Close()
	}
	if a.WireGuard != nil {
		a.WireGuard.Close()
	}
	if a.Extensions != nil {
		a.Extensions.Close()
	}
	if a.Memfiles != nil {
		a.Memfiles.Close()
	}
	if a.Builders != nil {
		a.Builders.Close()
	}
	if a.Websites != nil {
		a.Websites.Close()
	}
	if a.Staging != nil {
		a.Staging.Close()
	}
	if a.Hosts != nil {
		a.Hosts.Close()
	}
	if a.Tags != nil {
		a.Tags.Close()
	}
	if a.Cases != nil {
		a.Cases.Close()
	}
	if a.Health != nil {
		a.Health.Close()
	}
	if a.ClientLog != nil {
		a.ClientLog.Close()
	}
	if a.RPC != nil && !preserveLiveResources {
		a.RPC.Disconnect()
	}
	a.Events.Close()
	if a.cancel != nil {
		a.cancel()
	}
}

func (a *App) shouldPreserveLiveResourcesOnShutdown() bool {
	if a.ctx == nil {
		return false
	}
	env := runtime.Environment(a.ctx)
	return strings.EqualFold(env.BuildType, "dev")
}

func (a *App) OpenFileDialog(title string) (string, error) {
	return runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: title,
	})
}

func (a *App) startEventStream() {
	a.RPC.StartEventStream(a.ctx, func(ev *clientpb.Event) {
		if ev.EventType == "stream-closed" {
			a.RPC.InvalidateAgentCache()
			a.Console.ResetConsole()
			runtime.EventsEmit(a.ctx, "sliver-event", map[string]interface{}{"type": "stream-closed"})
			return
		}
		switch ev.EventType {
		case consts.SessionOpenedEvent, consts.SessionClosedEvent, consts.BeaconRegisteredEvent:
			a.RPC.InvalidateAgentCache()
		}
		if a.AutomationEvents != nil {
			a.AutomationEvents.HandleSliverEvent(ev)
		}
		payload := map[string]interface{}{"type": ev.EventType}
		if ev.Session != nil {
			payload["sessionID"] = ev.Session.ID
			payload["hostname"] = ev.Session.Hostname
			payload["username"] = ev.Session.Username
		}
		if ev.Job != nil {
			payload["job"] = ev.Job.Name
		}
		if len(ev.Data) > 0 {
			payload["data"] = string(ev.Data)
		}
		se := events.StoredEvent{Type: ev.EventType, Data: string(ev.Data), Time: time.Now().UnixMilli()}
		if ev.Session != nil {
			se.SessionID = ev.Session.ID
			se.Hostname = ev.Session.Hostname
			se.Username = ev.Session.Username
		}
		if ev.Job != nil {
			se.Job = ev.Job.Name
		}
		a.Events.Append(se)
		runtime.EventsEmit(a.ctx, "sliver-event", payload)
	})
}

func (a *App) GetEventHistory(since int64, limit int) ([]events.StoredEvent, error) {
	return a.Events.Query(since, limit), nil
}

func (a *App) SetEventsAcknowledged(seqs []int64, acked bool) int {
	return a.Events.SetAcked(seqs, acked)
}

// ---- Connection / RPC ----

func (a *App) GetClientConfigs() ([]string, error) {
	return a.RPC.GetClientConfigs()
}

func (a *App) GetClientConfigDetails() ([]rpc.ClientConfigSummary, error) {
	return a.RPC.GetClientConfigDetails()
}

func (a *App) ImportClientConfig(payload string) (string, error) {
	return a.RPC.ImportClientConfig(payload)
}

func (a *App) ExportClientConfig(name string) (string, error) {
	return a.RPC.ExportClientConfig(name)
}

func (a *App) DeleteClientConfig(name string) error {
	return a.RPC.DeleteClientConfig(name)
}

func (a *App) Disconnect() error {
	if a.Tunneling != nil {
		a.Tunneling.Close()
	}
	if a.Console != nil {
		_ = a.Console.TryResetConsole()
	}
	a.RPC.Disconnect()
	return nil
}

func (a *App) Connect(profileName string) error {
	if !a.Console.TryResetConsole() {
		log.Printf("connect: console is busy; skipping connect-time console reset")
	}
	if err := a.RPC.Connect(profileName); err != nil {
		return err
	}
	if a.RPC.Config != nil {
		a.Automation.SetServer(a.RPC.Config.LHost, uint32(a.RPC.Config.LPort))
		a.Tags.SetServer(a.RPC.Config.LHost, uint32(a.RPC.Config.LPort))
		a.Comments.SetServer(a.RPC.Config.LHost, uint32(a.RPC.Config.LPort))
	}
	if a.ClientLog != nil {
		if err := a.ClientLog.Start(a.ctx); err != nil {
			log.Printf("connect: client log stream failed: %v", err)
		}
	}
	a.startEventStream()
	return nil
}

// ---- Sessions / Beacons ----

func (a *App) GetSessions() (*clientpb.Sessions, error) {
	return a.Agents.Sessions()
}

func (a *App) GetBeacons() (*clientpb.Beacons, error) {
	return a.Agents.Beacons()
}

func (a *App) KillAgent(id string) error {
	return a.Agents.Kill(id)
}

func (a *App) RemoveBeacon(id string) error {
	return a.Agents.RemoveBeacon(id)
}

func (a *App) RenameAgent(id, name string) error {
	return a.Agents.Rename(id, name)
}

func (a *App) GetVersion() (*clientpb.Version, error) {
	return a.Agents.Version()
}

func (a *App) ListArmoryPackages(refresh bool) ([]armory.PackageInfo, error) {
	return a.Armory.ListPackages(refresh)
}

// ---- Pivots ----

func (a *App) GetPivots() (*clientpb.PivotGraph, error) {
	return a.Pivots.GetPivots()
}

func (a *App) GetPivotListeners() ([]pivots.PivotListenerSnapshot, error) {
	return a.Pivots.GetPivotListeners()
}

// ---- Operators ----

func (a *App) GetOperators() (*clientpb.Operators, error) {
	return a.Server.GetOperators()
}

// ---- Console / Commands ----

func (a *App) RunSessionCommand(sessionID, line string) (string, error) {
	output, err := a.Console.RunLine(sessionID, line)
	if err == nil && sessionID != "" && a.Console.CommandInvokesPing(line) {
		a.Discovery.HandlePingOutput(sessionID, output)
	}
	return output, err
}

// StartConsole spawns a per-session sliver client subprocess attached to
// a PTY. The frontend drives the returned jobID via WriteConsole /
// ResizeConsole / StopConsole and receives bytes on the "console-output"
// event. See internal/console/subproc.go.
func (a *App) StartConsole(sessionID string) (string, error) {
	return a.Console.StartConsole(sessionID)
}

func (a *App) WriteConsole(jobID, data string) error {
	return a.Console.WriteConsole(jobID, []byte(data))
}

func (a *App) ResizeConsole(jobID string, cols, rows int) error {
	return a.Console.ResizeConsole(jobID, cols, rows)
}

func (a *App) StopConsole(jobID string) error {
	return a.Console.StopConsole(jobID)
}

// SendToSessionConsole is what GUI actions (palette, right-click,
// panels) should call to run a command against a session — it routes
// via the session's live subprocess console so any interactive prompt
// the command triggers (forms.Select, tea programs) renders in xterm.js
// instead of leaking to the launching terminal. If no console is up
// yet, the line queues and runs on the next StartConsole.
func (a *App) SendToSessionConsole(sessionID, line string) error {
	return a.Console.SendToSessionConsole(sessionID, line)
}

func (a *App) ListCommands() ([]string, error) {
	return a.Console.ListCommands()
}

func (a *App) CompleteCommand(sessionID, line string) ([]string, error) {
	return a.Console.CompleteCommand(sessionID, line)
}

func (a *App) CompletePath(sessionID, partial string) ([]string, error) {
	return a.Console.CompletePath(sessionID, partial)
}

func (a *App) GetCommandCatalog(scope string) (*catalog.CommandCatalog, error) {
	return a.Catalog.GetCommandCatalog(scope)
}

// ---- Beacons ----

func (a *App) GetBeaconTasks(beaconID string) (*clientpb.BeaconTasks, error) {
	return a.Beacons.GetBeaconTasks(beaconID)
}

func (a *App) GetBeaconTaskOutput(taskID string) (string, error) {
	return a.Beacons.GetBeaconTaskOutput(taskID)
}

func (a *App) CancelBeaconTask(taskID string) error {
	return a.Beacons.CancelBeaconTask(taskID)
}

// ---- Process / File Tools ----

func (a *App) GetProcessList(sessionID string, fullInfo bool) (*sliverpb.Ps, error) {
	return a.Procs.GetProcessList(sessionID, fullInfo)
}

func (a *App) KillProcess(sessionID string, pid int32) error {
	return a.Procs.KillProcess(sessionID, pid)
}

func (a *App) TakeScreenshot(sessionID string) (string, error) {
	return a.Procs.TakeScreenshot(sessionID)
}

func (a *App) GetFileList(sessionID, path string) (*sliverpb.Ls, error) {
	return a.Files.GetFileList(sessionID, path)
}

func (a *App) DownloadFile(sessionID, remotePath string) error {
	return a.Files.DownloadFile(sessionID, remotePath)
}

func (a *App) DownloadDirectory(sessionID, remotePath string) error {
	return a.Files.DownloadDirectory(sessionID, remotePath)
}

func (a *App) DownloadMultipleTar(sessionID string, items []files.BulkDownloadItem) error {
	return a.Files.DownloadMultipleTar(sessionID, items)
}

func (a *App) GetDownloadHistory(sessionID, remotePath string) ([]files.DownloadRecord, error) {
	return a.Files.GetDownloadHistory(sessionID, remotePath)
}

func (a *App) GetAllDownloadHistory() ([]files.DownloadRecord, error) {
	return a.Files.GetAllDownloadHistory()
}

func (a *App) ClearDownloadHistory(sessionID, remotePath string) error {
	return a.Files.ClearDownloadHistory(sessionID, remotePath)
}

func (a *App) UploadFile(sessionID, remotePath string) error {
	return a.Files.UploadFile(sessionID, remotePath)
}

func (a *App) UploadFiles(sessionID, remotePath string, localPaths []string) error {
	return a.Files.UploadFiles(sessionID, remotePath, localPaths)
}

func (a *App) Cd(sessionID, path string) (string, error) {
	return a.Files.Cd(sessionID, path)
}

func (a *App) Pwd(sessionID string) (string, error) {
	return a.Files.Pwd(sessionID)
}

func (a *App) MakeDir(sessionID, path string) error {
	return a.Files.MakeDir(sessionID, path)
}

func (a *App) RemovePath(sessionID, path string, recursive bool) error {
	return a.Files.RemovePath(sessionID, path, recursive)
}

func (a *App) RenamePath(sessionID, src, dst string) error {
	return a.Files.RenamePath(sessionID, src, dst)
}

func (a *App) Chmod(sessionID, path, mode string, recursive bool) error {
	return a.Files.Chmod(sessionID, path, mode, recursive)
}

func (a *App) Chown(sessionID, path, uid, gid string, recursive bool) error {
	return a.Files.Chown(sessionID, path, uid, gid, recursive)
}

func (a *App) Chtimes(sessionID, path string, atimeUnix, mtimeUnix int64) error {
	return a.Files.Chtimes(sessionID, path, atimeUnix, mtimeUnix)
}

func (a *App) CopyPath(sessionID, src, dst string) (int64, error) {
	return a.Files.CopyPath(sessionID, src, dst)
}

func (a *App) ViewRemoteFile(sessionID, remotePath string) (string, error) {
	return a.Files.ViewRemoteFile(sessionID, remotePath)
}

func (a *App) GrepFiles(sessionID, pattern, path string, recursive bool, beforeLines, afterLines int32) (string, error) {
	return a.Files.GrepFiles(sessionID, pattern, path, recursive, beforeLines, afterLines)
}

func (a *App) GetServices(sessionID string) (*sliverpb.Services, error) {
	return a.Services.GetServices(sessionID)
}

func (a *App) StartService(sessionID, name string) error {
	return a.Services.StartService(sessionID, name)
}

func (a *App) StopService(sessionID, name string) error {
	return a.Services.StopService(sessionID, name)
}

func (a *App) RemoveService(sessionID, name string) error {
	return a.Services.RemoveService(sessionID, name)
}

// ---- Tunneling ----

func (a *App) StartSocks(sessionID, bindAddr, username, password string) (uint64, error) {
	return a.Tunneling.StartSocks(sessionID, bindAddr, username, password)
}

func (a *App) StopSocks(id uint64) error {
	return a.Tunneling.StopSocks(id)
}

func (a *App) StartPortfwd(sessionID, bindAddr, remoteAddr string) (uint64, error) {
	return a.Tunneling.StartPortfwd(sessionID, bindAddr, remoteAddr)
}

func (a *App) StopPortfwd(id uint64) error {
	return a.Tunneling.StopPortfwd(id)
}

func (a *App) StartRportfwd(sessionID, bindAddr string, bindPort int, forwardAddr string, forwardPort int) (uint64, error) {
	return a.Tunneling.StartRportfwd(sessionID, bindAddr, bindPort, forwardAddr, forwardPort)
}

func (a *App) StopRportfwd(id uint64, sessionID string) error {
	return a.Tunneling.StopRportfwd(id, sessionID)
}

func (a *App) ListRportfwds() ([]tunneling.ProxyInfo, error) {
	return a.Tunneling.ListRportfwds()
}

func (a *App) ListProxies() ([]tunneling.ProxyInfo, error) {
	return a.Tunneling.List(), nil
}

// ---- Shells ----

func (a *App) StartShell(sessionID, path string, enablePTY bool, rows, cols uint32) (*shells.ShellInfo, error) {
	return a.Shells.StartShell(sessionID, path, enablePTY, rows, cols)
}

func (a *App) WriteShell(id, data string) error {
	return a.Shells.WriteShell(id, data)
}

func (a *App) InterruptShell(id string) (bool, error) {
	return a.Shells.InterruptShell(id)
}

func (a *App) GetShellOutput(id string) (string, error) {
	return a.Shells.GetShellOutput(id)
}

func (a *App) ResizeShell(id string, rows, cols uint32) error {
	return a.Shells.ResizeShell(id, rows, cols)
}

func (a *App) CloseShell(id string) error {
	return a.Shells.CloseShell(id)
}

// ---- Server info ----

type ServerInfo struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Operator string `json:"operator"`
	CA       string `json:"ca"`
}

// GetServerInfo returns the teamserver we're currently connected to, so the UI
// can prefill listener bind hosts / C2 URLs / etc with something useful instead
// of forcing the operator to look it up.
func (a *App) GetServerInfo() ServerInfo {
	if a.RPC == nil || a.RPC.Config == nil {
		return ServerInfo{}
	}
	return ServerInfo{
		Host:     a.RPC.Config.LHost,
		Port:     a.RPC.Config.LPort,
		Operator: a.RPC.Config.Operator,
		CA:       a.RPC.Config.CACertificate,
	}
}

// ---- Implants / Builds / Profiles ----

func (a *App) GenerateImplant(goos, goarch, format, c2url, name string, isBeacon bool, beaconInterval int64) (string, error) {
	return a.Implants.Generate(goos, goarch, format, c2url, name, isBeacon, beaconInterval)
}

// GenerateImplantAdvanced builds an implant from the full option set exposed by
// the new generate modal. Wraps the same Sliver Generate RPC as the legacy path.
func (a *App) GenerateImplantAdvanced(req implants.GenerateRequest) (string, error) {
	return a.Implants.GenerateAdvanced(req)
}

func (a *App) SaveProfileAdvanced(req implants.GenerateRequest) error {
	return a.Implants.SaveProfileAdvanced(req)
}

func (a *App) SaveProfile(name, goos, goarch, format, c2url string, isBeacon bool, beaconInterval int64) error {
	return a.Implants.SaveProfile(name, goos, goarch, format, c2url, isBeacon, beaconInterval)
}

func (a *App) GenerateImplantFromProfile(profileConfigID, name string, format int) (string, error) {
	return a.Implants.GenerateFromProfile(profileConfigID, name, format)
}

func (a *App) GetImplantBuilds() (*clientpb.ImplantBuilds, error) {
	return a.Implants.GetImplantBuilds()
}

func (a *App) DeleteImplantBuild(name string) error {
	return a.Implants.DeleteImplantBuild(name)
}

func (a *App) GetProfiles() (*clientpb.ImplantProfiles, error) {
	return a.Implants.GetProfiles()
}

func (a *App) DeleteProfile(name string) error {
	return a.Implants.DeleteProfile(name)
}

func (a *App) RegenerateImplant(name string) (string, error) {
	return a.Implants.Regenerate(name)
}

func (a *App) GenerateStage(req staging.GenerateStageRequest) (string, error) {
	return a.Staging.GenerateStage(req)
}

func (a *App) StageImplantBuilds(builds []string) error {
	return a.Staging.StageImplantBuilds(builds)
}

// ---- Registry ----

func (a *App) ListRegistrySubKeys(sessionID, hive, path string) (*sliverpb.RegistrySubKeyList, error) {
	return a.Registry.ListSubKeys(sessionID, hive, path)
}

func (a *App) ListRegistryValues(sessionID, hive, path string) (*sliverpb.RegistryValuesList, error) {
	return a.Registry.ListValues(sessionID, hive, path)
}

func (a *App) ReadRegistryValue(sessionID, hive, path, key string) (*registry.Value, error) {
	return a.Registry.ReadValue(sessionID, hive, path, key)
}

func (a *App) WriteRegistryValue(sessionID, hive, path, key, valueType, value string) error {
	err := a.Registry.WriteValue(sessionID, hive, path, key, valueType, value)
	if err == nil {
		runtime.EventsEmit(a.ctx, "registry-updated", nil)
	}
	return err
}

func (a *App) CreateRegistryKey(sessionID, hive, path, key string) error {
	err := a.Registry.CreateKey(sessionID, hive, path, key)
	if err == nil {
		runtime.EventsEmit(a.ctx, "registry-updated", nil)
	}
	return err
}

func (a *App) DeleteRegistryEntry(sessionID, hive, path, key string) error {
	err := a.Registry.DeleteEntry(sessionID, hive, path, key)
	if err == nil {
		runtime.EventsEmit(a.ctx, "registry-updated", nil)
	}
	return err
}

// ---- Listeners / Jobs ----

func (a *App) GetJobs() (*clientpb.Jobs, error) {
	return a.Listeners.GetJobs()
}

func (a *App) KillJob(id uint32) error {
	return a.Listeners.KillJob(id)
}

func (a *App) StartListener(protocol, host string, port uint32, domains string) error {
	return a.Listeners.StartListener(protocol, host, port, domains)
}

func (a *App) StartTCPStagerListener(req staging.TCPListenerRequest) (*clientpb.StagerListener, error) {
	return a.Staging.StartTCPStagerListener(req)
}

// ---- Loot / Credentials ----

func (a *App) GetLoot() (*clientpb.AllLoot, error) {
	return a.Loot.GetLoot()
}

func (a *App) GetHosts() (*clientpb.AllHosts, error) {
	return a.Hosts.List()
}

func (a *App) GetHost(hostUUID string) (*clientpb.Host, error) {
	return a.Hosts.Get(hostUUID)
}

func (a *App) RemoveHost(hostUUID string) error {
	return a.Hosts.Remove(hostUUID)
}

func (a *App) RemoveHostIOC(iocID string) error {
	return a.Hosts.RemoveIOC(iocID)
}

func (a *App) DownloadLoot(lootID string) (string, error) {
	return a.Loot.DownloadLoot(lootID)
}

func (a *App) RemoveLoot(id string) error {
	return a.Loot.RemoveLoot(id)
}

func (a *App) GetCredentials() (*clientpb.Credentials, error) {
	return a.Loot.GetCredentials()
}

func (a *App) AddCredential(username, plaintext, hash, collection string) error {
	return a.Loot.AddCredential(username, plaintext, hash, collection)
}

func (a *App) RemoveCredential(id string) error {
	return a.Loot.RemoveCredential(id)
}

func (a *App) GetScreenshotData(lootID string) (string, error) {
	return a.Loot.GetScreenshotData(lootID)
}

func (a *App) GetAgentNotes() (map[string]string, error) {
	all := a.Comments.GetAllComments()
	notes := make(map[string]string)
	for key, list := range all {
		if strings.HasPrefix(key, "agent:") && len(list) > 0 {
			agentID := strings.TrimPrefix(key, "agent:")
			notes[agentID] = list[len(list)-1].Text
		}
	}
	return notes, nil
}

func (a *App) SaveAgentNote(agentID, text string) error {
	_, err := a.Comments.SetNote("agent", agentID, "Operator", text)
	if err != nil {
		return err
	}
	runtime.EventsEmit(a.ctx, "comments-updated", nil)
	runtime.EventsEmit(a.ctx, "agent-notes-updated", agentID)
	return nil
}

// ---- Server / Certificates / Websites ----

func (a *App) GetCertificates() (*clientpb.CertificateInfo, error) {
	return a.Server.GetCertificates()
}

func (a *App) GetWebsites() (*clientpb.Websites, error) {
	return a.Server.GetWebsites()
}

func (a *App) GetAliases() (interface{}, error) {
	return a.Server.GetAliases()
}

// ---- Automation ----

func (a *App) ListAutomationRules() ([]automation.AutomationRule, error) {
	return a.Automation.ListRules()
}

func (a *App) SaveAutomationRule(rule automation.AutomationRule) (automation.AutomationRule, error) {
	return a.Automation.SaveRule(rule)
}

func (a *App) DeleteAutomationRule(id string) error {
	return a.Automation.DeleteRule(id)
}

func (a *App) SetAutomationRuleEnabled(id string, enabled bool) error {
	return a.Automation.SetRuleEnabled(id, enabled)
}

func (a *App) RunAutomationRule(id, targetID string) error {
	return a.Automation.RunRule(id, targetID)
}

func (a *App) GetAutomationHistory() ([]automation.AutomationRun, error) {
	return a.Automation.GetHistory()
}

func (a *App) ClearAutomationHistory() error {
	return a.Automation.ClearHistory()
}

func (a *App) ExportAutomationRules() (string, error) {
	raw, err := a.Automation.ExportRules()
	if err != nil {
		return "", err
	}
	path, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "Export Automation Rules",
		DefaultFilename: fmt.Sprintf("sliver-automation-%s.json", time.Now().Format("2006-01-02")),
		Filters: []runtime.FileFilter{{
			DisplayName: "JSON files (*.json)",
			Pattern:     "*.json",
		}},
	})
	if err != nil || path == "" {
		return path, err
	}
	return path, os.WriteFile(path, []byte(raw), 0o600)
}

func (a *App) ImportAutomationRules(payload string) (automation.ImportResult, error) {
	return a.Automation.ImportRules(payload)
}

func (a *App) GetStarterAutomationRules() ([]automation.AutomationRule, error) {
	return a.Automation.StarterRules()
}

func (a *App) ImportStarterAutomationRule(id string) (automation.AutomationRule, error) {
	return a.Automation.ImportStarterRule(id)
}

// ---- Network Discovery ----

func (a *App) GetNetworkDiscoveries() []discovery.NetworkDiscovery {
	return a.Discovery.GetNetworkDiscoveries()
}

func (a *App) DiscoverNetwork(agentID, method, cidr string) ([]discovery.NetworkDiscovery, error) {
	return a.Discovery.DiscoverNetwork(agentID, method, cidr)
}

func (a *App) ClearNetworkDiscoveries(agentID string) {
	a.Discovery.ClearNetworkDiscoveries(agentID)
}

func (a *App) RemoveNetworkDiscoveries(agentID string, ips []string) {
	a.Discovery.RemoveNetworkDiscoveries(agentID, ips)
}

func (a *App) GetSystemTheme() string {
	return theme.Detect(a.ctx)
}

func (a *App) Show() {
	runtime.WindowShow(a.ctx)
}

// ---- Beacon lifecycle ----

func (a *App) GetBeacon(beaconID string) (*clientpb.Beacon, error) {
	return a.Beacons.GetBeacon(beaconID)
}

func (a *App) OpenBeaconSession(req beacons.OpenSessionRequest) (*sliverpb.OpenSession, error) {
	return a.Beacons.OpenSession(req)
}

func (a *App) CloseBeaconSession(beaconID, tunnelID string) error {
	return a.Beacons.CloseSession(beaconID, tunnelID)
}

func (a *App) UpdateBeaconIntegrity(beaconID, integrity string) error {
	return a.Beacons.UpdateBeaconIntegrity(beaconID, integrity)
}

// ---- Reconfigure ----

func (a *App) ReconfigureAgent(req agents.ReconfigureRequest) error {
	return a.Agents.Reconfigure(req)
}

// ---- Credentials extended ----

func (a *App) UpdateCredential(req loot.UpdateCredentialRequest) error {
	return a.Loot.UpdateCredential(req)
}

func (a *App) GetCredentialByID(id string) (*clientpb.Credential, error) {
	return a.Loot.GetCredentialByID(id)
}

func (a *App) GetCredentialsByHashType(hashType int32) (*clientpb.Credentials, error) {
	return a.Loot.GetCredentialsByHashType(hashType)
}

func (a *App) GetPlaintextCredentialsByHashType(hashType int32) (*clientpb.Credentials, error) {
	return a.Loot.GetPlaintextCredentialsByHashType(hashType)
}

func (a *App) SniffCredentialHashType(hash string) (*clientpb.Credential, error) {
	return a.Loot.SniffCredentialHashType(hash)
}

// ---- Server misc ----

func (a *App) GetCertificateAuthorityInfo() (*clientpb.CertificateAuthorityInfo, error) {
	return a.Server.CertificateAuthorityInfo()
}

func (a *App) GetCompiler() (*clientpb.Compiler, error) {
	return a.Server.Compiler()
}

func (a *App) GetCanaries() (*clientpb.Canaries, error) {
	return a.Server.Canaries()
}

func (a *App) RestartJobs(jobIDs []uint32) error {
	return a.Server.RestartJobs(jobIDs)
}

func (a *App) GenerateSpoofMetadata(req implants.SpoofMetadataRequest) error {
	return a.Implants.GenerateSpoofMetadata(req)
}

// PrimeSpoofMetadataFromPath reads a reference PE from the operator's disk
// and hands the bytes to GenerateSpoofMetadata — cheaper than round-tripping
// through wails' JSON bridge, which chokes on multi-megabyte binaries.
func (a *App) PrimeSpoofMetadataFromPath(implantName, sourcePath string) error {
	return a.Implants.PrimeSpoofMetadataFromPath(implantName, sourcePath)
}

func (a *App) LogClient(line string) {
	if a.ClientLog != nil {
		a.ClientLog.Log(line)
	}
}

// ---- Websites ----

func (a *App) GetWebsite(name string) (*clientpb.Website, error) {
	return a.Websites.GetWebsite(name)
}

func (a *App) RemoveWebsite(name string) error {
	return a.Websites.RemoveWebsite(name)
}

func (a *App) AddWebsiteContent(req websites.AddContentRequest) error {
	return a.Websites.AddContent(req)
}

func (a *App) UpdateWebsiteContent(req websites.AddContentRequest) error {
	return a.Websites.UpdateContent(req)
}

func (a *App) RemoveWebsiteContent(name string, paths []string) error {
	return a.Websites.RemoveContent(name, paths)
}

func (a *App) GetTrafficEncoderMap() (*clientpb.TrafficEncoderMap, error) {
	return a.Server.TrafficEncoderMap()
}

func (a *App) AddTrafficEncoder(localPath string, skipTests bool) (*clientpb.TrafficEncoderTests, error) {
	return a.Server.AddTrafficEncoder(localPath, skipTests)
}

func (a *App) RemoveTrafficEncoder(name string) error {
	return a.Server.RemoveTrafficEncoder(name)
}

func (a *App) GetShellcodeEncoderMap() (*clientpb.ShellcodeEncoderMap, error) {
	return uishellcode.EncoderMap(a.RPC)
}

func (a *App) GenerateShellcodeRDI(req uishellcode.RDIRequest) (string, error) {
	return uishellcode.GenerateRDI(a.ctx, a.RPC, req)
}

func (a *App) EncodeShellcode(req uishellcode.EncodeRequest) (string, error) {
	return uishellcode.Encode(a.ctx, a.RPC, req)
}

func (a *App) GetHTTPC2Profiles() (*clientpb.HTTPC2Configs, error) {
	return a.Server.HTTPC2Profiles()
}

func (a *App) GetHTTPC2ProfileByName(name string) (*clientpb.HTTPC2Config, error) {
	return a.Server.HTTPC2ProfileByName(name)
}

// ---- Monitoring Providers ----

func (a *App) MonitorStart() (*commonpb.Response, error) {
	return a.Monitor.MonitorStart()
}

func (a *App) MonitorStop() error {
	return a.Monitor.MonitorStop()
}

func (a *App) GetMonitorProviders() (*clientpb.MonitoringProviders, error) {
	return a.Monitor.ListConfig()
}

func (a *App) AddMonitorProvider(id, providerType, apiKey, apiPassword string) (*commonpb.Response, error) {
	return a.Monitor.AddConfig(&clientpb.MonitoringProvider{
		ID:          id,
		Type:        providerType,
		APIKey:      apiKey,
		APIPassword: apiPassword,
	})
}

func (a *App) RemoveMonitorProvider(id string) (*commonpb.Response, error) {
	return a.Monitor.DelConfig(&clientpb.MonitoringProvider{
		ID: id,
	})
}

// ---- WireGuard ----

func (a *App) StartWGListener(host string, port, nport, keyPort uint32, tunIP string) (*clientpb.ListenerJob, error) {
	return a.WireGuard.StartListener(host, port, nport, keyPort, tunIP)
}

func (a *App) GenerateWGClientConfig() (*clientpb.WGClientConfig, error) {
	return a.WireGuard.GenerateClientConfig()
}

func (a *App) GenerateUniqueWGIP() (*clientpb.UniqueWGIP, error) {
	return a.WireGuard.GenerateUniqueIP()
}

func (a *App) WGStartSocks(sessionID string, port int32) (*sliverpb.WGSocks, error) {
	return a.WireGuard.StartSocks(sessionID, port)
}

func (a *App) WGStopSocks(sessionID string, id int32) (*sliverpb.WGSocks, error) {
	return a.WireGuard.StopSocks(sessionID, id)
}

func (a *App) WGListSocksServers(sessionID string) (*sliverpb.WGSocksServers, error) {
	return a.WireGuard.ListSocksServers(sessionID)
}

func (a *App) WGStartPortForward(sessionID string, localPort int32, remoteAddr string) (*sliverpb.WGPortForward, error) {
	return a.WireGuard.StartPortForward(sessionID, localPort, remoteAddr)
}

func (a *App) WGStopPortForward(sessionID string, id int32) (*sliverpb.WGPortForward, error) {
	return a.WireGuard.StopPortForward(sessionID, id)
}

func (a *App) WGListForwarders(sessionID string) (*sliverpb.WGTCPForwarders, error) {
	return a.WireGuard.ListForwarders(sessionID)
}

// ---- Extensions ----

func (a *App) RegisterExtension(sessionID, name string, data []byte, os, init string) (*sliverpb.RegisterExtension, error) {
	return a.Extensions.RegisterExtension(sessionID, name, data, os, init)
}

func (a *App) RegisterExtensionFromPath(sessionID, name, localPath, targetOS, init string) (*sliverpb.RegisterExtension, error) {
	return a.Extensions.RegisterExtensionFromPath(sessionID, name, localPath, targetOS, init)
}

func (a *App) ListExtensions(sessionID string) (*sliverpb.ListExtensions, error) {
	return a.Extensions.ListExtensions(sessionID)
}

func (a *App) CallExtension(sessionID, name, export string, args []byte, serverStore bool) (*sliverpb.CallExtension, error) {
	return a.Extensions.CallExtension(sessionID, name, export, args, serverStore)
}

func (a *App) RegisterWasmExtension(sessionID, name string, wasmGz []byte) (*sliverpb.RegisterWasmExtension, error) {
	return a.Extensions.RegisterWasmExtension(sessionID, name, wasmGz)
}

func (a *App) RegisterWasmExtensionFromPath(sessionID, name, localPath string) (*sliverpb.RegisterWasmExtension, error) {
	return a.Extensions.RegisterWasmExtensionFromPath(sessionID, name, localPath)
}

func (a *App) ListWasmExtensions(sessionID string) (*sliverpb.ListWasmExtensions, error) {
	return a.Extensions.ListWasmExtensions(sessionID)
}

func (a *App) ExecWasmExtension(sessionID, name string, args []string, interactive bool) (*sliverpb.ExecWasmExtension, error) {
	return a.Extensions.ExecWasmExtension(sessionID, name, args, interactive)
}

// ---- Memfiles ----

func (a *App) MemfilesList(sessionID string) (*sliverpb.Ls, error) {
	return a.Memfiles.List(sessionID)
}

func (a *App) MemfilesAdd(sessionID string) (*sliverpb.MemfilesAdd, error) {
	return a.Memfiles.Add(sessionID)
}

func (a *App) MemfilesRemove(sessionID string, fd int64) (*sliverpb.MemfilesRm, error) {
	return a.Memfiles.Remove(sessionID, fd)
}

// ---- Pivot RPCs direct ----

func (a *App) PivotStartListener(sessionID string, pivotType int32, bindAddr string) (*sliverpb.PivotListener, error) {
	return a.Pivots.StartListener(sessionID, sliverpb.PivotType(pivotType), bindAddr)
}

func (a *App) PivotStopListener(sessionID string, id uint32) error {
	return a.Pivots.StopListener(sessionID, id)
}

// ---- Registry Read Hive ----

func (a *App) RegistryReadHive(sessionID, rootHive, requestedHive string) (*sliverpb.RegistryReadHive, error) {
	return a.Registry.ReadHive(sessionID, rootHive, requestedHive)
}

// ---- Password Cracking ----

func (a *App) Crackstations() (*clientpb.Crackstations, error) {
	return a.Crack.Crackstations()
}

func (a *App) CrackSubmitJob(attackMode int32, hashType int32, hashes []string, rulesFile []byte) (*clientpb.CrackResponse, error) {
	return a.Crack.SubmitJob(&clientpb.CrackCommand{
		AttackMode: clientpb.CrackAttackMode(attackMode),
		HashType:   clientpb.HashType(hashType),
		Hashes:     hashes,
		RulesFile:  rulesFile,
	})
}

func (a *App) CrackTaskByID(id, hostUUID string) (*clientpb.CrackTask, error) {
	return a.Crack.TaskByID(&clientpb.CrackTask{ID: id, HostUUID: hostUUID})
}

func (a *App) CrackTaskCancel(id, hostUUID string) error {
	task, err := a.Crack.TaskByID(&clientpb.CrackTask{ID: id})
	if err != nil {
		return err
	}
	if hostUUID != "" {
		task.HostUUID = hostUUID
	}
	task.CompletedAt = time.Now().Unix()
	task.Err = "cancelled by GUI operator"
	return a.Crack.TaskUpdate(task)
}

func (a *App) CrackFilesList() (*clientpb.CrackFiles, error) {
	return a.Crack.FilesList(&clientpb.CrackFile{})
}

func (a *App) CrackFileCreate(name string, fileType int32, isCompressed bool, uncompressedSize int64, maxFileSize, chunkSize int64) (*clientpb.CrackFile, error) {
	return a.Crack.FileCreate(&clientpb.CrackFile{
		Name:             name,
		Type:             clientpb.CrackFileType(fileType),
		IsCompressed:     isCompressed,
		UncompressedSize: uncompressedSize,
		MaxFileSize:      maxFileSize,
		ChunkSize:        chunkSize,
	})
}

func (a *App) CrackFileChunkUpload(crackFileID string, n uint32, data []byte) error {
	return a.Crack.FileChunkUpload(&clientpb.CrackFileChunk{
		CrackFileID: crackFileID,
		N:           n,
		Data:        data,
	})
}

func (a *App) CrackFileChunkDownload(crackFileID string, n uint32) (*clientpb.CrackFileChunk, error) {
	return a.Crack.FileChunkDownload(&clientpb.CrackFileChunk{
		CrackFileID: crackFileID,
		N:           n,
	})
}

func (a *App) CrackFileComplete(fileID string) error {
	return a.Crack.FileComplete(&clientpb.CrackFile{ID: fileID})
}

func (a *App) CrackFileDelete(fileID string) error {
	return a.Crack.FileDelete(&clientpb.CrackFile{ID: fileID})
}

func (a *App) CrackFileUploadFromPath(localPath string, fileType int32) (*clientpb.CrackFile, error) {
	return a.Crack.UploadFromPath(localPath, clientpb.CrackFileType(fileType))
}

// ---- Distributed Builders ----

func (a *App) GetBuilders() (*clientpb.Builders, error) {
	return a.Builders.Builders()
}

func (a *App) GenerateExternalBuild(configJSON, builderName, name string) (*clientpb.ExternalImplantConfig, error) {
	config, err := parseExternalBuildConfig(configJSON)
	if err != nil {
		return nil, err
	}
	return a.Builders.GenerateExternal(&clientpb.ExternalGenerateReq{
		Config:      config,
		BuilderName: builderName,
		Name:        name,
	})
}

func parseExternalBuildConfig(configJSON string) (*clientpb.ImplantConfig, error) {
	var req implants.GenerateRequest
	if err := json.Unmarshal([]byte(configJSON), &req); err == nil && isGenerateRequest(req) {
		return implants.ConfigFromGenerateRequest(req)
	}
	var config clientpb.ImplantConfig
	if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
		return nil, fmt.Errorf("invalid implant config JSON: %w", err)
	}
	return &config, nil
}

func isGenerateRequest(req implants.GenerateRequest) bool {
	return req.GOOS != "" || req.GOARCH != "" || req.Format != "" || len(req.C2URLs) > 0
}

func (a *App) GetExternalBuildConfig(buildID string) (*clientpb.ExternalImplantConfig, error) {
	return a.Builders.GetExternalBuildConfig(&clientpb.ImplantBuild{ID: buildID})
}

func (a *App) SaveExternalBuild(name, implantBuildID string, fileData []byte) error {
	return a.Builders.SaveExternalBuild(&clientpb.ExternalImplantBinary{
		Name:           name,
		ImplantBuildID: implantBuildID,
		File:           &commonpb.File{Data: fileData},
	})
}

func (a *App) BuilderTrigger(eventType string, data []byte) error {
	return a.Builders.Trigger(&clientpb.Event{EventType: eventType, Data: data})
}

func (a *App) CrackstationTrigger(eventType string, data []byte) error {
	return a.Crack.Trigger(&clientpb.Event{EventType: eventType, Data: data})
}

func (a *App) SaveHTTPC2ProfileJSON(profileJSON string, overwrite bool) error {
	var config clientpb.HTTPC2Config
	if err := json.Unmarshal([]byte(profileJSON), &config); err != nil {
		return fmt.Errorf("invalid HTTP C2 profile JSON: %w", err)
	}
	return a.Server.SaveHTTPC2Profile(&config, overwrite)
}

// ---- Entity tags & colors ----

func (a *App) GetEntityTags(entityType, entityID string) []string {
	return a.Tags.GetEntityTags(entityType, entityID)
}

func (a *App) SetEntityTags(entityType, entityID string, tagList []string) error {
	if err := a.Tags.SetEntityTags(entityType, entityID, tagList); err != nil {
		return err
	}
	key := entityType + ":" + entityID
	runtime.EventsEmit(a.ctx, "entity-tags-updated", key)
	if strings.EqualFold(strings.TrimSpace(entityType), "agent") {
		runtime.EventsEmit(a.ctx, "agent-tags-updated", entityID)
	}
	return nil
}

func (a *App) GetAllEntityTags() map[string][]string {
	return a.Tags.GetAllEntityTags()
}

func (a *App) GetEntityColor(entityType, entityID string) string {
	return a.Tags.GetEntityColor(entityType, entityID)
}

func (a *App) SetEntityColor(entityType, entityID string, color string) error {
	if err := a.Tags.SetEntityColor(entityType, entityID, color); err != nil {
		return err
	}
	key := entityType + ":" + entityID
	runtime.EventsEmit(a.ctx, "entity-colors-updated", key)
	if strings.EqualFold(strings.TrimSpace(entityType), "agent") {
		runtime.EventsEmit(a.ctx, "agent-colors-updated", entityID)
	}
	return nil
}

func (a *App) GetAllEntityColors() map[string]string {
	return a.Tags.GetAllEntityColors()
}

func (a *App) GetAgentTags(agentID string) []string {
	return a.Tags.GetAgentTags(agentID)
}

func (a *App) SetAgentTags(agentID string, tagList []string) error {
	if err := a.Tags.SetAgentTags(agentID, tagList); err != nil {
		return err
	}
	runtime.EventsEmit(a.ctx, "entity-tags-updated", "agent:"+agentID)
	runtime.EventsEmit(a.ctx, "agent-tags-updated", agentID)
	return nil
}

func (a *App) GetAllAgentTags() map[string][]string {
	return a.Tags.GetAllTags()
}

func (a *App) ListKnownTags() []string {
	return a.Tags.KnownTags()
}

func (a *App) GetAllAgentColors() map[string]string {
	return a.Tags.GetAllColors()
}

func (a *App) SetAgentColor(agentID string, color string) error {
	if err := a.Tags.SetAgentColor(agentID, color); err != nil {
		return err
	}
	runtime.EventsEmit(a.ctx, "entity-colors-updated", "agent:"+agentID)
	runtime.EventsEmit(a.ctx, "agent-colors-updated", agentID)
	return nil
}

// ---- Universal Entity Comments ----

func (a *App) GetEntityComments(entityType, entityID string) []comments.Comment {
	return a.Comments.GetComments(entityType, entityID)
}

func (a *App) GetAllComments() map[string][]comments.Comment {
	return a.Comments.GetAllComments()
}

func (a *App) AddEntityComment(entityType, entityID, author, text string) (comments.Comment, error) {
	c, err := a.Comments.AddComment(entityType, entityID, author, text)
	if err != nil {
		return comments.Comment{}, err
	}
	runtime.EventsEmit(a.ctx, "comments-updated", entityType+":"+entityID)
	return c, nil
}

func (a *App) DeleteEntityComment(commentID string) error {
	if err := a.Comments.DeleteComment(commentID); err != nil {
		return err
	}
	runtime.EventsEmit(a.ctx, "comments-updated", "")
	return nil
}

// ---- Case files ----

func (a *App) ListCases() []*casefile.Record {
	return a.Cases.List()
}

func (a *App) GetCase(id string) *casefile.Record {
	return a.Cases.Get(id)
}

func (a *App) CreateCase(name, description string) (*casefile.Record, error) {
	c, err := a.Cases.Create(name, description)
	if err == nil {
		runtime.EventsEmit(a.ctx, "case-updated", c.ID)
	}
	return c, err
}

func (a *App) UpdateCase(id, name, description, notes string) error {
	if err := a.Cases.Update(id, name, description, notes); err != nil {
		return err
	}
	runtime.EventsEmit(a.ctx, "case-updated", id)
	return nil
}

func (a *App) DeleteCase(id string) error {
	if err := a.Cases.Delete(id); err != nil {
		return err
	}
	runtime.EventsEmit(a.ctx, "case-updated", id)
	return nil
}

// AddToCase / RemoveFromCase — collection ∈ {"agent","loot","cred","host","canary"}.
func (a *App) AddToCase(caseID, collection, itemID string) error {
	if err := a.Cases.Add(caseID, casefile.Collection(collection), itemID); err != nil {
		return err
	}
	runtime.EventsEmit(a.ctx, "case-updated", caseID)
	return nil
}

func (a *App) RemoveFromCase(caseID, collection, itemID string) error {
	if err := a.Cases.Remove(caseID, casefile.Collection(collection), itemID); err != nil {
		return err
	}
	runtime.EventsEmit(a.ctx, "case-updated", caseID)
	return nil
}

// GenerateCaseReport renders a case as a Markdown document string.
func (a *App) GenerateCaseReport(caseID string) (string, error) {
	return a.Cases.GenerateMarkdown(caseID, casereport.NewReporter(a.Console, a.RPC))
}

func (a *App) ExportCaseReport(caseID string) (string, error) {
	c := a.Cases.Get(caseID)
	if c == nil {
		return "", fmt.Errorf("case %s not found", caseID)
	}
	md, err := a.GenerateCaseReport(caseID)
	if err != nil {
		return "", err
	}
	path, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "Export Case Report",
		DefaultFilename: casefile.ReportFilename(c.Name),
		Filters: []runtime.FileFilter{{
			DisplayName: "Markdown files (*.md)",
			Pattern:     "*.md",
		}},
	})
	if err != nil || path == "" {
		return path, err
	}
	return path, os.WriteFile(path, []byte(md), 0o600)
}

// ---- Health ----

func (a *App) HealthSnapshot() health.Snapshot {
	return a.Health.Snapshot()
}
