package gui

import (
	"github.com/bishopfox/sliver/protobuf/clientpb"

	"siren/internal/sliver/implants"
	"siren/internal/sliver/staging"
)

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

func (a *App) GenerateSpoofMetadata(req implants.SpoofMetadataRequest) error {
	return a.Implants.GenerateSpoofMetadata(req)
}

// PrimeSpoofMetadataFromPath reads a reference PE from the operator's disk
// and hands the bytes to GenerateSpoofMetadata — cheaper than round-tripping
// through wails' JSON bridge, which chokes on multi-megabyte binaries.
func (a *App) PrimeSpoofMetadataFromPath(implantName, sourcePath string) error {
	return a.Implants.PrimeSpoofMetadataFromPath(implantName, sourcePath)
}
