package gui

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/wailsapp/wails/v3/pkg/application"

	"siren/internal/sliver/rpc"
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

// ExportClientConfig writes the profile to a path the operator picks and
// returns it. The profile carries the client's private key, so it is written
// here rather than handed to the webview to save.
func (a *App) ExportClientConfig(name string) (string, error) {
	raw, err := a.RPC.ExportClientConfig(name)
	if err != nil {
		return "", err
	}
	localPath, err := a.bridge.SaveFileDialog(&application.SaveFileDialogOptions{
		Title:    "Export Sliver profile",
		Filename: profileFilename(name),
	})
	if err != nil {
		return "", fmt.Errorf("dialog error: %w", err)
	}
	if localPath == "" {
		return "", nil
	}
	if err := os.WriteFile(localPath, []byte(raw), 0o600); err != nil {
		return "", err
	}
	return localPath, nil
}

// profileFilename turns a profile's display name into a usable file name.
func profileFilename(name string) string {
	safe := strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			return r
		case r == '.', r == '@', r == '-', r == '_':
			return r
		default:
			return '_'
		}
	}, strings.TrimSpace(name))
	if safe == "" {
		safe = "sliver-profile"
	}
	return safe + ".cfg"
}

func (a *App) DeleteClientConfig(name string) error {
	return a.RPC.DeleteClientConfig(name)
}

func (a *App) Disconnect() error {
	a.connectionMu.Lock()
	defer a.connectionMu.Unlock()

	a.teardownConnectionResources()
	a.RPC.Disconnect()
	return nil
}

// teardownConnectionResources closes everything bound to the connection being
// retired, and is the superset of closeLiveResources used when the connection is
// changing rather than the app shutting down: the in-process console also has to
// be reset because it caches the previous server's targets.
//
// Each console subprocess is its own sliver client holding an authenticated
// session, so leaving any of this up would keep talking to a teamserver the
// operator has moved on from.
func (a *App) teardownConnectionResources() {
	a.closeLiveResources()
	if a.Console != nil && !a.Console.TryResetConsole() {
		log.Printf("connection: console is busy; skipping console reset")
	}
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
	// Teardown runs as Connect's hook, so it happens once the replacement
	// connection is up but before the old one is retired. Live tunnels, shells,
	// and console subprocesses write into the outgoing gRPC stream, and sliver's
	// server has historically panicked when that stream drops mid-tunnel. Running
	// it here rather than up front means a failed profile switch leaves them
	// running against the current server.
	if err := a.RPC.Connect(profileName, a.teardownConnectionResources); err != nil {
		return err
	}
	if a.RPC.Config() != nil {
		a.Automation.SetServer(a.RPC.Config().LHost, uint32(a.RPC.Config().LPort))
		a.Tags.SetServer(a.RPC.Config().LHost, uint32(a.RPC.Config().LPort))
		a.Comments.SetServer(a.RPC.Config().LHost, uint32(a.RPC.Config().LPort))
		a.KnownAgents.SetServer(a.RPC.Config().LHost, uint32(a.RPC.Config().LPort))
	}
	if a.ClientLog != nil {
		if err := a.ClientLog.Start(a.ctx); err != nil {
			log.Printf("connect: client log stream failed: %v", err)
		}
	}
	a.startEventStream()
	return nil
}
