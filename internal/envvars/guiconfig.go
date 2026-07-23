package envvars

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const guiSettingsFilename = "gui-settings.yaml"

type GUIConfig struct {
	DataDirOverride string `yaml:"gui_data_dir,omitempty"`
	LogDirOverride  string `yaml:"gui_log_dir,omitempty"`
}

func LoadGUIConfig(rootDir string) (*GUIConfig, error) {
	path := filepath.Join(rootDir, guiSettingsFilename)
	cfg := &GUIConfig{}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, err
	}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func SaveGUIConfig(rootDir string, cfg *GUIConfig) error {
	if cfg == nil {
		cfg = &GUIConfig{}
	}
	path := filepath.Join(rootDir, guiSettingsFilename)
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}
