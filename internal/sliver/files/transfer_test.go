package files

import (
	"bytes"
	"compress/gzip"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"

	"siren/internal/sliver/rpc"
)

func gzipBytes(data []byte) []byte {
	var buf bytes.Buffer
	w, _ := gzip.NewWriterLevel(&buf, gzip.BestSpeed)
	w.Write(data)
	w.Close()
	return buf.Bytes()
}

// sessionClient returns an rpc.Client whose agent cache knows sessionID as a
// session, so TargetRequest resolves it. Downloads route through
// TargetRequest like every other agent RPC.
func sessionClient(sessionID string) *rpc.Client {
	c := &rpc.Client{}
	c.PopulateSessions(&clientpb.Sessions{
		Sessions: []*clientpb.Session{{ID: sessionID}},
	})
	return c
}

// beaconClient returns an rpc.Client whose cache knows beaconID as a beacon
// and whose RPC stub serves the given task-content handler.
func beaconClient(beaconID string, handler fakeTaskHandler) *rpc.Client {
	c := &rpc.Client{}
	c.PopulateBeacons(&clientpb.Beacons{
		Beacons: []*clientpb.Beacon{{ID: beaconID}},
	})
	c.RPC = &fakeTaskRPC{getBeaconTaskContent: handler}
	return c
}

type fakeTaskHandler func(ctx context.Context, in *clientpb.BeaconTask, opts ...grpc.CallOption) (*clientpb.BeaconTask, error)

type fakeTaskRPC struct {
	rpcpb.SliverRPCClient
	getBeaconTaskContent fakeTaskHandler
}

func (f *fakeTaskRPC) GetBeaconTaskContent(
	ctx context.Context, in *clientpb.BeaconTask, opts ...grpc.CallOption,
) (*clientpb.BeaconTask, error) {
	return f.getBeaconTaskContent(ctx, in, opts...)
}

func TestDownloadChunked_SingleChunk(t *testing.T) {
	payload := []byte("hello world")
	s := &Service{
		rpc:     sessionClient("sess-1"),
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
		rpc:     sessionClient("sess-1"),
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
		rpc:     sessionClient("sess-1"),
		history: NewHistoryStore(""),
		dl: func(ctx context.Context, req *sliverpb.DownloadReq) (*sliverpb.Download, error) {
			if req.Start == 0 {
				return &sliverpb.Download{
					Data:  gzipBytes(payload[:defaultChunkSize]),
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

// Regression: downloads against a beacon must task the beacon (Async +
// BeaconID on the request) and await the queued task's payload, instead of
// issuing a session-scoped request that the server rejects with an RPC error.
func TestDownloadChunked_Beacon(t *testing.T) {
	payload := []byte("beacon file contents")
	taskData, err := proto.Marshal(&sliverpb.Download{
		Data: gzipBytes(payload),
		Stop: int64(len(payload)),
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var sawBeaconRequest bool
	handler := func(_ context.Context, in *clientpb.BeaconTask, _ ...grpc.CallOption) (*clientpb.BeaconTask, error) {
		return &clientpb.BeaconTask{ID: in.ID, State: "completed", Response: taskData}, nil
	}
	s := &Service{
		rpc:     beaconClient("beacon-1", handler),
		history: NewHistoryStore(""),
		dl: func(ctx context.Context, req *sliverpb.DownloadReq) (*sliverpb.Download, error) {
			if req.Request.GetBeaconID() == "beacon-1" && req.Request.GetAsync() {
				sawBeaconRequest = true
			}
			return &sliverpb.Download{
				Response: &commonpb.Response{Async: true, TaskID: "task-1"},
			}, nil
		},
	}

	dest := filepath.Join(t.TempDir(), "out.bin")
	written, err := s.downloadToFile("beacon-1", "/tmp/remote", dest, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !sawBeaconRequest {
		t.Fatal("expected download request to carry BeaconID + Async for a beacon target")
	}
	if written != int64(len(payload)) {
		t.Fatalf("expected %d bytes, got %d", len(payload), written)
	}
	data, _ := os.ReadFile(dest)
	if !bytes.Equal(data, payload) {
		t.Fatalf("expected %q, got %q", payload, data)
	}
}
