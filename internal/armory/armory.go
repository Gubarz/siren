package armory

import (
	"sort"
	_ "unsafe"

	"github.com/bishopfox/sliver/client/command/alias"
	"github.com/bishopfox/sliver/client/command/extensions"

	"sliver-gui/internal/console"
)

//go:linkname packageManifestsInCache github.com/bishopfox/sliver/client/command/armory.packageManifestsInCache
func packageManifestsInCache() ([]*alias.AliasManifest, []*extensions.ExtensionManifest)

type PackageInfo struct {
	ArmoryName  string `json:"armoryName"`
	CommandName string `json:"commandName"`
	Version     string `json:"version"`
	Type        string `json:"type"`
	Help        string `json:"help"`
	RepoURL     string `json:"repoUrl"`
}

type Service struct {
	console *console.Service
}

func New(con *console.Service) *Service {
	return &Service{console: con}
}

func (s *Service) ListPackages(refresh bool) ([]PackageInfo, error) {
	if refresh {
		if _, err := s.console.RunLine("", "armory --ignore-cache"); err != nil {
			return nil, err
		}
	}

	aliases, exts := packageManifestsInCache()
	return buildPackageInfos(aliases, exts), nil
}

func buildPackageInfos(aliases []*alias.AliasManifest, exts []*extensions.ExtensionManifest) []PackageInfo {
	var pkgs []PackageInfo
	for _, a := range aliases {
		pkgs = append(pkgs, PackageInfo{
			ArmoryName:  a.ArmoryName,
			CommandName: a.CommandName,
			Version:     a.Version,
			Type:        "Alias",
			Help:        a.Help,
			RepoURL:     a.RepoURL,
		})
	}
	for _, ext := range exts {
		for _, cmd := range ext.ExtCommand {
			pkgs = append(pkgs, PackageInfo{
				ArmoryName:  ext.ArmoryName,
				CommandName: cmd.CommandName,
				Version:     ext.Version,
				Type:        "Extension",
				Help:        cmd.Help,
				RepoURL:     ext.RepoURL,
			})
		}
	}

	sort.Slice(pkgs, func(i, j int) bool {
		return pkgs[i].CommandName < pkgs[j].CommandName
	})
	return pkgs
}
