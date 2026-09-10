package console

import (
	"testing"

	"siren/internal/sliver/rpc"
)

// Routed-command frames are read off the same stream as implant output, so any
// bytes matching the frame prefix reach this handler. Only the tunnel commands
// may be routed through the parent's in-process console; anything else has to
// stay in the subprocess.
func TestRoutedJobCommandOnlyAcceptsTunnelCommandsForARealSession(t *testing.T) {
	svc := New(rpc.NewClient())
	var seen []string
	svc.SetRoutedCommandHandler(func(sessionID, line string) RoutedCommandResult {
		seen = append(seen, sessionID+"|"+line)
		return RoutedCommandResult{Handled: true}
	})

	job := &subprocJob{id: "job-1", sessionID: "session-1"}
	svc.handleRoutedConsoleJobCommand(job, "socks5 start --host 127.0.0.1 --port 1080")
	for _, line := range []string{"rm -rf /", "exit", "shell attach", "use abc"} {
		svc.handleRoutedConsoleJobCommand(job, line)
	}
	// A job with no session has nothing to route against.
	svc.handleRoutedConsoleJobCommand(&subprocJob{id: "job-2"}, "socks5")

	if len(seen) != 1 {
		t.Fatalf("routed %v, want only the tunnel command from a job with a session", seen)
	}
}
