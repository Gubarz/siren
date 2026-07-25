package gui

import (
	"github.com/bishopfox/sliver/protobuf/sliverpb"
)

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
