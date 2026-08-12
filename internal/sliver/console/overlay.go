package console

import (
	"context"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/google/uuid"

	"siren/internal/journal"
)

func withCommandOverlay(ctx context.Context, targetID, targetKind, hostname, line string) context.Context {
	overlay, _ := journal.OverlayFrom(ctx)
	if overlay.ActorKind == "" {
		overlay.ActorKind = "operator"
	}
	if overlay.Panel == "" {
		overlay.Panel = "console"
	}
	if overlay.CommandLine == "" {
		overlay.CommandLine = line
	}
	if overlay.CorrelationID == "" {
		overlay.CorrelationID = uuid.NewString()
	}
	overlay.TargetID = targetID
	if overlay.TargetKind == "" {
		overlay.TargetKind = targetKind
	}
	if overlay.Hostname == "" {
		overlay.Hostname = hostname
	}
	return journal.WithContext(ctx, overlay)
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
