package automationexec

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/gubarz/revils/store"

	"siren/internal/execctx"
)

func TestWriteEvidenceVerifiedAndFailed(t *testing.T) {
	key := make([]byte, 32)
	st, err := store.Open(store.Config{DBPath: filepath.Join(t.TempDir(), "ev.sqlite"), Key: key})
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	defer st.Close()
	ctx := execctx.WithRun(context.Background(), "run-7", "commands#0")
	writeEvidence(st, ctx, "sess-1", "session", "out", nil)
	writeEvidence(st, ctx, "sess-1", "session", "", errors.New("boom"))
	rows, err := st.Query(store.Filter{Kind: store.KindEvidence, RunID: "run-7", Asc: true, Limit: 10})
	if err != nil || len(rows) != 2 {
		t.Fatalf("evidence: %v %+v", err, rows)
	}
	if rows[0].Status != store.StatusVerified || rows[1].Status != store.StatusFailed {
		t.Fatalf("statuses: %s/%s", rows[0].Status, rows[1].Status)
	}
	if rows[0].StageID != "commands#0" || rows[0].SessionID != "sess-1" {
		t.Fatalf("row: %+v", rows[0])
	}
}
