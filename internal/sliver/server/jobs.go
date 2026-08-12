package server

import (
	"context"

	"github.com/bishopfox/sliver/protobuf/clientpb"

	"siren/internal/sliver/rpc"
)

// RestartJobs asks the teamserver to tear down and rebuild the listener/job
// entries by ID. Empty jobIDs means "everything" — mirrors sliver's console
// `jobs -k` + auto-restart pattern.
func (s *Service) RestartJobs(jobIDs []uint32) error {
	if !s.rpc.Connected() {
		return rpc.ErrNotConnected
	}
	_, err := s.rpc.RestartJobs(context.Background(), &clientpb.RestartJobReq{JobIDs: jobIDs})
	return err
}
