package gui

import (
	"encoding/json"
	"fmt"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"

	"siren/internal/sliver/implants"
)

// ---- Distributed Builders ----

func (a *App) GetBuilders() (*clientpb.Builders, error) {
	return a.Builders.Builders()
}

func (a *App) GenerateExternalBuild(configJSON, builderName, name string) (*clientpb.ExternalImplantConfig, error) {
	config, err := parseExternalBuildConfig(configJSON)
	if err != nil {
		return nil, err
	}
	return a.Builders.GenerateExternal(&clientpb.ExternalGenerateReq{
		Config:      config,
		BuilderName: builderName,
		Name:        name,
	})
}

func parseExternalBuildConfig(configJSON string) (*clientpb.ImplantConfig, error) {
	var req implants.GenerateRequest
	if err := json.Unmarshal([]byte(configJSON), &req); err == nil && isGenerateRequest(req) {
		return implants.ConfigFromGenerateRequest(req)
	}
	var config clientpb.ImplantConfig
	if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
		return nil, fmt.Errorf("invalid implant config JSON: %w", err)
	}
	return &config, nil
}

func isGenerateRequest(req implants.GenerateRequest) bool {
	return req.GOOS != "" || req.GOARCH != "" || req.Format != "" || len(req.C2URLs) > 0
}

func (a *App) GetExternalBuildConfig(buildID string) (*clientpb.ExternalImplantConfig, error) {
	return a.Builders.GetExternalBuildConfig(&clientpb.ImplantBuild{ID: buildID})
}

func (a *App) SaveExternalBuild(name, implantBuildID string, fileData []byte) error {
	return a.Builders.SaveExternalBuild(&clientpb.ExternalImplantBinary{
		Name:           name,
		ImplantBuildID: implantBuildID,
		File:           &commonpb.File{Data: fileData},
	})
}

func (a *App) BuilderTrigger(eventType string, data []byte) error {
	return a.Builders.Trigger(&clientpb.Event{EventType: eventType, Data: data})
}
