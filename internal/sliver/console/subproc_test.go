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

func TestPortfwdCommandLine(t *testing.T) {
	root := &cobra.Command{Use: "sliver"}
	portfwd := &cobra.Command{Use: "portfwd"}
	add := &cobra.Command{Use: "add"}
	remove := &cobra.Command{Use: "rm"}
	add.Flags().StringP("bind", "b", "127.0.0.1:8080", "")
	add.Flags().StringP("remote", "r", "", "")
	remove.Flags().IntP("id", "i", 0, "")
	root.AddCommand(portfwd)
	portfwd.AddCommand(add, remove)
	if err := add.Flags().Set("bind", "127.0.0.1:9000"); err != nil {
		t.Fatal(err)
	}
	if err := add.Flags().Set("remote", "10.0.0.5:80"); err != nil {
		t.Fatal(err)
	}

	got := portfwdCommandLine(add, nil)
	want := "portfwd add --bind 127.0.0.1:9000 --remote 10.0.0.5:80"
	if got != want {
		t.Fatalf("portfwdCommandLine() = %q, want %q", got, want)
	}
	if err := remove.Flags().Set("id", "23"); err != nil {
		t.Fatal(err)
	}
	if got := portfwdCommandLine(remove, nil); got != "portfwd rm --id 23" {
		t.Fatalf("portfwdCommandLine(rm) = %q, want portfwd rm --id 23", got)
	}
}

func TestIsRoutedConsoleCommandIncludesTunnelCommands(t *testing.T) {
	for _, line := range []string{"socks5", "socks5 start", "portfwd", "portfwd add --remote 10.0.0.5:80", " portfwd rm --id 1 ", "rportfwd", "rportfwd add --bind :4444 --remote :8080"} {
		if !isRoutedConsoleCommand(line) {
			t.Fatalf("isRoutedConsoleCommand(%q) = false", line)
		}
	}
	for _, line := range []string{"portfwds", "rportfwds", "echo portfwd"} {
		if isRoutedConsoleCommand(line) {
			t.Fatalf("isRoutedConsoleCommand(%q) = true", line)
		}
	}
}

func TestRportfwdCommandLine(t *testing.T) {
	root := &cobra.Command{Use: "sliver"}
	rportfwd := &cobra.Command{Use: "rportfwd"}
	add := &cobra.Command{Use: "add"}
	remove := &cobra.Command{Use: "rm"}
	add.Flags().StringP("bind", "b", "", "")
	add.Flags().StringP("remote", "r", "", "")
	remove.Flags().Uint32P("id", "i", 0, "")
	root.AddCommand(rportfwd)
	rportfwd.AddCommand(add, remove)
	if err := add.Flags().Set("bind", "0.0.0.0:4444"); err != nil {
		t.Fatal(err)
	}
	if err := add.Flags().Set("remote", "127.0.0.1:8080"); err != nil {
		t.Fatal(err)
	}
	if got := rportfwdCommandLine(add, nil); got != "rportfwd add --bind 0.0.0.0:4444 --remote 127.0.0.1:8080" {
		t.Fatalf("rportfwdCommandLine(add) = %q", got)
	}
	if err := remove.Flags().Set("id", "31"); err != nil {
		t.Fatal(err)
	}
	if got := rportfwdCommandLine(remove, nil); got != "rportfwd rm --id 31" {
		t.Fatalf("rportfwdCommandLine(rm) = %q", got)
	}
}

func TestWriteConsoleRoutesSocksBeforeSubmittingToSubprocess(t *testing.T) {
	svc := New(nil)
	emitter := &recordingEmitter{}
	svc.SetEmitter(emitter)
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
	if len(emitter.names) != 2 || emitter.names[0] != "console-output" || emitter.names[1] != "tunnels-changed" {
		t.Fatalf("emitted events = %#v, want console-output then tunnels-changed", emitter.names)
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

type recordingEmitter struct {
	names []string
}

func TestSubprocSessionLeasesShareOneJob(t *testing.T) {
	mgr := &subprocMgr{}
	job := &subprocJob{id: "job-1", sessionID: "session-1"}
	mgr.add(job)

	if got := mgr.acquireSession("session-1"); got != "job-1" {
		t.Fatalf("acquireSession() = %q, want job-1", got)
	}
	if _, shouldStop := mgr.release("job-1"); shouldStop {
		t.Fatal("first release stopped a console with another window lease")
	}
	if _, shouldStop := mgr.release("job-1"); !shouldStop {
		t.Fatal("final release did not stop the console")
	}
	if _, shouldStop := mgr.release("job-1"); shouldStop {
		t.Fatal("repeated release tried to stop the same console again")
	}
}

func TestAcquireConsoleReportsExistingJob(t *testing.T) {
	svc := New(nil)
	svc.subproc.add(&subprocJob{id: "job-1", sessionID: "session-1"})

	id, existing, err := svc.AcquireConsole("session-1")
	if err != nil {
		t.Fatal(err)
	}
	if id != "job-1" || !existing {
		t.Fatalf("AcquireConsole() = %q, %v; want job-1, true", id, existing)
	}
}

func TestSubprocOutputSnapshotReplaysAndBoundsHistory(t *testing.T) {
	job := &subprocJob{}
	job.appendOutput([]byte("first"))
	job.appendOutput([]byte(" second"))
	if got := string(job.outputSnapshot()); got != "first second" {
		t.Fatalf("outputSnapshot() = %q, want first second", got)
	}

	large := bytes.Repeat([]byte{'x'}, consoleOutputReplayMaxBytes+10)
	job.appendOutput(large)
	if got := len(job.outputSnapshot()); got != consoleOutputReplayMaxBytes {
		t.Fatalf("bounded snapshot length = %d, want %d", got, consoleOutputReplayMaxBytes)
	}
}

func (e *recordingEmitter) Emit(name string, _ any) {
	e.names = append(e.names, name)
}

func (p *recordingPTY) Read([]byte) (int, error) {
	return 0, io.EOF
}

func (p *recordingPTY) Close() error {
	return nil
}
