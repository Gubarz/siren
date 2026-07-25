package gui

import (
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"

	"sliver-gui/internal/sliver/shells"
	"sliver-gui/internal/sliver/tunneling"
)

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

// ---- Pivot RPCs direct ----

func (a *App) PivotStartListener(sessionID string, pivotType int32, bindAddr string) (*sliverpb.PivotListener, error) {
	return a.Pivots.StartListener(sessionID, sliverpb.PivotType(pivotType), bindAddr)
}

func (a *App) PivotStopListener(sessionID string, id uint32) error {
	return a.Pivots.StopListener(sessionID, id)
}
