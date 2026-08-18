package gui

import (
	"context"
	"time"

	"siren/internal/sliver/catalog"
)

// ---- Console / Commands ----

func (a *App) RunSessionCommand(sessionID, line string) (string, error) {
	var output string
	var err error
	if sessionID != "" && a.RPC.LookupBeacon(sessionID) != nil {
		var taskID string
		output, taskID, err = a.Console.RunAutomationLine(sessionID, line)
		if err == nil {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()
			output, _, err = a.Beacons.AwaitBeaconTask(ctx, sessionID, output, taskID)
		}
	} else {
		output, err = a.Console.RunLine(sessionID, line)
	}
	if err == nil && sessionID != "" && a.Console.CommandInvokesPing(line) {
		a.Discovery.HandlePingOutput(sessionID, output)
	}
	return output, err
}

// StartConsole spawns a per-session sliver client subprocess attached to
// a PTY. The frontend drives the returned jobID via WriteConsole /
// ResizeConsole / StopConsole and receives bytes on the "console-output"
// event. See internal/console/subproc.go.
func (a *App) StartConsole(sessionID string) (string, error) {
	return a.Console.StartConsole(sessionID)
}

type consoleLease struct {
	JobID    string `json:"jobID"`
	Existing bool   `json:"existing"`
}

// AcquireConsole distinguishes a new console from another view attaching to
// the one per-session console. Attached views replay output with stdin muted so
// historical terminal queries cannot inject responses into the live prompt.
func (a *App) AcquireConsole(sessionID string) (consoleLease, error) {
	jobID, existing, err := a.Console.AcquireConsole(sessionID)
	return consoleLease{JobID: jobID, Existing: existing}, err
}

func (a *App) WriteConsole(jobID, data string) error {
	return a.Console.WriteConsole(jobID, []byte(data))
}

func (a *App) ResizeConsole(jobID string, cols, rows int) error {
	return a.Console.ResizeConsole(jobID, cols, rows)
}

func (a *App) StopConsole(jobID string) error {
	return a.Console.StopConsole(jobID)
}

func (a *App) GetConsoleOutput(jobID string) (string, error) {
	return a.Console.GetConsoleOutput(jobID)
}

// SendToSessionConsole is what GUI actions (palette, right-click,
// panels) should call to run a command against a session — it routes
// via the session's live subprocess console so any interactive prompt
// the command triggers (forms.Select, tea programs) renders in xterm.js
// instead of leaking to the launching terminal. If no console is up
// yet, the line queues and runs on the next StartConsole.
func (a *App) SendToSessionConsole(sessionID, line string) error {
	return a.Console.SendToSessionConsole(sessionID, line)
}

func (a *App) ListCommands() ([]string, error) {
	return a.Console.ListCommands()
}

func (a *App) CompleteCommand(sessionID, line string) ([]string, error) {
	return a.Console.CompleteCommand(sessionID, line)
}

func (a *App) CompletePath(sessionID, partial string) ([]string, error) {
	return a.Console.CompletePath(sessionID, partial)
}

func (a *App) GetCommandCatalog(scope string) (*catalog.CommandCatalog, error) {
	return a.Catalog.GetCommandCatalog(scope)
}
