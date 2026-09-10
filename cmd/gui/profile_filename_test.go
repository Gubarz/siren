package gui

import (
	"strings"
	"testing"
)

// The profile name is attacker-influenced only in the sense that a teamserver
// operator picks it, but it still ends up as a file name next to a directory the
// user chose, so nothing resembling a path separator may survive.
func TestProfileFilenameIsSafeAsAFileName(t *testing.T) {
	names := []string{
		"operator@10.0.0.1 (a1b2c3d4)",
		"../../etc/passwd",
		`..\..\windows\system32\evil`,
		"  spaced  ",
		"",
	}
	for _, name := range names {
		got := profileFilename(name)
		if strings.ContainsAny(got, `/\`) {
			t.Fatalf("profileFilename(%q) = %q, still contains a path separator", name, got)
		}
		if strings.ContainsAny(got, " \t\n") {
			t.Fatalf("profileFilename(%q) = %q, still contains whitespace", name, got)
		}
		if !strings.HasSuffix(got, ".cfg") {
			t.Fatalf("profileFilename(%q) = %q, want a .cfg suffix", name, got)
		}
	}

	if got := profileFilename(""); got != "sliver-profile.cfg" {
		t.Fatalf("profileFilename(%q) = %q, want the fallback name", "", got)
	}
}
