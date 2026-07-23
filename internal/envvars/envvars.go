package envvars

import (
	"fmt"
	"os"

	"github.com/bishopfox/sliver/client/assets"
)

type EnvVar struct {
	Name  string `json:"name"`
	Value string `json:"value"`
	Set   bool   `json:"set"`
}

type EnvInfo struct {
	EffectiveDataDir string   `json:"effective_data_dir"`
	EffectiveLogDir  string   `json:"effective_log_dir"`
	ConfigDir        string   `json:"config_dir"`
	RootDir          string   `json:"root_dir"`
	ActiveVars       []EnvVar `json:"active_vars"`
}

var monitoredVars = []string{
	"SLIVER_CLIENT_ROOT_DIR",
	"SLIVER_GUI_DATA_DIR",
	"SLIVER_GUI_LOG_DIR",
	"SLIVER_NO_UPDATE_CHECK",
}

var PassthroughEnvVars = []string{
	"SLIVER_CLIENT_ROOT_DIR",
	"SLIVER_GUI_DATA_DIR",
	"SLIVER_GUI_LOG_DIR",
	"SLIVER_NO_UPDATE_CHECK",
	"HOME", "USER", "PATH",
	"NO_COLOR", "NO_PROXY", "HTTP_PROXY", "HTTPS_PROXY",
}

func ResolveDataDir(guiCfg *GUIConfig) (string, error) {
	if guiCfg != nil && guiCfg.DataDirOverride != "" {
		return MustDir(guiCfg.DataDirOverride)
	}
	if v := os.Getenv("SLIVER_GUI_DATA_DIR"); v != "" {
		return MustDir(v)
	}
	return assets.GetRootAppDir(), nil
}

func ResolveLogDir(guiCfg *GUIConfig) (string, error) {
	if guiCfg != nil && guiCfg.LogDirOverride != "" {
		return MustDir(guiCfg.LogDirOverride)
	}
	if v := os.Getenv("SLIVER_GUI_LOG_DIR"); v != "" {
		return MustDir(v)
	}
	return assets.GetClientLogsDir(), nil
}

func MustDir(path string) (string, error) {
	if err := os.MkdirAll(path, 0o700); err != nil {
		return "", fmt.Errorf("cannot create directory %s: %w", path, err)
	}
	return path, nil
}

func GetEnvInfo(guiCfg *GUIConfig) EnvInfo {
	dataDir, _ := ResolveDataDir(guiCfg)
	logDir, _ := ResolveLogDir(guiCfg)
	info := EnvInfo{
		EffectiveDataDir: dataDir,
		EffectiveLogDir:  logDir,
		ConfigDir:        assets.GetConfigDir(),
		RootDir:          assets.GetRootAppDir(),
		ActiveVars:       make([]EnvVar, len(monitoredVars)),
	}
	for i, name := range monitoredVars {
		v, ok := os.LookupEnv(name)
		info.ActiveVars[i] = EnvVar{Name: name, Value: v, Set: ok}
	}
	return info
}

func BuildPassthroughEnv(extra ...string) []string {
	var env []string
	for _, name := range PassthroughEnvVars {
		if v, ok := os.LookupEnv(name); ok {
			env = append(env, name+"="+v)
		}
	}
	env = append(env, extra...)
	return env
}
