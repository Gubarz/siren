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
	override := ""
	if guiCfg != nil {
		override = guiCfg.DataDirOverride
	}
	return resolveDir(override, "SLIVER_GUI_DATA_DIR", assets.GetRootAppDir)
}

func ResolveLogDir(guiCfg *GUIConfig) (string, error) {
	override := ""
	if guiCfg != nil {
		override = guiCfg.LogDirOverride
	}
	return resolveDir(override, "SLIVER_GUI_LOG_DIR", assets.GetClientLogsDir)
}

// resolveDir applies the GUI override, then the env var, then the fallback.
// Every returned directory exists.
func resolveDir(override, envName string, fallback func() string) (string, error) {
	if override != "" {
		return MustDir(override)
	}
	if v := os.Getenv(envName); v != "" {
		return MustDir(v)
	}
	return fallback(), nil
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
