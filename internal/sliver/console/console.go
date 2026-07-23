package console

import (
	"context"
	"fmt"
	"io"
	stdlog "log"
	"os"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/bishopfox/sliver/client/command"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/kballard/go-shellquote"
	"github.com/spf13/cobra"

	"sliver-gui/internal/sliver/rpc"
)

var BeaconTaskNoticePattern = regexp.MustCompile(`(?i)Tasked beacon .*\(([0-9a-f]{8})\)`)

const commandTimeout = 90 * time.Second

type completionTrie struct {
	children map[string]*completionTrie
}

func buildCompletions(cmd *cobra.Command) *completionTrie {
	root := &completionTrie{children: make(map[string]*completionTrie)}
	for _, c := range cmd.Commands() {
		if c.Hidden {
			continue
		}
		child := buildCompletions(c)
		root.children[c.Name()] = child
	}
	return root
}

func (t *completionTrie) lookup(args []string) []string {
	if t == nil {
		return nil
	}
	if len(args) == 0 {
		out := make([]string, 0, len(t.children))
		for name := range t.children {
			out = append(out, name)
		}
		return out
	}

	token := args[0]
	if len(args) == 1 {
		var out []string
		for name := range t.children {
			if strings.HasPrefix(name, token) {
				out = append(out, name)
			}
		}
		return out
	}

	if child, ok := t.children[token]; ok {
		return child.lookup(args[1:])
	}
	return nil
}

type Emitter interface {
	Emit(name string, payload any)
}

type Service struct {
	rpc *rpc.Client

	mu           sync.Mutex
	sliverCon    *console.SliverClient
	sliverCmds   func() *cobra.Command
	serverCmds   func() *cobra.Command
	sliverRoot   *cobra.Command
	serverRoot   *cobra.Command
	sliverCmpl   *completionTrie
	serverCmpl   *completionTrie
	menuSession  string
	output       outputSink
	consoleInit  bool
	consoleOnce  sync.Once
	consoleErr   error

	emitter Emitter
	subproc subprocMgr

	routedCommand RoutedCommandHandler
}

func (s *Service) SetEmitter(e Emitter) {
	s.emitter = e
}

type RoutedCommandResult struct {
	Handled bool
	Output  string
	Refresh bool
}

type RoutedCommandHandler func(sessionID, line string) RoutedCommandResult

func (s *Service) SetRoutedCommandHandler(handler RoutedCommandHandler) {
	s.routedCommand = handler
}

func New(rpc *rpc.Client) *Service {
	return &Service{rpc: rpc}
}

func (s *Service) init() error {
	s.consoleOnce.Do(func() {
		// Sliver's tunnel/socks internals use the standard library logger, which
		// otherwise dumps every proxied connection to the parent terminal.
		stdlog.SetOutput(io.Discard)
		// Sliver's interactive prompts (e.g. `pivot stop` with no id) read from
		// os.Stdin. Detach it so those prompts fail immediately instead of
		// hijacking the terminal that launched the GUI.
		if devNull, err := os.Open(os.DevNull); err == nil {
			os.Stdin = devNull
		}
		con := console.NewConsole(false)
		serverCmds := command.ServerCommands(con, nil)
		sliverCmds := command.SliverCommands(con)

		details := &console.ConnectionDetails{Config: s.rpc.Config}
		if err := console.StartClient(con, s.rpc.RPC, s.rpc.Conn, details, serverCmds, sliverCmds, false, ""); err != nil {
			s.consoleErr = err
			return
		}
		installOutputSink(con, &s.output)
		con.App.PreCmdRunHooks = append(con.App.PreCmdRunHooks, func() error {
			if menu := con.App.ActiveMenu(); menu != nil {
				routeCommandOutput(menu.Command, &s.output)
			}
			return nil
		})
		s.sliverCon = con
		s.sliverCmds = sliverCmds
		s.serverCmds = serverCmds
		s.sliverRoot = sliverCmds()
		s.serverRoot = serverCmds()
		s.sliverCmpl = buildCompletions(s.sliverRoot)
		s.serverCmpl = buildCompletions(s.serverRoot)
		s.consoleInit = true
	})
	return s.consoleErr
}

func (s *Service) ResetConsole() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.resetConsoleLocked()
}

