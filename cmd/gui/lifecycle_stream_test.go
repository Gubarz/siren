package gui

import (
	"testing"

	"siren/internal/bootstrap"
	"siren/internal/sliver/console"
	"siren/internal/sliver/rpc"
)

// The in-process console has to be closed when the event stream dies, because
// that is what disarms Sliver's connection watcher before it calls os.Exit.
func TestResetConsoleAfterStreamClose(t *testing.T) {
	if (&App{SharedStack: &bootstrap.SharedStack{}}).resetConsoleAfterStreamClose() {
		t.Fatal("reported a reset with no console service")
	}

	a := &App{SharedStack: &bootstrap.SharedStack{Console: console.New(rpc.NewClient())}}
	if !a.resetConsoleAfterStreamClose() {
		t.Fatal("did not reset the console after the stream closed")
	}
}
