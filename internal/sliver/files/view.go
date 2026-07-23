package files

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"time"

	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"

	"sliver-gui/internal/sliver/rpc"
)

const maxViewSize = 10 * 1024 * 1024

func (s *Service) ViewRemoteFile(sessionID, remotePath string) (string, error) {
	if !s.rpc.Connected() {
		return "", rpc.ErrNotConnected
	}

	req := &sliverpb.DownloadReq{
		Request: &commonpb.Request{
			SessionID: sessionID,
			Timeout:   int64(defaultRPCTimeout / time.Second),
		},
		Path:             remotePath,
		MaxBytes:         maxViewSize,
		RestrictedToFile: true,
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultRPCTimeout)
	defer cancel()

	resp, err := s.rpc.RPC.Download(ctx, req)
	if err != nil {
		return "", err
	}
	if err := rpc.CheckResponse(resp); err != nil {
		return "", err
	}

	if resp.Encoder == "gzip" {
		decoded, err := decompressGzipBytes(resp.Data)
		if err != nil {
			return "", fmt.Errorf("decompress: %w", err)
		}
		resp.Data = decoded
	}

	return base64.StdEncoding.EncodeToString(resp.Data), nil
}

func decompressGzipBytes(data []byte) ([]byte, error) {
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer func() { _ = reader.Close() }()
	var buf bytes.Buffer
	_, err = io.Copy(&buf, reader)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
