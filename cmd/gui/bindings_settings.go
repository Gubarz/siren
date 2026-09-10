package gui

import (
	"github.com/bishopfox/sliver/client/assets"

	"siren/internal/buildinfo"
	"siren/internal/envvars"
	"siren/internal/sliver/health"
)

func (a *App) GetBuildInfo() buildinfo.Info {
	return buildinfo.Get()
}

// ---- Health ----

func (a *App) HealthSnapshot() health.Snapshot {
	return a.Health.Snapshot()
}

// ---- Env Vars ----

func (a *App) GetEnvInfo() envvars.EnvInfo {
	guiCfg, _ := envvars.LoadGUIConfig(assets.GetRootAppDir())
	return envvars.GetEnvInfo(guiCfg)
}

func (a *App) SetDataDirOverride(dir string) error {
	return setDirOverride(dir, func(cfg *envvars.GUIConfig) { cfg.DataDirOverride = dir })
}

func (a *App) ClearDataDirOverride() error {
	rootDir := assets.GetRootAppDir()
	cfg, _ := envvars.LoadGUIConfig(rootDir)
	cfg.DataDirOverride = ""
	return envvars.SaveGUIConfig(rootDir, cfg)
}

func (a *App) SetLogDirOverride(dir string) error {
	return setDirOverride(dir, func(cfg *envvars.GUIConfig) { cfg.LogDirOverride = dir })
}

// setDirOverride persists the mutated GUI config only when dir exists or can
// be created.
func setDirOverride(dir string, apply func(*envvars.GUIConfig)) error {
	rootDir := assets.GetRootAppDir()
	cfg, _ := envvars.LoadGUIConfig(rootDir)
	apply(cfg)
	if _, err := envvars.MustDir(dir); err != nil {
		return err
	}
	return envvars.SaveGUIConfig(rootDir, cfg)
}

func (a *App) ClearLogDirOverride() error {
	rootDir := assets.GetRootAppDir()
	cfg, _ := envvars.LoadGUIConfig(rootDir)
	cfg.LogDirOverride = ""
	return envvars.SaveGUIConfig(rootDir, cfg)
}
