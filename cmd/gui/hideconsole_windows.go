//go:build windows

package gui

import (
	"syscall"
)

var kernel32 = syscall.NewLazyDLL("kernel32.dll")

// hideConsoleWindow detaches the process from its console. siren ships as
// a console-subsystem binary so the --sliver-console subprocess can attach
// to a ConPTY (a GUI-subsystem child has no console, gets only raw pipes,
// and its readline spins forever, ballooning memory). The GUI itself has no
// use for the console, so we get rid of it entirely: conhost can (re)show a
// hidden console window (seen around WebView2 initialization), which makes
// ShowWindow-based hiding unreliable, while FreeConsole closes the window
// for good. It is safe when launched from a terminal too: the terminal
// window stays (owned by conhost, shared with the other processes), only
// siren detaches. The --sliver-console subprocess is unaffected — it gets
// its own console from the ConPTY and never reaches this code path.
func hideConsoleWindow() {
	kernel32.NewProc("FreeConsole").Call()
}
