package console

import (
	"bytes"
	"encoding/base64"
	"os"
	"strings"

	sliverconsole "github.com/bishopfox/sliver/client/console"
	reefconsole "github.com/reeflective/console"
	"github.com/spf13/cobra"
)

const (
	shellOpenFramePrefix = "\x1b]777;sliver-gui-open-shell="
	shellOpenFrameSuffix = "\x07"
)

func pinServerTargetCommands(base reefconsole.Commands, sessionID string, con *sliverconsole.SliverClient) reefconsole.Commands {
	return func() *cobra.Command {
		root := base()
		if useCmd := findTopLevelCommand(root, "use"); useCmd != nil {
			pinUseCommand(useCmd, sessionID, con)
		}
		if sessionsCmd := findTopLevelCommand(root, "sessions"); sessionsCmd != nil {
			pinSessionsCommand(sessionsCmd, sessionID, con)
		}
		return root
	}
}

func pinSliverTargetCommands(base reefconsole.Commands, sessionID string, con *sliverconsole.SliverClient) reefconsole.Commands {
	return func() *cobra.Command {
		root := base()
		if backgroundCmd := findTopLevelCommand(root, "background"); backgroundCmd != nil {
			wrapCommandRun(backgroundCmd, func(*cobra.Command, []string) bool {
				return false
			}, func() {
				printPinnedConsoleMessage(con, sessionID)
			})
		}
		if shellCmd := findTopLevelCommand(root, "shell"); shellCmd != nil {
			wrapCommandRun(shellCmd, func(command *cobra.Command, args []string) bool {
				emitShellOpenFrame(shellCommandTail(command, args))
				return false
			}, func() {})
		}
		return root
	}
}

func pinUseCommand(cmd *cobra.Command, sessionID string, con *sliverconsole.SliverClient) {
	wrapCommandRun(cmd, func(command *cobra.Command, args []string) bool {
		return command.Name() == "use" && len(args) == 1 && matchesPinnedTarget(args[0], sessionID)
	}, func() {
		printPinnedConsoleMessage(con, sessionID)
	})
	for _, child := range cmd.Commands() {
		pinUseSubcommands(child, sessionID, con)
	}
}

func pinUseSubcommands(cmd *cobra.Command, sessionID string, con *sliverconsole.SliverClient) {
	wrapCommandRun(cmd, func(*cobra.Command, []string) bool {
		return false
	}, func() {
		printPinnedConsoleMessage(con, sessionID)
	})
	for _, child := range cmd.Commands() {
		pinUseSubcommands(child, sessionID, con)
	}
}

func pinSessionsCommand(cmd *cobra.Command, sessionID string, con *sliverconsole.SliverClient) {
	wrapCommandRun(cmd, func(command *cobra.Command, _ []string) bool {
		interact, _ := command.Flags().GetString("interact")
		return interact == "" || matchesPinnedTarget(interact, sessionID)
	}, func() {
		printPinnedConsoleMessage(con, sessionID)
	})
}

func wrapCommandRun(cmd *cobra.Command, allow func(*cobra.Command, []string) bool, reject func()) {
	originalRun := cmd.Run
	originalRunE := cmd.RunE
	cmd.Run = func(command *cobra.Command, args []string) {
		if !allow(command, args) {
			reject()
			return
		}
		if originalRun != nil {
			originalRun(command, args)
			return
		}
		if originalRunE != nil {
			if err := originalRunE(command, args); err != nil {
				command.PrintErrln(err)
			}
		}
	}
	cmd.RunE = nil
}

func findTopLevelCommand(root *cobra.Command, name string) *cobra.Command {
	if root == nil {
		return nil
	}
	for _, command := range root.Commands() {
		if command.Name() == name {
			return command
		}
	}
	return nil
}

func matchesPinnedTarget(candidate, sessionID string) bool {
	candidate = strings.TrimSpace(candidate)
	if candidate == "" {
		return false
	}
	return candidate == sessionID || (len(candidate) >= 8 && strings.HasPrefix(sessionID, candidate))
}

func printPinnedConsoleMessage(con *sliverconsole.SliverClient, sessionID string) {
	shortID := sessionID
	if len(shortID) > 8 {
		shortID = shortID[:8]
	}
	con.PrintErrorf("This embedded console is pinned to %s. Open another agent's Console tab to switch targets.\n", shortID)
}

func shellCommandTail(cmd *cobra.Command, args []string) string {
	if cmd != nil {
		if shellPath, _ := cmd.Flags().GetString("shell-path"); shellPath != "" {
			return shellPath
		}
	}
	return strings.Join(args, " ")
}

func emitShellOpenFrame(tail string) {
	payload := base64.StdEncoding.EncodeToString([]byte(tail))
	_, _ = os.Stdout.WriteString(shellOpenFramePrefix + payload + shellOpenFrameSuffix)
}

func filterShellOpenFrames(carry, chunk []byte) ([]byte, []string, []byte) {
	buf := make([]byte, 0, len(carry)+len(chunk))
	buf = append(buf, carry...)
	buf = append(buf, chunk...)

	prefix := []byte(shellOpenFramePrefix)
	suffix := []byte(shellOpenFrameSuffix)
	visible := make([]byte, 0, len(buf))
	var tails []string

	for len(buf) > 0 {
		idx := bytes.Index(buf, prefix)
		if idx < 0 {
			keep := prefixSuffixLen(buf, prefix)
			visible = append(visible, buf[:len(buf)-keep]...)
			nextCarry := append([]byte(nil), buf[len(buf)-keep:]...)
			return visible, tails, nextCarry
		}

		visible = append(visible, buf[:idx]...)
		rest := buf[idx+len(prefix):]
		end := bytes.Index(rest, suffix)
		if end < 0 {
			nextCarry := append([]byte(nil), buf[idx:]...)
			return visible, tails, nextCarry
		}

		payload := rest[:end]
		if decoded, err := base64.StdEncoding.DecodeString(string(payload)); err == nil {
			tails = append(tails, string(decoded))
		}
		buf = rest[end+len(suffix):]
	}

	return visible, tails, nil
}

func prefixSuffixLen(buf, prefix []byte) int {
	max := len(prefix) - 1
	if len(buf) < max {
		max = len(buf)
	}
	for n := max; n > 0; n-- {
		if bytes.Equal(buf[len(buf)-n:], prefix[:n]) {
			return n
		}
	}
	return 0
}
