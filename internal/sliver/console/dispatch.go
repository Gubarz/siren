package console

import (
	"context"
	"fmt"
	"strings"

	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/kballard/go-shellquote"
	"github.com/spf13/cobra"
)

// RunLine is the sync entrypoint used by every "type a command" surface:
// modals that build CLI strings, the raw console panel, and the automation
// engine's non-scripted rules. Every call round-trips through sliver's own
// cobra dispatch so its parse/complete/print behavior stays authoritative.
func (s *Service) RunLine(sessionID, line string) (string, error) {
	return s.RunLineContext(context.Background(), sessionID, line)
}

// RunAutomationLine is RunLine plus beacon-task callback bookkeeping — when
// a rule kicks off a beacon-async command, the returned taskID lets the
// engine correlate the eventual callback with the run record.
func (s *Service) RunAutomationLine(sessionID, line string) (string, string, error) {
	return s.RunAutomationLineContext(context.Background(), sessionID, line)
}

func (s *Service) RunLineContext(ctx context.Context, sessionID, line string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.init(); err != nil {
		return "", err
	}

	sess, beacon, err := s.resolveTarget(sessionID, line)
	if err != nil {
		return "", err
	}

	s.setActiveTarget(sessionID, sess, beacon)
	ctx = withCommandOverlay(ctx, sessionID, targetKindOf(sess, beacon), hostnameOf(sess, beacon), line)
	if output, handled, err := s.runDirectCommand(line, sess, beacon); handled {
		return output, err
	}
	return s.execCapture(ctx, line)
}

func (s *Service) RunAutomationLineContext(ctx context.Context, sessionID, line string) (string, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.init(); err != nil {
		return "", "", err
	}

	sess, beacon, err := s.resolveTarget(sessionID, line)
	if err != nil {
		return "", "", err
	}

	s.setActiveTarget(sessionID, sess, beacon)
	ctx = withCommandOverlay(ctx, sessionID, targetKindOf(sess, beacon), hostnameOf(sess, beacon), line)
	output, err := s.execCapture(ctx, line)
	if err != nil {
		return output, "", err
	}

	matches := BeaconTaskNoticePattern.FindStringSubmatch(output)
	if len(matches) != 2 {
		return output, "", nil
	}
	return output, s.RemoveBeaconTaskCallbackByPrefix(matches[1]), nil
}

func (s *Service) resolveTarget(sessionID, line string) (*clientpb.Session, *clientpb.Beacon, error) {
	root := s.serverCmds()
	if sessionID != "" {
		root = s.sliverCmds()
	}
	if reason := unsupportedConsoleCommand(root, line); reason != "" {
		return nil, nil, fmt.Errorf("%s", reason)
	}
	if sessionID == "" {
		return nil, nil, nil
	}
	return s.FindTarget(sessionID)
}

func (s *Service) setActiveTarget(sessionID string, sess *clientpb.Session, beacon *clientpb.Beacon) {
	if sessionID != "" {
		s.sliverCon.ActiveTarget.Set(sess, beacon)
		s.sliverCon.App.SwitchMenu(consts.ImplantMenu)
	} else {
		s.sliverCon.ActiveTarget.Set(nil, nil)
		s.sliverCon.App.SwitchMenu(consts.ServerMenu)
	}
}

func (s *Service) execCapture(ctx context.Context, line string) (string, error) {
	menu := s.sliverCon.App.ActiveMenu()
	ctx, cancel := context.WithTimeout(ctx, commandTimeout)
	defer cancel()

	out, runErr := s.output.capture(func() error {
		return menu.RunCommandLine(ctx, line)
	})
	trimmed := strings.TrimRight(out, "\n")
	if runErr == nil {
		s.publishConsoleOutput(ctx, line, trimmed)
	}
	return trimmed, runErr
}

// unsupportedConsoleCommand returns a human message if the parsed command
// requires an interactive terminal we can't provide from the GUI. Guarding
// upstream keeps a broken UX (silent hang, terminal takeover) out of the
// dispatch path.
func unsupportedConsoleCommand(root *cobra.Command, line string) string {
	args, err := shellquote.Split(line)
	if err != nil || len(args) == 0 {
		return ""
	}
	command, _, err := root.Find(args)
	if err != nil || command == nil {
		return ""
	}
	return NonInteractiveCommandReason(commandPath(command))
}

func commandPath(command *cobra.Command) string {
	var parts []string
	for current := command; current != nil && current.Parent() != nil; current = current.Parent() {
		parts = append(parts, current.Name())
	}
	for left, right := 0, len(parts)-1; left < right; left, right = left+1, right-1 {
		parts[left], parts[right] = parts[right], parts[left]
	}
	return strings.Join(parts, " ")
}

func NonInteractiveCommandReason(path string) string {
	switch path {
	case "shell attach":
		return "Attaching to a managed shell requires an interactive terminal."
	case "edit":
		return "Use the GUI file browser; edit requires an interactive terminal."
	case "hexedit":
		return "Hex editing requires an interactive terminal."
	case "docs":
		return "The documentation browser requires an interactive terminal."
	case "switch":
		return "Server switching requires an interactive terminal selector."
	case "ai":
		return "The AI conversation interface requires an interactive terminal."
	default:
		return ""
	}
}
