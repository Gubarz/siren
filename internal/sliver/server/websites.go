package server

import (
	"context"
	"encoding/json"
	"os"

	"github.com/bishopfox/sliver/client/assets"
	"github.com/bishopfox/sliver/client/command/alias"
	"github.com/bishopfox/sliver/client/command/extensions"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"

	"siren/internal/sliver/rpc"
)

type AliasInfo struct {
	Name        string `json:"name"`
	CommandName string `json:"commandName"`
	Version     string `json:"version"`
	Type        string `json:"type"`
	Help        string `json:"help"`
}

func (s *Service) GetWebsites() (*clientpb.Websites, error) {
	if !s.rpc.Connected() {
		return nil, rpc.ErrNotConnected
	}
	return s.rpc.Websites(context.Background(), &commonpb.Empty{})
}

func (s *Service) GetAliases() (interface{}, error) {
	result := []AliasInfo{}

	for _, path := range assets.GetInstalledAliasManifests() {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var m alias.AliasManifest
		if err := json.Unmarshal(data, &m); err != nil {
			continue
		}
		result = append(result, AliasInfo{
			Name: m.Name, CommandName: m.CommandName,
			Version: m.Version, Type: "alias", Help: m.Help,
		})
	}

	for _, path := range assets.GetInstalledExtensionManifests() {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var m extensions.ExtensionManifest
		if err := json.Unmarshal(data, &m); err != nil {
			continue
		}
		for _, cmd := range m.ExtCommand {
			result = append(result, AliasInfo{
				Name: m.Name, CommandName: cmd.CommandName,
				Version: m.Version, Type: "extension", Help: cmd.Help,
			})
		}
	}

	return result, nil
}
