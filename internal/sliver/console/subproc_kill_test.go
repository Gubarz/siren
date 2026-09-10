//go:build unix

package console

import (
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

// A console subprocess that ignores SIGTERM still has to go. watchConsole waits
// on it, and would otherwise block for good, never releasing the job
// registration or removing the temp config that holds the operator's key.
func TestKillEscalatesToSIGKILL(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "ready")
	cmd := exec.Command("sh", "-c", "trap '' TERM; : > '"+marker+"'; while :; do :; done")
	// The real child is a PTY session leader, so give this one its own group and
	// exercise the group signal rather than only the fallback.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	t.Cleanup(func() { _ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) })

	// Wait for the trap to be installed, or the SIGTERM below would land on the
	// default disposition and prove nothing.
	deadline := time.Now().Add(5 * time.Second)
	for {
		if _, err := os.Stat(marker); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("the shell never installed its TERM trap")
		}
		time.Sleep(10 * time.Millisecond)
	}

	proc := &unixProc{cmd: cmd}
	if err := proc.Kill(); err != nil {
		t.Fatalf("Kill() = %v, want nil", err)
	}

	done := make(chan error, 1)
	go func() { done <- proc.Wait() }()
	select {
	case <-done:
	case <-time.After(subprocKillGrace + 5*time.Second):
		t.Fatal("the child outlived the grace period; SIGKILL was never sent")
	}
}
