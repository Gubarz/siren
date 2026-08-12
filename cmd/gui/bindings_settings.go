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
	rootDir := assets.GetRootAppDir()
	cfg, _ := envvars.LoadGUIConfig(rootDir)
	cfg.DataDirOverride = dir
	if _, err := envvars.MustDir(dir); err != nil {
		return err
	}
	return envvars.SaveGUIConfig(rootDir, cfg)
}

func (a *App) ClearDataDirOverride() error {
	rootDir := assets.GetRootAppDir()
	cfg, _ := envvars.LoadGUIConfig(rootDir)
	cfg.DataDirOverride = ""
	return envvars.SaveGUIConfig(rootDir, cfg)
}

func (a *App) SetLogDirOverride(dir string) error {
	rootDir := assets.GetRootAppDir()
	cfg, _ := envvars.LoadGUIConfig(rootDir)
	cfg.LogDirOverride = dir
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
