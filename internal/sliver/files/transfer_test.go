package files

import (
	"bytes"
	"compress/gzip"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/bishopfox/sliver/protobuf/sliverpb"

	"sliver-gui/internal/sliver/rpc"
)

func gzipBytes(data []byte) []byte {
	var buf bytes.Buffer
	w, _ := gzip.NewWriterLevel(&buf, gzip.BestSpeed)
	w.Write(data)
	w.Close()
	return buf.Bytes()
}

func TestDownloadChunked_SingleChunk(t *testing.T) {
	payload := []byte("hello world")
	s := &Service{
		rpc:     &rpc.Client{},
		history: NewHistoryStore(""),
		dl: func(ctx context.Context, req *sliverpb.DownloadReq) (*sliverpb.Download, error) {
			return &sliverpb.Download{
				Data: gzipBytes(payload), Start: req.Start, Stop: int64(len(payload)),
			}, nil
		},
	}
	dest := filepath.Join(t.TempDir(), "out.bin")
	written, err := s.downloadToFile("sess-1", "/tmp/remote", dest, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if written != int64(len(payload)) {
		t.Fatalf("expected %d bytes, got %d", len(payload), written)
	}
	data, _ := os.ReadFile(dest)
	if !bytes.Equal(data, payload) {
		t.Fatalf("expected %q, got %q", payload, data)
	}
}

func TestDownloadChunked_Multi(t *testing.T) {
	payload := make([]byte, defaultChunkSize+100)
	for i := range payload {
		payload[i] = byte(i % 256)
	}
	chunks := [][]byte{
		gzipBytes(payload[:defaultChunkSize]),
		gzipBytes(payload[defaultChunkSize:]),
	}
	s := &Service{
		rpc:     &rpc.Client{},
		history: NewHistoryStore(""),
		dl: func(ctx context.Context, req *sliverpb.DownloadReq) (*sliverpb.Download, error) {
			idx := int(req.Start / defaultChunkSize)
			if idx >= len(chunks) {
				return &sliverpb.Download{Start: req.Start, Stop: req.Start}, nil
			}
			stop := req.Start + int64(len(payload)-int(req.Start))
			if stop > req.Stop {
				stop = req.Stop
			}
			return &sliverpb.Download{
				Data: chunks[idx], Start: req.Start, Stop: stop,
			}, nil
		},
	}
	dest := filepath.Join(t.TempDir(), "out.bin")
	written, err := s.downloadToFile("sess-1", "/tmp/remote", dest, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if written != int64(len(payload)) {
		t.Fatalf("expected %d bytes, got %d", len(payload), written)
	}
	data, _ := os.ReadFile(dest)
	if !bytes.Equal(data, payload) {
		t.Fatalf("output mismatch: len(expected)=%d, len(got)=%d", len(payload), len(data))
	}
}

func TestDownloadChunked_Error(t *testing.T) {
	payload := make([]byte, defaultChunkSize+100)
	s := &Service{
		rpc:     &rpc.Client{},
		history: NewHistoryStore(""),
		dl: func(ctx context.Context, req *sliverpb.DownloadReq) (*sliverpb.Download, error) {
			if req.Start == 0 {
				return &sliverpb.Download{
					Data: gzipBytes(payload[:defaultChunkSize]),
					Start: 0, Stop: defaultChunkSize,
				}, nil
			}
			return nil, context.DeadlineExceeded
		},
	}
	dest := filepath.Join(t.TempDir(), "out.bin")
	written, err := s.downloadToFile("sess-1", "/tmp/remote", dest, nil)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if written != int64(defaultChunkSize) {
		t.Fatalf("expected partial write of %d, got %d", defaultChunkSize, written)
	}
}
