package files

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/wailsapp/wails/v2/pkg/runtime"

	"sliver-gui/internal/rpc"
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

	localPath, err := runtime.SaveFileDialog(s.ctx, runtime.SaveDialogOptions{
		Title:           title,
		DefaultFilename: defaultName,
	})
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

	req := &sliverpb.DownloadReq{
		Request: &commonpb.Request{
			SessionID: sessionID,
			Timeout:   int64(defaultRPCTimeout / time.Second),
		},
		Path:    remotePath,
		Recurse: recurse,
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultRPCTimeout)
	defer cancel()

	resp, err := s.rpc.RPC.Download(ctx, req)
	if err != nil {
		return err
	}

	compressedSize := int64(len(resp.Data))
	emit("received", compressedSize, compressedSize)

	if resp.Encoder == "gzip" {
		decoded, err := decompressGzip(resp.Data, compressedSize, emit)
		if err != nil {
			return err
		}
		resp.Data = decoded
	}

	emit("write", 0, 0)
	if err := os.WriteFile(localPath, resp.Data, 0644); err != nil {
		return err
	}
	emit("done", 0, 0)
	return nil
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

func decompressGzip(data []byte, compressedSize int64, emit func(string, int64, int64)) ([]byte, error) {
	reader := bytes.NewReader(data)
	gzipReader, err := gzip.NewReader(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzipReader.Close()

	var buf bytes.Buffer
	chunk := make([]byte, 32*1024)
	for {
		n, readErr := gzipReader.Read(chunk)
		if n > 0 {
			buf.Write(chunk[:n])
			consumed := compressedSize - int64(reader.Len())
			emit("decompress", consumed, compressedSize)
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return nil, fmt.Errorf("failed to decode gzip download: %w", readErr)
		}
	}
	return buf.Bytes(), nil
}

func (s *Service) UploadFile(sessionID string, remotePath string) error {
	if !s.rpc.Connected() {
		return rpc.ErrNotConnected
	}

	localPath, err := runtime.OpenFileDialog(s.ctx, runtime.OpenDialogOptions{
		Title: "Select File to Upload",
	})
	if err != nil {
		return fmt.Errorf("dialog error: %w", err)
	}
	if localPath == "" {
		return nil
	}

	return s.uploadLocalFile(sessionID, remotePath, localPath)
}

func (s *Service) UploadFiles(sessionID string, remotePath string, localPaths []string) error {
	if !s.rpc.Connected() {
		return rpc.ErrNotConnected
	}
	for _, localPath := range localPaths {
		info, err := os.Stat(localPath)
		if err != nil {
			return fmt.Errorf("%s: %w", filepath.Base(localPath), err)
		}
		if info.IsDir() {
			return fmt.Errorf("%s is a directory; drop files only", filepath.Base(localPath))
		}
		if err := s.uploadLocalFile(sessionID, remotePath, localPath); err != nil {
			return fmt.Errorf("%s: %w", filepath.Base(localPath), err)
		}
	}
	return nil
}

func (s *Service) uploadLocalFile(sessionID string, remotePath string, localPath string) error {
	f, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("failed to open local file: %w", err)
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat local file: %w", err)
	}
	totalSize := info.Size()

	encodedData, err := s.compressFileGzip(f, totalSize, localPath)
	if err != nil {
		return err
	}

	if s.ctx != nil {
		runtime.EventsEmit(s.ctx, "file-upload-progress", map[string]interface{}{
			"path":  localPath,
			"total": totalSize,
			"phase": "upload",
		})
	}

	return s.uploadRequest(sessionID, remotePath, filepath.Base(localPath), encodedData)
}

func (s *Service) uploadRequest(sessionID, remotePath, fileName string, encodedData []byte) error {
	req := &sliverpb.UploadReq{
		Request: &commonpb.Request{
			SessionID: sessionID,
			Timeout:   int64(defaultRPCTimeout / time.Second),
		},
		Path:     remotePath,
		FileName: fileName,
		Data:     encodedData,
		Encoder:  "gzip",
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultRPCTimeout)
	defer cancel()

	resp, err := s.rpc.RPC.Upload(ctx, req)
	if err != nil {
		return err
	}
	if err := rpc.CheckResponse(resp); err != nil {
		return err
	}
	if resp.WrittenFiles == 0 {
		return fmt.Errorf("target did not report writing the uploaded file")
	}
	return nil
}

func remoteFilename(remotePath string) string {
	if idx := strings.LastIndexAny(remotePath, "/\\"); idx >= 0 {
		return remotePath[idx+1:]
	}
	return remotePath
}

func (s *Service) compressFileGzip(f *os.File, totalSize int64, localPath string) ([]byte, error) {
	pr, pw := io.Pipe()
	gw, _ := gzip.NewWriterLevel(pw, gzip.BestSpeed)

	go func() {
		_, copyErr := io.Copy(gw, f)
		gw.Close()
		if copyErr != nil {
			_ = pw.CloseWithError(fmt.Errorf("gzip: %w", copyErr))
		} else {
			_ = pw.Close()
		}
	}()

	var buf bytes.Buffer
	chunk := make([]byte, 32*1024)
	for {
		n, readErr := pr.Read(chunk)
		if n > 0 {
			buf.Write(chunk[:n])
			if s.ctx != nil {
				runtime.EventsEmit(s.ctx, "file-upload-progress", map[string]interface{}{
					"path":     localPath,
					"total":    totalSize,
					"compress": buf.Len(),
				})
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return nil, fmt.Errorf("failed to compress upload: %w", readErr)
		}
	}
	return buf.Bytes(), nil
}
