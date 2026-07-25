package gui

import (
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"

	"sliver-gui/internal/sliver/agents"
	"sliver-gui/internal/sliver/armory"
	"sliver-gui/internal/sliver/beacons"
	"sliver-gui/internal/sliver/pivots"
)

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

// ---- Beacons ----

func (a *App) GetBeaconTasks(beaconID string) (*clientpb.BeaconTasks, error) {
	return a.Beacons.GetBeaconTasks(beaconID)
}

func (a *App) GetBeaconTaskOutput(taskID string) (*beacons.TaskOutput, error) {
	return a.Beacons.GetBeaconTaskOutput(taskID)
}

func (a *App) CancelBeaconTask(taskID string) error {
	return a.Beacons.CancelBeaconTask(taskID)
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
