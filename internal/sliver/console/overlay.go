package console

import (
	"context"

	"github.com/bishopfox/sliver/protobuf/clientpb"

	"siren/internal/execctx"
)

// withCommandOverlay installs target and run identity for one command
// execution. The returned restore must run when the command returns; command
// execution is serialized under the console mutex, so one current slot is
// enough.
func withCommandOverlay(ctx context.Context, targetID, targetKind, hostname, _ string) (context.Context, func()) {
	runID, stageID := execctx.Run(ctx)
	restore := execctx.SetCurrent(execctx.Snapshot{
		RunID: runID, StageID: stageID,
		TargetID: targetID, TargetKind: targetKind, Hostname: hostname,
	})
	return execctx.WithTarget(ctx, targetID, targetKind, hostname), restore
}

func targetKindOf(sess *clientpb.Session, beacon *clientpb.Beacon) string {
	if sess != nil {
		return "session"
	}
	if beacon != nil {
		return "beacon"
	}
	return ""
}

func hostnameOf(sess *clientpb.Session, beacon *clientpb.Beacon) string {
	if sess != nil {
		return sess.Hostname
	}
	if beacon != nil {
		return beacon.Hostname
	}
	return ""
}
