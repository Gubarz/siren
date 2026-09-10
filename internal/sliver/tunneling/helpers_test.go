package tunneling

import (
	"strings"
	"testing"
)

type startArgs struct {
	sessionID  string
	bindAddr   string
	remoteAddr string
}

// captureStart records the arguments passed to a proxy start function.
func captureStart(id uint64) (func(string, string, string) (uint64, error), *startArgs) {
	got := &startArgs{}
	start := func(sessionID, bindAddr, remoteAddr string) (uint64, error) {
		got.sessionID = sessionID
		got.bindAddr = bindAddr
		got.remoteAddr = remoteAddr
		return id, nil
	}
	return start, got
}

type consoleAddFunc func(func(string, string, string) (uint64, error), string, []string) (string, error)

// assertConsoleProxyAdd drives a console "add" handler and asserts the start
// arguments plus the rendered output.
func assertConsoleProxyAdd(t *testing.T, add consoleAddFunc, id uint64, args []string, wantBind, wantRemote, wantOutput string) {
	t.Helper()
	start, got := captureStart(id)
	output, err := add(start, "session-1", args)
	if err != nil {
		t.Fatal(err)
	}
	if got.sessionID != "session-1" || got.bindAddr != wantBind || got.remoteAddr != wantRemote {
		t.Fatalf("start args = (%q, %q, %q)", got.sessionID, got.bindAddr, got.remoteAddr)
	}
	if !strings.Contains(output, wantOutput) {
		t.Fatalf("output = %q, want %q", output, wantOutput)
	}
}
