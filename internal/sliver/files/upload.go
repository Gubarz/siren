package files

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/wailsapp/wails/v2/pkg/runtime"

	"sliver-gui/internal/sliver/rpc"
)

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
	defer func() { _ = f.Close() }()

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
	request, err := s.rpc.TargetRequest(sessionID, defaultRPCTimeout)
	if err != nil {
		return err
	}
	req := &sliverpb.UploadReq{
		Request:  request,
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
	if err := s.rpc.AwaitAsyncResponse(ctx, resp, resp); err != nil {
		return err
	}
	if resp.WrittenFiles == 0 {
		return fmt.Errorf("target did not report writing the uploaded file")
	}
	return nil
}

func (s *Service) compressFileGzip(f *os.File, totalSize int64, localPath string) ([]byte, error) {
	pr, pw := io.Pipe()
	gw, _ := gzip.NewWriterLevel(pw, gzip.BestSpeed)

	go func() {
		_, copyErr := io.Copy(gw, f)
		_ = gw.Close()
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
