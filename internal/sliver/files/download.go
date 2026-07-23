package files

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
)

const defaultChunkSize = 16 * 1024 * 1024

func (s *Service) downloadToFile(
	sessionID, remotePath, localPath string, emit func(string, int64, int64),
) (int64, error) {
	f, err := os.OpenFile(localPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return 0, fmt.Errorf("failed to create file: %w", err)
	}
	defer func() { _ = f.Close() }()

	offset := int64(0)
	totalWritten := int64(0)
	for {
		data, stop, err := s.downloadChunk(sessionID, remotePath, offset, defaultChunkSize)
		if err != nil {
			return totalWritten, fmt.Errorf("chunk at offset %d: %w", offset, err)
		}
		if len(data) == 0 {
			break
		}

		if emit != nil {
			emit("received", offset+int64(len(data)), 0)
		}
		n, err := gzipWriteTo(f, data)
		if err != nil {
			return totalWritten, fmt.Errorf("decompress: %w", err)
		}
		totalWritten += n
		if emit != nil {
			emit("write", totalWritten, 0)
		}

		prevOffset := offset
		offset = stop
		if stop < prevOffset+defaultChunkSize {
			break
		}
	}
	return totalWritten, nil
}

func (s *Service) downloadChunk(
	sessionID, remotePath string, offset, chunkSize int64,
) ([]byte, int64, error) {
	chunkEnd := offset + chunkSize
	req := &sliverpb.DownloadReq{
		Request: &commonpb.Request{
			SessionID: sessionID,
			Timeout:   int64(defaultRPCTimeout / time.Second),
		},
		Path:  remotePath,
		Start: offset,
		Stop:  chunkEnd,
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultRPCTimeout)
	defer cancel()

	resp, err := s.downloadRPC(ctx, req)
	if err != nil {
		return nil, 0, err
	}
	return resp.Data, resp.Stop, nil
}

func gzipWriteTo(w io.Writer, data []byte) (int64, error) {
	gzr, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return 0, err
	}
	defer func() { _ = gzr.Close() }()
	return io.Copy(w, gzr)
}

func (s *Service) downloadData(
	sessionID, remotePath string, recurse bool, emit func(string, int64, int64),
) ([]byte, error) {
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

	resp, err := s.downloadRPC(ctx, req)
	if err != nil {
		return nil, err
	}

	compressedSize := int64(len(resp.Data))
	emit("received", compressedSize, compressedSize)

	if resp.Encoder == "gzip" {
		decoded, err := decompressGzip(resp.Data, compressedSize, emit)
		if err != nil {
			return nil, err
		}
		return decoded, nil
	}
	return resp.Data, nil
}

func decompressGzip(data []byte, compressedSize int64, emit func(string, int64, int64)) ([]byte, error) {
	reader := bytes.NewReader(data)
	gzipReader, err := gzip.NewReader(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer func() { _ = gzipReader.Close() }()

	var buf bytes.Buffer
	chunk := make([]byte, 32*1024)
	for {
		n, readErr := gzipReader.Read(chunk)
		if n > 0 {
			buf.Write(chunk[:n])
			if emit != nil {
				consumed := compressedSize - int64(reader.Len())
				emit("decompress", consumed, compressedSize)
			}
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
