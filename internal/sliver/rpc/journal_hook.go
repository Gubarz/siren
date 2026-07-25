package rpc

//go:generate go run ./gen

import (
	"context"
	"sync/atomic"

	"sliver-gui/internal/journal"
)

// JournalHook is the hand-written half of the journal decorator: the
// generated wrapper calls record() after every RPC. Nil-safe everywhere —
// capture must never break operations.
type JournalHook struct {
	journal *journal.Service
	connID  atomic.Value
}

func NewJournalHook(j *journal.Service) *JournalHook {
	h := &JournalHook{journal: j}
	h.connID.Store("")
	return h
}

func (h *JournalHook) SetConnection(id string) {
	if h == nil {
		return
	}
	h.connID.Store(id)
}

func (h *JournalHook) connIDStr() string {
	if h == nil {
		return ""
	}
	s, _ := h.connID.Load().(string)
	return s
}

// record is called from the generated decorator on the RPC hot path. It is
// enqueue-only and never surfaces errors to the RPC caller.
func (h *JournalHook) record(ctx context.Context, verb string, durationMs int64, callErr error) {
	if h == nil || h.journal == nil {
		return
	}
	if journal.ClassifyVerb(verb) == journal.VerbDrop {
		return
	}
	e := journal.Entry{
		Verb:         verb,
		ConnectionID: h.connIDStr(),
		DurationMs:   durationMs,
		Status:       "ok",
	}
	if callErr != nil {
		e.Status = "error"
		e.Err = callErr.Error()
	}
	if overlay, ok := journal.OverlayFrom(ctx); ok {
		e.ApplyOverlay(overlay)
	} else {
		e.ApplyOverlay(journal.Overlay{}) // defaults ActorKind
	}
	h.journal.Record(e)
}
