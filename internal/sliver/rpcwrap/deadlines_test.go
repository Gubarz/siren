package rpcwrap

import (
	"context"
	"testing"
	"time"
)

// Every RPC entry point here goes through bounded, so that a half-open
// connection cannot leave a Wails binding blocked forever. A caller that already
// set a deadline keeps it.
func TestBoundedAppliesTheDefaultOnlyWithoutACallerDeadline(t *testing.T) {
	ctx, cancel := bounded(context.Background())
	defer cancel()

	deadline, ok := ctx.Deadline()
	if !ok {
		t.Fatal("bounded() left a context with no deadline")
	}
	if remaining := time.Until(deadline); remaining <= 0 || remaining > defaultTimeout {
		t.Fatalf("deadline is %v away, want within %v", remaining, defaultTimeout)
	}

	parent, parentCancel := context.WithTimeout(context.Background(), time.Minute)
	defer parentCancel()

	got, gotCancel := bounded(parent)
	defer gotCancel()

	gotDeadline, _ := got.Deadline()
	wantDeadline, _ := parent.Deadline()
	if !gotDeadline.Equal(wantDeadline) {
		t.Fatalf("bounded() replaced the caller deadline %v with %v", wantDeadline, gotDeadline)
	}
}
