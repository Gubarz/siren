package console

import (
	"errors"
	"fmt"
	"io"
	stdlog "log"
	"os"

	"github.com/bishopfox/sliver/client/command"
	sliverconsole "github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/client/transport"
)

// RunConsoleSubprocess is the main entry for a --sliver-console re-exec.
// It reconnects to the server using the config the parent handed us,
// then hands control to sliver's own StartClient (which owns fd 0/1/2
// for readline). Session pinning is done via an rcScript containing
// `use <sessionID>` — this goes through sliver's real dispatch so menu
// switching, cwd, and any per-session bootstrap happen the same as if
// the user had typed it. Blocks until the user exits.
func RunConsoleSubprocess(configPath, sessionID string) error {
	if err := requireConsoleTerminal(); err != nil {
		return err
	}

	cfg, err := loadConfigFromFile(configPath)
	if err != nil {
		return err
	}
	_ = os.Remove(configPath)

	// Sliver's tunnel/socks/log-stream internals dump into the standard
	// logger. Left visible those messages ("Set stream", "TunnelLoop
	// exited", etc.) mix with real console output — we don't want them
	// in the user's terminal.
	stdlog.SetOutput(io.Discard)

	rpcClient, grpcConn, err := transport.MTLSConnect(cfg)
	if err != nil {
		return err
	}
	defer func() { _ = grpcConn.Close() }()

	// rcScript is the script *body*, not a path (sliver reads it directly
	// via bufio.NewScanner(strings.NewReader(...))). Passing `use <id>`
	// makes the console drop into the implant menu on the first prompt.
	rcScript := ""
	if sessionID != "" {
		rcScript = "use " + sessionID + "\n"
	}

	con := sliverconsole.NewConsole(false)
	serverCmds := command.ServerCommands(con, nil)
	sliverCmds := command.SliverCommands(con)
	if sessionID != "" {
		serverCmds = pinServerTargetCommands(serverCmds, sessionID, con)
		sliverCmds = pinSliverTargetCommands(sliverCmds, sessionID, con)
	}
	details := &sliverconsole.ConnectionDetails{Config: cfg}
	return sliverconsole.StartClient(con, rpcClient, grpcConn, details, serverCmds, sliverCmds, true, rcScript)
}

func requireConsoleTerminal() error {
	// Sliver's readline can spin forever if it starts with dead std handles.
	if os.Stdin == nil || os.Stdout == nil {
		return errors.New("console subprocess started without a terminal")
	}
	if _, err := os.Stdin.Stat(); err != nil {
		return fmt.Errorf("console subprocess stdin is unusable: %w", err)
	}
	if _, err := os.Stdout.Stat(); err != nil {
		return fmt.Errorf("console subprocess stdout is unusable: %w", err)
	}
	return nil
}
