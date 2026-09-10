// Package execctx carries run, stage, and target identity through command
// execution so recording and evidence writes can attribute work without
// importing automation or console packages.
package execctx

import (
	"context"
	"sync/atomic"
)

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

// Snapshot is the execution identity visible to code that runs with a
// background context and therefore cannot read the caller's context values.
type Snapshot struct {
	RunID, StageID, TargetID, TargetKind, Hostname string
}

var current atomic.Pointer[Snapshot]

// SetCurrent installs the execution identity for commands that run with
// background contexts and returns a restore function. Callers must invoke the
// restore; console command execution is serialized, so one slot is enough.
func SetCurrent(s Snapshot) func() {
	prev := current.Swap(&s)
	return func() { current.Store(prev) }
}

func Current() Snapshot {
	if p := current.Load(); p != nil {
		return *p
	}
	return Snapshot{}
}
