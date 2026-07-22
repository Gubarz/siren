package agents

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"

	"sliver-gui/internal/sliver/rpc"
)

const reconfigureTimeout = 60 * time.Second

// ReconfigureRequest is the GUI-visible superset of sliverpb.ReconfigureReq.
// All fields are optional — zero values fall through to the implant's
// current setting server-side.
type ReconfigureRequest struct {
	AgentID           string `json:"agentId"`
	ReconnectInterval int64  `json:"reconnectInterval,omitempty"`
	BeaconInterval    int64  `json:"beaconInterval,omitempty"`
	BeaconJitter      int64  `json:"beaconJitter,omitempty"`
}

// Reconfigure asks the running implant to update its poll cadence live
// without forcing a full re-implant / rebuild.
func (s *Service) Reconfigure(req ReconfigureRequest) error {
	if !s.rpc.Connected() {
		return rpc.ErrNotConnected
	}
	if strings.TrimSpace(req.AgentID) == "" {
		return fmt.Errorf("agent ID is required")
	}
	request, err := s.buildReconfigureRequest(req.AgentID)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), reconfigureTimeout)
	defer cancel()
	_, err = s.rpc.RPC.Reconfigure(ctx, &sliverpb.ReconfigureReq{
		ReconnectInterval: req.ReconnectInterval,
		BeaconInterval:    req.BeaconInterval,
		BeaconJitter:      req.BeaconJitter,
		Request:           request,
	})
	return err
}

// buildReconfigureRequest routes to session or beacon (async for beacons)
// based on which side of the fence the target lives on.
func (s *Service) buildReconfigureRequest(agentID string) (*commonpb.Request, error) {
	sess, beacon, err := s.console.FindTarget(agentID)
	if err != nil {
		return nil, err
	}
	req := &commonpb.Request{
		Timeout: int64(reconfigureTimeout / time.Second),
	}
	if sess != nil {
		req.SessionID = sess.ID
		return req, nil
	}
	if beacon != nil {
		req.BeaconID = beacon.ID
		req.Async = true
		return req, nil
	}
	return nil, fmt.Errorf("agent %s not found", agentID)
}
