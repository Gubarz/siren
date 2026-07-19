//go:build unix

package console

import (
	"encoding/base64"
	"testing"

	"github.com/spf13/cobra"
)

func TestMatchesPinnedTarget(t *testing.T) {
	const sessionID = "abcdef1234567890"
	cases := []struct {
		name      string
		candidate string
		want      bool
	}{
		{name: "full id", candidate: sessionID, want: true},
		{name: "short id", candidate: "abcdef12", want: true},
		{name: "too short", candidate: "abcdef1", want: false},
		{name: "other id", candidate: "12345678", want: false},
		{name: "empty", candidate: "", want: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := matchesPinnedTarget(tc.candidate, sessionID); got != tc.want {
				t.Fatalf("matchesPinnedTarget(%q, %q) = %v, want %v", tc.candidate, sessionID, got, tc.want)
			}
		})
	}
}

func TestShellCommandTail(t *testing.T) {
	cmd := &cobra.Command{Use: "shell"}
	cmd.Flags().StringP("shell-path", "s", "", "")
	if err := cmd.Flags().Set("shell-path", "/bin/bash"); err != nil {
		t.Fatal(err)
	}

	if got := shellCommandTail(cmd, nil); got != "/bin/bash" {
		t.Fatalf("shellCommandTail with flag = %q, want /bin/bash", got)
	}
	if got := shellCommandTail(&cobra.Command{Use: "shell"}, []string{"ignored"}); got != "ignored" {
		t.Fatalf("shellCommandTail with args = %q, want ignored", got)
	}
}

func TestFilterShellOpenFrames(t *testing.T) {
	payload := base64.StdEncoding.EncodeToString([]byte("/bin/bash"))
	frame := []byte(shellOpenFramePrefix + payload + shellOpenFrameSuffix)

	visible, tails, carry := filterShellOpenFrames(nil, append([]byte("before"), append(frame, []byte("after")...)...))
	if string(visible) != "beforeafter" {
		t.Fatalf("visible = %q, want beforeafter", visible)
	}
	if len(tails) != 1 || tails[0] != "/bin/bash" {
		t.Fatalf("tails = %#v, want /bin/bash", tails)
	}
	if len(carry) != 0 {
		t.Fatalf("carry = %q, want empty", carry)
	}
}

func TestFilterShellOpenFramesAcrossChunks(t *testing.T) {
	payload := base64.StdEncoding.EncodeToString([]byte("pwsh"))
	frame := []byte(shellOpenFramePrefix + payload + shellOpenFrameSuffix)

	visible, tails, carry := filterShellOpenFrames(nil, append([]byte("before"), frame[:10]...))
	if string(visible) != "before" || len(tails) != 0 || len(carry) == 0 {
		t.Fatalf("first chunk visible=%q tails=%#v carry=%q", visible, tails, carry)
	}

	visible, tails, carry = filterShellOpenFrames(carry, append(frame[10:], []byte("after")...))
	if string(visible) != "after" {
		t.Fatalf("second chunk visible = %q, want after", visible)
	}
	if len(tails) != 1 || tails[0] != "pwsh" {
		t.Fatalf("second chunk tails = %#v, want pwsh", tails)
	}
	if len(carry) != 0 {
		t.Fatalf("second chunk carry = %q, want empty", carry)
	}
}
