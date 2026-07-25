package gui

import (
	"context"
	"log"
	"strings"
	"time"

	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"google.golang.org/protobuf/proto"

	"sliver-gui/internal/localstate/events"
	"sliver-gui/internal/wailsadapter"
)

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
			switch ev.EventType {
			case consts.BeaconRegisteredEvent, consts.BeaconTaskResultEvent:
				b := &clientpb.Beacon{}
				if proto.Unmarshal(ev.Data, b) == nil && b.ID != "" {
					payload["beaconID"] = b.ID
					payload["hostname"] = b.Hostname
					payload["username"] = b.Username
				}
			}
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
