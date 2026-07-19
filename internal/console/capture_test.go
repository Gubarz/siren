package console

import (
	"strings"
	"testing"

	sliverconsole "github.com/bishopfox/sliver/client/console"
	"github.com/spf13/cobra"

	"sliver-gui/internal/rpc"
)

func TestOutputSinkCapturesSliverPrintf(t *testing.T) {
	con := sliverconsole.NewConsole(false)
	var sink outputSink

	installOutputSink(con, &sink)
	out, err := sink.capture(func() error {
		con.PrintInfof("hello %s\n", "world")
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "hello world") {
		t.Fatalf("captured output %q does not include Sliver print", out)
	}
}

func TestOutputSinkCapturesCobraHelp(t *testing.T) {
	root := &cobra.Command{Use: "root", Short: "root command"}
	root.AddCommand(&cobra.Command{Use: "child", Short: "child command"})

	var sink outputSink
	routeCommandOutput(root, &sink)

	out, err := sink.capture(func() error {
		root.SetArgs([]string{"help"})
		return root.Execute()
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Available Commands") || !strings.Contains(out, "child") {
		t.Fatalf("captured help output %q does not include Cobra help", out)
	}
}

func TestRunLineCapturesHelp(t *testing.T) {
	svc := New(rpc.NewClient())

	out, err := svc.RunLine("", "help")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Usage:") || !strings.Contains(out, "help for this command") {
		t.Fatalf("captured help output %q does not look like help", out)
	}
}

func TestTerminalOnlyCommandReasons(t *testing.T) {
	for _, path := range []string{"help", "kill", "generate", "websites add-content", "cat"} {
		if reason := NonInteractiveCommandReason(path); reason != "" {
			t.Fatalf("%s should be allowed, got reason %q", path, reason)
		}
	}

	for _, path := range []string{"shell attach", "edit", "hexedit", "docs", "switch", "ai"} {
		if reason := NonInteractiveCommandReason(path); reason == "" {
			t.Fatalf("%s should be rejected in the GUI console", path)
		}
	}
}
