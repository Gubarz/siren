package gui

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/wailsapp/wails/v3/pkg/application"
	wailsevents "github.com/wailsapp/wails/v3/pkg/events"
	"google.golang.org/protobuf/proto"

	"siren/internal/bus"
	"siren/internal/localstate/events"
)

func (a *App) ServiceStartup(ctx context.Context, _ application.ServiceOptions) error {
	a.ctx, a.cancel = context.WithCancel(ctx)

	a.Console.SetEmitter(a.bridge)
	a.Automation.SetEmitter(a.bridge)

	a.Implants.SetUI(a.bridge)
	a.Files.SetUI(a.bridge)
	a.Loot.SetUI(a.bridge)
	a.Shells.SetUI(a.bridge)
	a.Discovery.SetUI(a.bridge)
	a.Staging.SetUI(a.bridge)

	a.Automation.Start(ctx)
	a.CheckinPub.Start(ctx)

	if a.BloodHound != nil {
		a.BloodHound.ConnectIfConfigured(ctx)
		go a.BloodHound.StartSync(a.ctx, 5*time.Minute)
	}

	// v3 delivers file drops to the backend (only for elements marked with
	// data-file-drop-target); re-emit to the frontend with the v2 event shape.
	a.registerFileDropWindow(a.window)

	a.startBusSubscribers()

	go func() {
		time.Sleep(300 * time.Millisecond)
		a.window.Show()
	}()
	return nil
}

func (a *App) registerFileDropWindow(window *application.WebviewWindow) {
	window.OnWindowEvent(wailsevents.Common.WindowFilesDropped, func(event *application.WindowEvent) {
		files := event.Context().DroppedFiles()
		x, y := 0, 0
		if details := event.Context().DropTargetDetails(); details != nil {
			x, y = details.X, details.Y
		}
		window.EmitEvent("files-dropped", map[string]interface{}{"files": files, "x": x, "y": y})
	})
}

func (a *App) ServiceShutdown() error {
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
	if a.BloodHound != nil {
		a.BloodHound.Close()
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
	if a.Journal != nil {
		if err := a.Journal.Close(); err != nil {
			log.Printf("shutdown: close journal: %v", err)
		}
	}
	a.Events.Close()
	if a.cancel != nil {
		a.cancel()
	}
	return nil
}

// shouldPreserveLiveResourcesOnShutdown reports whether the app is running
// under the dev server (wails3 dev sets FRONTEND_DEVSERVER_URL), in which case
// live sessions survive the restart of the GUI process.
func (a *App) shouldPreserveLiveResourcesOnShutdown() bool {
	return os.Getenv("FRONTEND_DEVSERVER_URL") != ""
}

func (a *App) OpenFileDialog(title string) (string, error) {
	return a.bridge.OpenFileDialog(&application.OpenFileDialogOptions{
		Title: title,
	})
}

func (a *App) startEventStream() {
	connID := ""
	if a.RPC.Config != nil {
		connID = fmt.Sprintf("%s:%d", a.RPC.Config.LHost, a.RPC.Config.LPort)
	}
	a.RPC.StartEventStream(a.ctx, func(ev *clientpb.Event) {
		a.Bus.Publish(bus.Event{
			Type:         "sliver." + ev.EventType,
			Source:       "grpc-stream",
			ConnectionID: connID,
			Payload:      sliverEventPayload(ev),
		})
	})
}

// sliverEventPayload flattens a sliver event into a protobuf-free DTO map.
// It carries everything today's frontend emit, events store, and automation
// fan-out consume.
func sliverEventPayload(ev *clientpb.Event) map[string]interface{} {
	payload := map[string]interface{}{"type": ev.EventType}
	if ev.Session != nil {
		payload["sessionID"] = ev.Session.ID
		payload["name"] = ev.Session.Name
		payload["hostname"] = ev.Session.Hostname
		payload["username"] = ev.Session.Username
		payload["os"] = ev.Session.OS
		payload["arch"] = ev.Session.Arch
	}
	if ev.Job != nil {
		payload["job"] = ev.Job.Name
	}
	if len(ev.Data) > 0 {
		switch ev.EventType {
		case consts.BeaconRegisteredEvent, consts.BeaconTaskResultEvent:
			b := &clientpb.Beacon{}
			if proto.Unmarshal(ev.Data, b) == nil && b.ID != "" {
				payload["beaconID"] = b.ID
				payload["name"] = b.Name
				payload["hostname"] = b.Hostname
				payload["username"] = b.Username
				payload["os"] = b.OS
				payload["arch"] = b.Arch
			}
		}
		payload["data"] = string(ev.Data)
	}
	return payload
}

func (a *App) GetEventHistory(since int64, limit int) ([]events.StoredEvent, error) {
	return a.Events.Query(since, limit), nil
}

func (a *App) SetEventsAcknowledged(seqs []int64, acked bool) int {
	return a.Events.SetAcked(seqs, acked)
}
