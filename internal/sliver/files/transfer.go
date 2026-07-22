package files

import (
	"archive/tar"
	"context"
	"fmt"
	"os"
	goruntime "runtime"
	"strings"
	"time"

	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/wailsapp/wails/v2/pkg/runtime"

	"sliver-gui/internal/sliver/rpc"
)

func (s *Service) DownloadDirectory(sessionID string, remotePath string) error {
	return s.downloadWithDialog(sessionID, remotePath, "Save Directory", remoteFilename(remotePath)+".tar", true)
}

func (s *Service) DownloadFile(sessionID string, remotePath string) error {
	return s.downloadWithDialog(sessionID, remotePath, "Save File", remoteFilename(remotePath), false)
}

func (s *Service) downloadWithDialog(sessionID, remotePath, title, defaultName string, recurse bool) error {
	if !s.rpc.Connected() {
		return rpc.ErrNotConnected
	}

	goruntime.LockOSThread()
	localPath, err := runtime.SaveFileDialog(s.ctx, runtime.SaveDialogOptions{
		Title:           title,
		DefaultFilename: defaultName,
	})
	goruntime.UnlockOSThread()
	if err != nil {
		return fmt.Errorf("dialog error: %w", err)
	}
	if localPath == "" {
		return nil
	}

	return s.downloadToPath(sessionID, remotePath, localPath, recurse)
}

func (s *Service) downloadToPath(sessionID, remotePath, localPath string, recurse bool) error {
	emit := s.downloadEmitter(localPath)
	emit("request", 0, 0)

	recordID := s.addDownloadHistory(sessionID, remotePath, localPath, recurse)

	var totalWritten int64
	var err error
	if recurse {
		totalWritten, err = s.singleShotDownload(sessionID, remotePath, localPath, emit)
	} else {
		totalWritten, err = s.downloadToFile(sessionID, remotePath, localPath, emit)
	}
	if err != nil {
		s.finalizeDownloadHistory(recordID, "failed", totalWritten, err.Error())
		return err
	}

	s.finalizeDownloadHistory(recordID, "completed", totalWritten, "")
	emit("done", totalWritten, totalWritten)
	return nil
}

func (s *Service) singleShotDownload(
	sessionID, remotePath, localPath string, emit func(string, int64, int64),
) (int64, error) {
	data, err := s.downloadData(sessionID, remotePath, true, emit)
	if err != nil {
		return 0, err
	}
	if err := os.WriteFile(localPath, data, 0644); err != nil {
		return 0, err
	}
	return int64(len(data)), nil
}

func (s *Service) downloadItemToFile(
	sessionID string, item BulkDownloadItem, dest string, emit func(string, int64, int64),
) (int64, error) {
	if item.IsDirectory {
		return s.singleShotDownload(sessionID, item.RemotePath, dest, emit)
	}
	return s.downloadToFile(sessionID, item.RemotePath, dest, emit)
}

func (s *Service) addDownloadHistory(sessionID, remotePath, localPath string, isDir bool) string {
	if s.history == nil {
		return ""
	}
	return s.history.AddRecord(DownloadRecord{
		SessionID:   sessionID,
		RemotePath:  remotePath,
		LocalPath:   localPath,
		IsDirectory: isDir,
		Timestamp:   nowString(),
		Status:      "in_progress",
	})
}

func (s *Service) finalizeDownloadHistory(recordID, status string, size int64, errStr string) {
	if s.history != nil && recordID != "" {
		s.history.UpdateRecord(recordID, status, size, errStr)
	}
}

func (s *Service) downloadEmitter(localPath string) func(string, int64, int64) {
	return func(phase string, current, total int64) {
		if s.ctx != nil {
			runtime.EventsEmit(s.ctx, "download-progress", map[string]interface{}{
				"path":    localPath,
				"phase":   phase,
				"current": current,
				"total":   total,
			})
		}
	}
}

func remoteFilename(remotePath string) string {
	if idx := strings.LastIndexAny(remotePath, "/\\"); idx >= 0 {
		return remotePath[idx+1:]
	}
	return remotePath
}

type BulkDownloadItem struct {
	RemotePath  string `json:"remotePath"`
	IsDirectory bool   `json:"isDirectory"`
}

func (s *Service) DownloadMultipleTar(sessionID string, items []BulkDownloadItem) error {
	if !s.rpc.Connected() {
		return rpc.ErrNotConnected
	}
	if len(items) == 0 {
		return nil
	}

	defaultName := fmt.Sprintf("archive_%d.tar", time.Now().Unix())
	goruntime.LockOSThread()
	localPath, err := runtime.SaveFileDialog(s.ctx, runtime.SaveDialogOptions{
		Title:           "Save Tar Archive",
		DefaultFilename: defaultName,
	})
	goruntime.UnlockOSThread()
	if err != nil {
		return fmt.Errorf("dialog error: %w", err)
	}
	if localPath == "" {
		return nil
	}

	emit := s.downloadEmitter(localPath)
	emit("request", 0, 0)

	outFile, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("failed to create archive file: %w", err)
	}
	defer outFile.Close()

	tw := tar.NewWriter(outFile)
	defer tw.Close()

	for _, item := range items {
		if err := s.downloadMultipleItem(sessionID, item, localPath, tw, emit); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) downloadMultipleItem(
	sessionID string, item BulkDownloadItem, localPath string,
	tw *tar.Writer, emit func(string, int64, int64),
) error {
	recordID := s.addDownloadHistory(sessionID, item.RemotePath, localPath, item.IsDirectory)
	tmpFile, err := os.CreateTemp("", "sliver-dl-*")
	if err != nil {
		s.finalizeDownloadHistory(recordID, "failed", 0, err.Error())
		return fmt.Errorf("temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	written, err := s.downloadItemToFile(sessionID, item, tmpFile.Name(), emit)
	if err != nil {
		s.finalizeDownloadHistory(recordID, "failed", written, err.Error())
		return fmt.Errorf("failed to download %s: %w", item.RemotePath, err)
	}
	if _, err := tmpFile.Seek(0, 0); err != nil {
		s.finalizeDownloadHistory(recordID, "failed", written, err.Error())
		return err
	}

	baseName := remoteFilename(item.RemotePath)
	var tarErr error
	if item.IsDirectory {
		tarErr = addDirToTar(tw, baseName, tmpFile)
	} else {
		tarErr = addFileToTar(tw, baseName, tmpFile)
	}
	if tarErr != nil {
		s.finalizeDownloadHistory(recordID, "failed", written, tarErr.Error())
		return tarErr
	}
	s.finalizeDownloadHistory(recordID, "completed", written, "")
	return nil
}

func (s *Service) downloadRPC(ctx context.Context, req *sliverpb.DownloadReq) (*sliverpb.Download, error) {
	if s.dl != nil {
		return s.dl(ctx, req)
	}
	return s.rpc.RPC.Download(ctx, req)
}


