// Package execctx carries run, stage, and target identity through command
// execution so recording and evidence writes can attribute work without
// importing automation or console packages.
package execctx

import "context"

type runKey struct{}

type runInfo struct {
	runID   string
	stageID string
}

func WithRun(ctx context.Context, runID, stageID string) context.Context {
	return context.WithValue(ctx, runKey{}, runInfo{runID: runID, stageID: stageID})
}

func Run(ctx context.Context) (string, string) {
	info, _ := ctx.Value(runKey{}).(runInfo)
	return info.runID, info.stageID
}

type targetKey struct{}

type targetInfo struct {
	id, kind, hostname string
}

func WithTarget(ctx context.Context, targetID, targetKind, hostname string) context.Context {
	return context.WithValue(ctx, targetKey{}, targetInfo{id: targetID, kind: targetKind, hostname: hostname})
}

func Target(ctx context.Context) (string, string, string) {
	info, _ := ctx.Value(targetKey{}).(targetInfo)
	return info.id, info.kind, info.hostname
}
