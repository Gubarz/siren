package console

import (
	"strings"
	"testing"

	"siren/internal/sliver/rpc"
)

func TestValidSessionID(t *testing.T) {
	valid := []string{
		"",
		"a1b2c3d4-e5f6-7890-abcd-ef1234567890",
		"ABCDEF12",
	}
	for _, id := range valid {
		if !validSessionID(id) {
			t.Fatalf("validSessionID(%q) = false, want true", id)
		}
	}

	invalid := []string{
		"abc", // shorter than a Sliver id
		"a1b2c3d4\nuse other-session",
		"a1b2c3d4 extra",
		"a1b2c3d4;rm -rf /",
		"a1b2c3d4\x00",
		strings.Repeat("a", 65),
	}
	for _, id := range invalid {
		if validSessionID(id) {
			t.Fatalf("validSessionID(%q) = true, want false", id)
		}
	}
}

// The id is interpolated into a line-oriented rcScript, so an id carrying a
// newline would be run as a second console command in the subprocess.
func TestMalformedSessionIDIsRejectedBeforeItReachesTheScript(t *testing.T) {
	bad := "a1b2c3d4\nuse other-session"

	if _, _, err := New(rpc.NewClient()).AcquireConsole(bad); err == nil || !strings.Contains(err.Error(), "malformed session id") {
		t.Fatalf("AcquireConsole(bad id) = %v, want a malformed session id rejection", err)
	}
	if err := RunConsoleSubprocess("unused.cfg", bad); err == nil || !strings.Contains(err.Error(), "malformed session id") {
		t.Fatalf("RunConsoleSubprocess(bad id) = %v, want a malformed session id rejection", err)
	}
}
