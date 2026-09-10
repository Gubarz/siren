package gui

import (
	"testing"

	"siren/internal/bootstrap"
	"siren/internal/sliver/console"
	"siren/internal/sliver/rpc"
	"siren/internal/sliver/shells"
	"siren/internal/sliver/tunneling"
)

// newTeardownTestApp builds the services Disconnect touches without a Wails app
// or a teamserver. Tunnels, shells, and consoles are empty and the RPC client is
// disconnected, which is the state a freshly launched GUI is in.
func newTeardownTestApp() *App {
	rpcClient := rpc.NewClient()
	con := console.New(rpcClient)
	return &App{
		SharedStack: &bootstrap.SharedStack{RPC: rpcClient, Console: con},
		Tunneling:   tunneling.New(rpcClient),
		Shells:      shells.New(rpcClient, con),
	}
}

// Disconnecting before ever connecting is a normal thing to click, and teardown
// has to tolerate having nothing to close.
func TestDisconnectBeforeConnectingIsSafeAndRepeatable(t *testing.T) {
	a := newTeardownTestApp()

	for i := range 2 {
		if err := a.Disconnect(); err != nil {
			t.Fatalf("Disconnect() call %d = %v, want nil", i+1, err)
		}
	}
	if proxies := a.Tunneling.List(); len(proxies) != 0 {
		t.Fatalf("Tunneling.List() = %v, want empty after Disconnect", proxies)
	}
}

// Teardown must tolerate an App whose optional services were never built, which
// is the state before NewApp finishes wiring everything up.
func TestDisconnectWithNoLiveServicesIsSafe(t *testing.T) {
	a := &App{SharedStack: &bootstrap.SharedStack{RPC: rpc.NewClient()}}

	if err := a.Disconnect(); err != nil {
		t.Fatalf("Disconnect() = %v, want nil", err)
	}
}

// An unknown profile fails during lookup without dialling, so Connect's teardown
// hook never runs; the App must still be usable afterwards.
func TestConnectWithUnknownProfileLeavesAppUsable(t *testing.T) {
	a := newTeardownTestApp()

	if err := a.Connect("profile-that-does-not-exist"); err == nil {
		t.Fatal("Connect() with an unknown profile = nil, want an error")
	}
	if err := a.Disconnect(); err != nil {
		t.Fatalf("Disconnect() after a failed Connect = %v, want nil", err)
	}
}
