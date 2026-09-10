// Package testutil holds polling and assertion helpers shared by test suites.
package testutil

import (
	"errors"
	"testing"
	"time"
)

const (
	waitTimeout = 3 * time.Second
	waitPoll    = 10 * time.Millisecond
)

// WaitFor polls cond until it returns true or waitTimeout elapses.
func WaitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(waitTimeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(waitPoll)
	}
	t.Fatalf("timed out waiting for %s", what)
}

// RequireUncalled fails unless err wraps target and calls is zero, i.e. the
// operation failed before issuing its RPC.
func RequireUncalled(t *testing.T, err, target error, calls int, what string) {
	t.Helper()
	if !errors.Is(err, target) {
		t.Fatalf("%s error = %v, want %v", what, err, target)
	}
	if calls != 0 {
		t.Fatalf("%s issued %d calls despite %v", what, calls, target)
	}
}

// AssertSlice fails unless got and want have the same length and elements.
func AssertSlice[T comparable](t *testing.T, got, want []T) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("len(%v) = %d, want %d", got, len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("slice[%d] = %v, want %v (full slice %v)", i, got[i], want[i], got)
		}
	}
}
