package gui

import (
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/wailsapp/wails/v2/pkg/runtime"

	"siren/internal/sliver/files"
	"siren/internal/sliver/registry"
)

// ---- Process / File Tools ----

func (a *App) GetProcessList(sessionID string, fullInfo bool) (*sliverpb.Ps, error) {
	return a.Procs.GetProcessList(sessionID, fullInfo)
}

func (a *App) KillProcess(sessionID string, pid int32) error {
	return a.Procs.KillProcess(sessionID, pid)
}

func (a *App) TakeScreenshot(sessionID string) (string, error) {
	return a.Procs.TakeScreenshot(sessionID)
}

func (a *App) GetFileList(sessionID, path string) (*sliverpb.Ls, error) {
	return a.Files.GetFileList(sessionID, path)
}

func (a *App) DownloadFile(sessionID, remotePath string) error {
	return a.Files.DownloadFile(sessionID, remotePath)
}

func (a *App) DownloadDirectory(sessionID, remotePath string) error {
	return a.Files.DownloadDirectory(sessionID, remotePath)
}

func (a *App) DownloadMultipleTar(sessionID string, items []files.BulkDownloadItem) error {
	return a.Files.DownloadMultipleTar(sessionID, items)
}

func (a *App) GetDownloadHistory(sessionID, remotePath string) ([]files.DownloadRecord, error) {
	return a.Files.GetDownloadHistory(sessionID, remotePath)
}

func (a *App) GetAllDownloadHistory() ([]files.DownloadRecord, error) {
	return a.Files.GetAllDownloadHistory()
}

func (a *App) ClearDownloadHistory(sessionID, remotePath string) error {
	return a.Files.ClearDownloadHistory(sessionID, remotePath)
}

func (a *App) UploadFile(sessionID, remotePath string) error {
	return a.Files.UploadFile(sessionID, remotePath)
}

func (a *App) UploadFiles(sessionID, remotePath string, localPaths []string) error {
	return a.Files.UploadFiles(sessionID, remotePath, localPaths)
}

func (a *App) Cd(sessionID, path string) (string, error) {
	return a.Files.Cd(sessionID, path)
}

func (a *App) Pwd(sessionID string) (string, error) {
	return a.Files.Pwd(sessionID)
}

func (a *App) MakeDir(sessionID, path string) error {
	return a.Files.MakeDir(sessionID, path)
}

func (a *App) RemovePath(sessionID, path string, recursive bool) error {
	return a.Files.RemovePath(sessionID, path, recursive)
}

func (a *App) RenamePath(sessionID, src, dst string) error {
	return a.Files.RenamePath(sessionID, src, dst)
}

func (a *App) Chmod(sessionID, path, mode string, recursive bool) error {
	return a.Files.Chmod(sessionID, path, mode, recursive)
}

func (a *App) Chown(sessionID, path, uid, gid string, recursive bool) error {
	return a.Files.Chown(sessionID, path, uid, gid, recursive)
}

func (a *App) Chtimes(sessionID, path string, atimeUnix, mtimeUnix int64) error {
	return a.Files.Chtimes(sessionID, path, atimeUnix, mtimeUnix)
}

func (a *App) CopyPath(sessionID, src, dst string) (int64, error) {
	return a.Files.CopyPath(sessionID, src, dst)
}

func (a *App) ViewRemoteFile(sessionID, remotePath string) (string, error) {
	return a.Files.ViewRemoteFile(sessionID, remotePath)
}

func (a *App) GrepFiles(sessionID, pattern, path string, recursive bool, beforeLines, afterLines int32) (string, error) {
	return a.Files.GrepFiles(sessionID, pattern, path, recursive, beforeLines, afterLines)
}

func (a *App) GetServices(sessionID string) (*sliverpb.Services, error) {
	return a.Services.GetServices(sessionID)
}

func (a *App) StartService(sessionID, name string) error {
	return a.Services.StartService(sessionID, name)
}

func (a *App) StopService(sessionID, name string) error {
	return a.Services.StopService(sessionID, name)
}

func (a *App) RemoveService(sessionID, name string) error {
	return a.Services.RemoveService(sessionID, name)
}

func (a *App) GetTokenPrivs(sessionID string) (*sliverpb.GetPrivs, error) {
	return a.Console.GetTokenPrivs(sessionID)
}

func (a *App) RevToSelfToken(sessionID string) error {
	return a.Console.RevToSelfToken(sessionID)
}

func (a *App) GetEnv(sessionID string) (*sliverpb.EnvInfo, error) {
	return a.Env.GetEnv(sessionID)
}

func (a *App) SetEnv(sessionID, key, value string) error {
	return a.Env.SetEnv(sessionID, key, value)
}

func (a *App) UnsetEnv(sessionID, name string) error {
	return a.Env.UnsetEnv(sessionID, name)
}

// ---- Registry ----

func (a *App) ListRegistrySubKeys(sessionID, hive, path string) (*sliverpb.RegistrySubKeyList, error) {
	return a.Registry.ListSubKeys(sessionID, hive, path)
}

func (a *App) ListRegistryValues(sessionID, hive, path string) (*sliverpb.RegistryValuesList, error) {
	return a.Registry.ListValues(sessionID, hive, path)
}

func (a *App) ReadRegistryValue(sessionID, hive, path, key string) (*registry.Value, error) {
	return a.Registry.ReadValue(sessionID, hive, path, key)
}

func (a *App) WriteRegistryValue(sessionID, hive, path, key, valueType, value string) error {
	err := a.Registry.WriteValue(sessionID, hive, path, key, valueType, value)
	if err == nil {
		runtime.EventsEmit(a.ctx, "registry-updated", nil)
	}
	return err
}

func (a *App) CreateRegistryKey(sessionID, hive, path, key string) error {
	err := a.Registry.CreateKey(sessionID, hive, path, key)
	if err == nil {
		runtime.EventsEmit(a.ctx, "registry-updated", nil)
	}
	return err
}

func (a *App) DeleteRegistryEntry(sessionID, hive, path, key string) error {
	err := a.Registry.DeleteEntry(sessionID, hive, path, key)
	if err == nil {
		runtime.EventsEmit(a.ctx, "registry-updated", nil)
	}
	return err
}

// ---- Memfiles ----

func (a *App) MemfilesList(sessionID string) (*sliverpb.Ls, error) {
	return a.Memfiles.List(sessionID)
}

func (a *App) MemfilesAdd(sessionID string) (*sliverpb.MemfilesAdd, error) {
	return a.Memfiles.Add(sessionID)
}

func (a *App) MemfilesRemove(sessionID string, fd int64) (*sliverpb.MemfilesRm, error) {
	return a.Memfiles.Remove(sessionID, fd)
}

// ---- Registry Read Hive ----

func (a *App) RegistryReadHive(sessionID, rootHive, requestedHive string) (*sliverpb.RegistryReadHive, error) {
	return a.Registry.ReadHive(sessionID, rootHive, requestedHive)
}