func (s *Service) TryResetConsole() bool {
	if !s.mu.TryLock() {
		return false
	}
	defer s.mu.Unlock()

	s.resetConsoleLocked()
	return true
}

func (s *Service) resetConsoleLocked() {
	if s.sliverCon != nil {
		_ = s.sliverCon.CloseConnection()
	}
	s.consoleInit = false
	s.sliverCon = nil
	s.sliverCmds = nil
	s.serverCmds = nil
	s.sliverRoot = nil
	s.serverRoot = nil
	s.sliverCmpl = nil
	s.serverCmpl = nil
	s.menuSession = ""
	s.consoleErr = nil
	s.consoleOnce = sync.Once{}
}

func (s *Service) FindTarget(id string) (*clientpb.Session, *clientpb.Beacon, error) {
	if sess := s.rpc.LookupSession(id); sess != nil {
		return sess, nil, nil
	}
	if beacon := s.rpc.LookupBeacon(id); beacon != nil {
		return nil, beacon, nil
	}

	ctx := context.Background()
	if sessions, err := s.rpc.RPC.GetSessions(ctx, &commonpb.Empty{}); err == nil {
		s.rpc.PopulateSessions(sessions)
		for _, sess := range sessions.Sessions {
			if sess.ID == id {
				return sess, nil, nil
			}
		}
	}
	if beacons, err := s.rpc.RPC.GetBeacons(ctx, &commonpb.Empty{}); err == nil {
		s.rpc.PopulateBeacons(beacons)
		for _, b := range beacons.Beacons {
			if b.ID == id {
				return nil, b, nil
			}
		}
	}
	return nil, nil, fmt.Errorf("agent not found: %s", id)
}

func (s *Service) ListCommands() ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.init(); err != nil {
		return nil, err
	}

	root := s.sliverRoot
	serverRoot := s.serverRoot

	cmdMap := make(map[string]bool)
	var names []string

	for _, c := range root.Commands() {
		if c.Hidden {
			continue
		}
		cmdMap[c.Name()] = true
	}
	for _, c := range serverRoot.Commands() {
		if c.Hidden {
			continue
		}
		cmdMap[c.Name()] = true
	}

	for name := range cmdMap {
		names = append(names, name)
	}

	sort.Strings(names)
	return names, nil
}

func (s *Service) Render(fn func() error) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.init(); err != nil {
		return "", err
	}

	output, err := s.output.capture(fn)
	return strings.TrimRight(output, "\n"), err
}

func (s *Service) CommandInvokesPing(line string) bool {
	args, err := shellquote.Split(line)
	if err != nil {
		return false
	}
	for _, arg := range args {
		name := strings.ToLower(strings.TrimSpace(arg))
		name = strings.TrimSuffix(name, ".exe")
		if name == "ping" || strings.HasSuffix(name, `/ping`) || strings.HasSuffix(name, `\ping`) {
			return true
		}
	}
	return false
}

func (s *Service) SliverCon() *console.SliverClient {
	return s.sliverCon
}

func (s *Service) RemoveBeaconTaskCallback(taskID string) {
	if s.sliverCon == nil {
		return
	}
	s.sliverCon.BeaconTaskCallbacksMutex.Lock()
	delete(s.sliverCon.BeaconTaskCallbacks, taskID)
	s.sliverCon.BeaconTaskCallbacksMutex.Unlock()
}

func (s *Service) RemoveBeaconTaskCallbackByPrefix(prefix string) string {
	if s.sliverCon == nil {
		return ""
	}

	prefix = strings.ToLower(prefix)
	s.sliverCon.BeaconTaskCallbacksMutex.Lock()
	defer s.sliverCon.BeaconTaskCallbacksMutex.Unlock()
	for taskID := range s.sliverCon.BeaconTaskCallbacks {
		if strings.HasPrefix(strings.ToLower(taskID), prefix) {
			delete(s.sliverCon.BeaconTaskCallbacks, taskID)
			return taskID
		}
	}
	return ""
}

func (s *Service) GetCommandRoot(scope string) (*cobra.Command, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.init(); err != nil {
		return nil, err
	}

	switch scope {
	case "session":
		return s.sliverRoot, nil
	case "server":
		return s.serverRoot, nil
	default:
		return nil, fmt.Errorf("unknown command scope %q", scope)
	}
}
