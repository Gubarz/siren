package gui

import (
	"log"

	"sliver-gui/internal/sliver/rpc"
)

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
	a.connectionMu.Lock()
	defer a.connectionMu.Unlock()

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
	a.connectionMu.Lock()
	defer a.connectionMu.Unlock()

	// Wails dev mode can host the desktop webview and one or more browser
	// clients at the same time. They share this App instance, so a second UI
	// connecting to the active profile must not replace the live connection.
	if a.RPC.IsConnectedTo(profileName) {
		return nil
	}
	if a.ClientLog != nil {
		a.ClientLog.Close()
	}
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
