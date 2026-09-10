package console

import (
	"strings"
	"testing"

	"github.com/bishopfox/sliver/client/command"
	sliverconsole "github.com/bishopfox/sliver/client/console"

	"siren/internal/sliver/rpc"
)

// Sliver registers `exit` on the server menu and its handler ends the process
// via os.Exit. The GUI dispatches every typed line through that same cobra
// tree, so the guard must reject it before menu.RunCommandLine is reached.
func TestGuardRejectsSliverExitCommand(t *testing.T) {
	con := sliverconsole.NewConsole(false)
	root := command.ServerCommands(con, nil)()

	for _, line := range []string{"exit", "exit now"} {
		if reason := unsupportedConsoleCommand(root, line); reason == "" {
			t.Fatalf("unsupportedConsoleCommand(%q) allowed the command, want a rejection", line)
		}
	}
}

// RunLine is the path a modal, the raw console panel, and the automation engine
// all take. Running `exit` through it must return an error instead of ending the
// process.
func TestRunLineRejectsExit(t *testing.T) {
	// Sliver's console logger exits the process when its log directory is
	// unwritable, so pin the client root before init() opens it.
	t.Setenv("SLIVER_CLIENT_ROOT_DIR", t.TempDir())

	svc := New(rpc.NewClient())

	out, err := svc.RunLine("", "exit")
	if err == nil {
		t.Fatalf("RunLine(exit) = %q, want a rejection error", out)
	}
	if !strings.Contains(err.Error(), "terminate the GUI") {
		t.Fatalf("RunLine(exit) error = %v, want the exit rejection reason", err)
	}
}
