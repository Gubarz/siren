package console

import (
	"bytes"
	"encoding/base64"
	"io"
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

func TestFilterConsoleCommandFrames(t *testing.T) {
	payload := base64.StdEncoding.EncodeToString([]byte("socks5 start --host 127.0.0.1 --port 1081"))
	frame := []byte(consoleCommandFramePrefix + payload + controlFrameSuffix)

	visible, tails, commands, carry := filterConsoleControlFrames(nil, append([]byte("before"), append(frame, []byte("after")...)...))
	if string(visible) != "beforeafter" {
		t.Fatalf("visible = %q, want beforeafter", visible)
	}
	if len(tails) != 0 {
		t.Fatalf("tails = %#v, want none", tails)
	}
	if len(commands) != 1 || commands[0] != "socks5 start --host 127.0.0.1 --port 1081" {
		t.Fatalf("commands = %#v, want socks5 start command", commands)
	}
	if len(carry) != 0 {
		t.Fatalf("carry = %q, want empty", carry)
	}
}

func TestWriteConsoleRoutesSocksBeforeSubmittingToSubprocess(t *testing.T) {
	svc := New(nil)
	pty := &recordingPTY{}
	svc.subproc.add(&subprocJob{id: "job-1", sessionID: "session-1", pty: pty})

	var routedLine string
	svc.SetRoutedCommandHandler(func(sessionID, line string) RoutedCommandResult {
		if sessionID != "session-1" {
			t.Fatalf("sessionID = %q, want session-1", sessionID)
		}
		routedLine = line
		return RoutedCommandResult{Handled: true, Output: "[*] Started SOCKS5\n", Refresh: true}
	})

	if err := svc.WriteConsole("job-1", []byte("socks5 start")); err != nil {
		t.Fatal(err)
	}
	if err := svc.WriteConsole("job-1", []byte("\r")); err != nil {
		t.Fatal(err)
	}

	if routedLine != "socks5 start" {
		t.Fatalf("routed line = %q, want socks5 start", routedLine)
	}
	got := pty.String()
	if !bytes.Contains([]byte(got), []byte("socks5 start")) {
		t.Fatalf("pty writes = %q, want typed command echoed to child before submit", got)
	}
	if bytes.Contains([]byte(got), []byte("socks5 start\r")) {
		t.Fatalf("pty writes = %q, command should not be submitted to child", got)
	}
	if !bytes.Contains([]byte(got), []byte("\x05\x15")) {
		t.Fatalf("pty writes = %q, want readline clear sequence", got)
	}
}

func TestSendToSessionConsoleRoutesSocksWithoutWritingSubprocess(t *testing.T) {
	svc := New(nil)
	pty := &recordingPTY{}
	svc.subproc.add(&subprocJob{id: "job-1", sessionID: "session-1", pty: pty})

	var routedLine string
	svc.SetRoutedCommandHandler(func(_, line string) RoutedCommandResult {
		routedLine = line
		return RoutedCommandResult{Handled: true, Output: "[*] Started SOCKS5\n", Refresh: true}
	})

	if err := svc.SendToSessionConsole("session-1", "socks5 start --port 1080"); err != nil {
		t.Fatal(err)
	}
	if routedLine != "socks5 start --port 1080" {
		t.Fatalf("routed line = %q, want socks5 start --port 1080", routedLine)
	}
	if got := pty.String(); got != "" {
		t.Fatalf("pty writes = %q, want no subprocess input", got)
	}
}

type recordingPTY struct {
	bytes.Buffer
}

func (p *recordingPTY) Read([]byte) (int, error) {
	return 0, io.EOF
}

func (p *recordingPTY) Close() error {
	return nil
}
