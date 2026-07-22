package beacons

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"

	"sliver-gui/internal/sliver/rpc"
)

// A beacon's interactive session can be flipped on/off remotely. The server
// dispatches these through the beacon's next callback, so responses are async
// and take one beacon interval to materialize.

const lifecycleTimeout = 60 * time.Second

// GetBeacon returns the full record for a single beacon (fresher than the
// GetBeacons summary; carries the last-seen integrity level, config, etc.).
func (s *Service) GetBeacon(beaconID string) (*clientpb.Beacon, error) {
	if !s.rpc.Connected() {
		return nil, rpc.ErrNotConnected
	}
	if strings.TrimSpace(beaconID) == "" {
		return nil, fmt.Errorf("beacon ID is required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), lifecycleTimeout)
	defer cancel()
	return s.rpc.RPC.GetBeacon(ctx, &clientpb.Beacon{ID: beaconID})
}

// OpenSessionRequest is the GUI-facing shape of sliverpb.OpenSession —
// exposed as its own type so the wails binding stays JSON-friendly.
type OpenSessionRequest struct {
	BeaconID string   `json:"beaconId"`
	C2URLs   []string `json:"c2Urls"`
	Delay    int64    `json:"delay"`
}

// OpenSession promotes a beacon into an interactive session, tunneling
// through the C2 list supplied (or the beacon's existing C2 if empty).
func (s *Service) OpenSession(req OpenSessionRequest) (*sliverpb.OpenSession, error) {
	if !s.rpc.Connected() {
		return nil, rpc.ErrNotConnected
	}
	if strings.TrimSpace(req.BeaconID) == "" {
		return nil, fmt.Errorf("beacon ID is required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), lifecycleTimeout)
	defer cancel()
	return s.rpc.RPC.OpenSession(ctx, &sliverpb.OpenSession{
		C2S:   req.C2URLs,
		Delay: req.Delay,
		Request: &commonpb.Request{
			BeaconID: req.BeaconID,
			Async:    true,
			Timeout:  int64(lifecycleTimeout / time.Second),
		},
	})
}

// CloseSession tears down a beacon's interactive session, leaving the beacon
// itself intact (still polling).
func (s *Service) CloseSession(beaconID, tunnelID string) error {
	if !s.rpc.Connected() {
		return rpc.ErrNotConnected
	}
	if strings.TrimSpace(beaconID) == "" {
		return fmt.Errorf("beacon ID is required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), lifecycleTimeout)
	defer cancel()
	req := &sliverpb.CloseSession{
		Request: &commonpb.Request{
			BeaconID: beaconID,
			Async:    true,
			Timeout:  int64(lifecycleTimeout / time.Second),
		},
	}
	_, err := s.rpc.RPC.CloseSession(ctx, req)
	return err
}

// UpdateBeaconIntegrity records a fresh integrity classification for a
// beacon — used after we've observed a token-change (getsystem / make-token
// / runas) so the sessions table stays accurate without a full re-check.
func (s *Service) UpdateBeaconIntegrity(beaconID, integrity string) error {
	if !s.rpc.Connected() {
		return rpc.ErrNotConnected
	}
	if strings.TrimSpace(beaconID) == "" {
		return fmt.Errorf("beacon ID is required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), lifecycleTimeout)
	defer cancel()
	_, err := s.rpc.RPC.UpdateBeaconIntegrityInformation(ctx, &clientpb.BeaconIntegrity{
		BeaconID:  beaconID,
		Integrity: integrity,
	})
	return err
}
