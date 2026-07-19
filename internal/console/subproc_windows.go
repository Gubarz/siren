//go:build windows

package console

import "errors"

const ConsoleModeFlag = "--sliver-console"

// ErrConsoleSubprocUnsupported means the interactive per-session console
// is not yet ported to Windows (needs ConPTY). Everything else works.
var ErrConsoleSubprocUnsupported = errors.New("interactive console subprocess is not yet supported on Windows")

type subprocMgr struct{}

func (s *Service) StartConsole(_ string) (string, error)         { return "", ErrConsoleSubprocUnsupported }
func (s *Service) WriteConsole(_ string, _ []byte) error         { return ErrConsoleSubprocUnsupported }
func (s *Service) ResizeConsole(_ string, _ int, _ int) error    { return ErrConsoleSubprocUnsupported }
func (s *Service) StopConsole(_ string) error                    { return nil }
func (s *Service) SendToSessionConsole(_ string, _ string) error { return ErrConsoleSubprocUnsupported }
func (s *Service) CloseSubprocs()                                {}
func RunConsoleSubprocess(_ string, _ string) error {
	return ErrConsoleSubprocUnsupported
}
