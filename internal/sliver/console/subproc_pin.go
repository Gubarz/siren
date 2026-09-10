package console

import (
	"bytes"
	"encoding/base64"
	"os"
	"strconv"
	"strings"

	sliverconsole "github.com/bishopfox/sliver/client/console"
	"github.com/kballard/go-shellquote"
	reefconsole "github.com/reeflective/console"
	"github.com/spf13/cobra"
)

const (
	shellOpenFramePrefix      = "\x1b]777;siren-open-shell="
	consoleCommandFramePrefix = "\x1b]777;siren-command="
	controlFrameSuffix        = "\x07"
	shellOpenFrameSuffix      = controlFrameSuffix
	// controlFrameNonceSeparator sits between the frame's nonce and its payload.
	controlFrameNonceSeparator = ":"
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
		if socksCmd := findTopLevelCommand(root, "socks5"); socksCmd != nil {
			pinConsoleCommand(socksCmd, socksCommandLine)
		}
		if portfwdCmd := findTopLevelCommand(root, "portfwd"); portfwdCmd != nil {
			pinConsoleCommand(portfwdCmd, portfwdCommandLine)
		}
		if rportfwdCmd := findTopLevelCommand(root, "rportfwd"); rportfwdCmd != nil {
			pinConsoleCommand(rportfwdCmd, rportfwdCommandLine)
		}
		return root
	}
}

// pinConsoleCommand routes every invocation of cmd and its descendants to the
// GUI as a console command frame instead of running the Sliver client action.
func pinConsoleCommand(cmd *cobra.Command, commandLine func(*cobra.Command, []string) string) {
	wrapCommandRun(cmd, func(command *cobra.Command, args []string) bool {
		emitConsoleCommandFrame(commandLine(command, args))
		return false
	}, func() {})
	for _, child := range cmd.Commands() {
		pinConsoleCommand(child, commandLine)
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

// controlFrameNonce authenticates the frames this process writes. The subprocess
// sets it from the environment at startup; the parent leaves it empty and
// validates each frame against the nonce it minted for that job.
var controlFrameNonce string

func emitShellOpenFrame(tail string) {
	_, _ = os.Stdout.WriteString(controlFrame(shellOpenFramePrefix, tail))
}

func emitConsoleCommandFrame(line string) {
	_, _ = os.Stdout.WriteString(controlFrame(consoleCommandFramePrefix, line))
}

func controlFrame(prefix, payload string) string {
	encoded := base64.StdEncoding.EncodeToString([]byte(payload))
	return prefix + controlFrameNonce + controlFrameNonceSeparator + encoded + controlFrameSuffix
}

func socksCommandLine(cmd *cobra.Command, args []string) string {
	switch commandPath(cmd) {
	case "socks5 start":
		host, _ := cmd.Flags().GetString("host")
		port, _ := cmd.Flags().GetString("port")
		user, _ := cmd.Flags().GetString("user")
		parts := []string{"socks5", "start", "--host", host, "--port", port}
		if user != "" {
			parts = append(parts, "--user", user)
		}
		return shellquote.Join(parts...)
	case "socks5 stop":
		id, _ := cmd.Flags().GetUint64("id")
		return shellquote.Join("socks5", "stop", "--id", strconv.FormatUint(id, 10))
	default:
		if len(args) > 0 {
			return shellquote.Join(append([]string{"socks5"}, args...)...)
		}
		return "socks5"
	}
}

func portfwdCommandLine(cmd *cobra.Command, args []string) string {
	switch commandPath(cmd) {
	case "portfwd add":
		bind, _ := cmd.Flags().GetString("bind")
		remote, _ := cmd.Flags().GetString("remote")
		return shellquote.Join("portfwd", "add", "--bind", bind, "--remote", remote)
	case "portfwd rm":
		id, _ := cmd.Flags().GetInt("id")
		return shellquote.Join("portfwd", "rm", "--id", strconv.Itoa(id))
	default:
		if len(args) > 0 {
			return shellquote.Join(append([]string{"portfwd"}, args...)...)
		}
		return "portfwd"
	}
}

func rportfwdCommandLine(cmd *cobra.Command, args []string) string {
	switch commandPath(cmd) {
	case "rportfwd add":
		bind, _ := cmd.Flags().GetString("bind")
		remote, _ := cmd.Flags().GetString("remote")
		return shellquote.Join("rportfwd", "add", "--bind", bind, "--remote", remote)
	case "rportfwd rm":
		id, _ := cmd.Flags().GetUint32("id")
		return shellquote.Join("rportfwd", "rm", "--id", strconv.FormatUint(uint64(id), 10))
	default:
		if len(args) > 0 {
			return shellquote.Join(append([]string{"rportfwd"}, args...)...)
		}
		return "rportfwd"
	}
}

func filterConsoleControlFrames(expectedNonce string, carry, chunk []byte) ([]byte, []string, []string, []byte) {
	buf := make([]byte, 0, len(carry)+len(chunk))
	buf = append(buf, carry...)
	buf = append(buf, chunk...)

	visible := make([]byte, 0, len(buf))
	var shellTails []string
	var commands []string

	for len(buf) > 0 {
		idx, prefix, kind := nextControlFrame(buf)
		if idx < 0 {
			keep := maxControlPrefixSuffixLen(buf)
			visible = append(visible, buf[:len(buf)-keep]...)
			nextCarry := append([]byte(nil), buf[len(buf)-keep:]...)
			return visible, shellTails, commands, nextCarry
		}

		visible = append(visible, buf[:idx]...)
		rest := buf[idx+len(prefix):]
		end := bytes.Index(rest, []byte(controlFrameSuffix))
		if end < 0 {
			nextCarry := append([]byte(nil), buf[idx:]...)
			return visible, shellTails, commands, nextCarry
		}

		body := rest[:end]
		buf = rest[end+len(controlFrameSuffix):]

		// Anything that is not carrying this job's nonce came from the stream
		// rather than from our own subprocess. Drop it rather than acting on it.
		nonce, payload, ok := bytes.Cut(body, []byte(controlFrameNonceSeparator))
		if !ok || string(nonce) != expectedNonce {
			continue
		}
		if decoded, err := base64.StdEncoding.DecodeString(string(payload)); err == nil {
			switch kind {
			case "shell":
				shellTails = append(shellTails, string(decoded))
			case "command":
				commands = append(commands, string(decoded))
			}
		}
	}

	return visible, shellTails, commands, nil
}

func nextControlFrame(buf []byte) (int, []byte, string) {
	shellPrefix := []byte(shellOpenFramePrefix)
	commandPrefix := []byte(consoleCommandFramePrefix)
	shellIndex := bytes.Index(buf, shellPrefix)
	commandIndex := bytes.Index(buf, commandPrefix)
	if shellIndex < 0 && commandIndex < 0 {
		return -1, nil, ""
	}
	if commandIndex < 0 || (shellIndex >= 0 && shellIndex < commandIndex) {
		return shellIndex, shellPrefix, "shell"
	}
	return commandIndex, commandPrefix, "command"
}

func maxControlPrefixSuffixLen(buf []byte) int {
	shellKeep := prefixSuffixLen(buf, []byte(shellOpenFramePrefix))
	commandKeep := prefixSuffixLen(buf, []byte(consoleCommandFramePrefix))
	if shellKeep > commandKeep {
		return shellKeep
	}
	return commandKeep
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
