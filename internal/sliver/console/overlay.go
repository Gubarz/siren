package console

import (
	"context"

	"github.com/bishopfox/sliver/protobuf/clientpb"

	"siren/internal/execctx"
)

func withCommandOverlay(ctx context.Context, targetID, targetKind, hostname, _ string) context.Context {
	return execctx.WithTarget(ctx, targetID, targetKind, hostname)
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
