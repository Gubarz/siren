//go:build windows

package gui

import (
	"syscall"
	"unsafe"
)

var (
	user32                 = syscall.NewLazyDLL("user32.dll")
	kernel32               = syscall.NewLazyDLL("kernel32.dll")
	procGetConsoleWindow   = kernel32.NewProc("GetConsoleWindow")
	procConsoleProcessList = kernel32.NewProc("GetConsoleProcessList")
	procShowWindow         = user32.NewProc("ShowWindow")
)

// hideConsoleWindow hides the process's own console window. siren ships as
// a console-subsystem binary so the --sliver-console subprocess can attach
// to a ConPTY (a GUI-subsystem child has no console, gets only raw pipes,
// and its readline spins forever, ballooning memory). The GUI itself has no
// use for the console, so we hide it. Only hide when we own the console
// exclusively: when launched from an existing terminal (e.g. powershell)
// the console is shared, and hiding it would take the user's terminal down.
func hideConsoleWindow() {
	if err := procGetConsoleWindow.Find(); err != nil {
		return
	}
	if err := procConsoleProcessList.Find(); err != nil {
		return
	}
	if err := procShowWindow.Find(); err != nil {
		return
	}
	cw, _, _ := procGetConsoleWindow.Call()
	if cw == 0 {
		return
	}
	var pids [2]uint32
	n, _, _ := procConsoleProcessList.Call(
		uintptr(unsafe.Pointer(&pids[0])),
		uintptr(len(pids)),
	)
	if n > 1 {
		return // shared console (launched from a terminal)
	}
	procShowWindow.Call(cw, 0) // SW_HIDE
}