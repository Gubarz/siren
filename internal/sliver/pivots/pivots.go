package pivots

import (
	"context"
	"sync"
	"time"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"

	"siren/internal/sliver/rpc"
)

type PivotListenerSnapshot struct {
	ParentSessionID string
	ID              uint32
	Type            string
	BindAddress     string
	Pivots          []PivotConnectionSnapshot
}

type PivotConnectionSnapshot struct {
	PeerID        int64
	RemoteAddress string
}

type Service struct {
	rpc *rpc.Client
}

func New(rpc *rpc.Client) *Service {
	return &Service{rpc: rpc}
}

func (s *Service) StartListener(sessionID string, pivotType sliverpb.PivotType, bindAddr string) (*sliverpb.PivotListener, error) {
	if !s.rpc.Connected() {
		return nil, rpc.ErrNotConnected
	}
	return s.rpc.RPC.PivotStartListener(context.Background(), &sliverpb.PivotStartListenerReq{
		Type:        pivotType,
		BindAddress: bindAddr,
		Request:     &commonpb.Request{SessionID: sessionID},
	})
}

func (s *Service) StopListener(sessionID string, id uint32) error {
	if !s.rpc.Connected() {
		return rpc.ErrNotConnected
	}
	_, err := s.rpc.RPC.PivotStopListener(context.Background(), &sliverpb.PivotStopListenerReq{
		ID:      id,
		Request: &commonpb.Request{SessionID: sessionID},
	})
	return err
}

func (s *Service) GetPivots() (*clientpb.PivotGraph, error) {
	if !s.rpc.Connected() {
		return nil, rpc.ErrNotConnected
	}
	return s.rpc.RPC.PivotGraph(context.Background(), &commonpb.Empty{})
}

func (s *Service) GetPivotListeners() ([]PivotListenerSnapshot, error) {
	if !s.rpc.Connected() {
		return nil, rpc.ErrNotConnected
	}

	sessions, err := s.rpc.RPC.GetSessions(context.Background(), &commonpb.Empty{})
	if err != nil {
		return nil, err
	}

	snapshots := []PivotListenerSnapshot{}
	var snapshotsMu sync.Mutex
	var waitGroup sync.WaitGroup
	for _, session := range sessions.Sessions {
		sessionID := session.ID
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			found := s.sessionPivotSnapshots(sessionID)
			snapshotsMu.Lock()
			snapshots = append(snapshots, found...)
			snapshotsMu.Unlock()
		}()
	}
	waitGroup.Wait()
	return snapshots, nil
}

func (s *Service) sessionPivotSnapshots(sessionID string) []PivotListenerSnapshot {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	listeners, err := s.rpc.RPC.PivotSessionListeners(
		ctx,
		&sliverpb.PivotListenersReq{
			Request: &commonpb.Request{SessionID: sessionID},
		},
	)
	if err != nil || listeners.GetResponse().GetErr() != "" {
		return nil
	}

	snapshots := []PivotListenerSnapshot{}
	for _, listener := range listeners.Listeners {
		snapshot := PivotListenerSnapshot{
			ParentSessionID: sessionID,
			ID:              listener.ID,
			Type:            listener.Type.String(),
			BindAddress:     listener.BindAddress,
			Pivots:          []PivotConnectionSnapshot{},
		}
		for _, pivot := range listener.Pivots {
			snapshot.Pivots = append(snapshot.Pivots, PivotConnectionSnapshot{
				PeerID:        pivot.PeerID,
				RemoteAddress: pivot.RemoteAddress,
			})
		}
		snapshots = append(snapshots, snapshot)
	}
	return snapshots
}
