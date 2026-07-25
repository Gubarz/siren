package gui

import (
	"github.com/wailsapp/wails/v2/pkg/runtime"

	"sliver-gui/internal/sliver/discovery"
	"sliver-gui/internal/theme"
)

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
