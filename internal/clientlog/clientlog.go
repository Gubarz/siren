// Package clientlog batches GUI-side log lines and streams them to the
// teamserver via the ClientLog RPC. The intent is that when an operator
// hits a UI bug, the server's operator log already has our matching
// context — no more "paste your stderr" back-and-forth.
package clientlog

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"

	"sliver-gui/internal/rpc"
)

const (
	defaultStream   = "sliver-gui"
	flushInterval   = 2 * time.Second
	channelCapacity = 256
)

// Service owns a background goroutine that drains queued log entries into
// the sliver ClientLog stream. Callers just fire-and-forget via Log().
type Service struct {
	rpc *rpc.Client

	mu      sync.Mutex
	queue   chan []byte
	cancel  context.CancelFunc
	running bool
}

func New(client *rpc.Client) *Service {
	return &Service{rpc: client}
}

// Start opens the streaming RPC and spins up the pump goroutine. Safe to
// call multiple times — subsequent calls are no-ops.
func (s *Service) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.running {
		return nil
	}
	if !s.rpc.Connected() {
		return rpc.ErrNotConnected
	}
	pumpCtx, cancel := context.WithCancel(ctx)
	stream, err := s.rpc.RPC.ClientLog(pumpCtx)
	if err != nil {
		cancel()
		return err
	}
	s.queue = make(chan []byte, channelCapacity)
	s.cancel = cancel
	s.running = true
	go s.pump(pumpCtx, stream)
	return nil
}

// Log enqueues a line for delivery. Drops silently if the queue is full or
// the pump isn't running — logging must never block the UI thread.
func (s *Service) Log(line string) {
	s.mu.Lock()
	queue := s.queue
	s.mu.Unlock()
	if queue == nil {
		return
	}
	select {
	case queue <- []byte(line):
	default:
	}
}

// Close stops the pump and closes the underlying stream.
func (s *Service) Close() {
	s.mu.Lock()
	cancel := s.cancel
	s.cancel = nil
	s.queue = nil
	s.running = false
	s.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

// pump reads from the queue and writes to the stream. A ticker flushes
// nothing itself but keeps the loop responsive for shutdown.
func (s *Service) pump(ctx context.Context, stream rpcpb.SliverRPC_ClientLogClient) {
	defer stream.CloseSend()
	ticker := time.NewTicker(flushInterval)
	defer ticker.Stop()
	for {
		if err := s.pumpTick(ctx, stream, ticker); err != nil {
			return
		}
	}
}

// pumpTick handles a single loop iteration; split out so pump() stays under
// the 45-line budget and the select clauses remain readable.
func (s *Service) pumpTick(ctx context.Context, stream rpcpb.SliverRPC_ClientLogClient, ticker *time.Ticker) error {
	s.mu.Lock()
	queue := s.queue
	s.mu.Unlock()
	if queue == nil {
		return errors.New("clientlog: queue closed")
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-ticker.C:
		return nil
	case data, ok := <-queue:
		if !ok {
			return errors.New("clientlog: queue closed")
		}
		return stream.Send(&clientpb.ClientLogData{Stream: defaultStream, Data: data})
	}
}
