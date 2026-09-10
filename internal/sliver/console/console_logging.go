package console

import (
	sliverconsole "github.com/bishopfox/sliver/client/console"
)

// newSliverConsole builds sliver's console with its own console logger disabled.
//
// Sliver enables console logging by default, and on that path it calls
// log.Fatalf when it cannot create or open <client-root>/logs/console. log.Fatalf
// ends the process, so an unwritable or full log directory would kill the GUI on
// the first console command. Siren keeps its own capture store and event
// pipeline, and disabling this also stops sliver writing full command lines,
// including inline credentials, to a plaintext log.
func newSliverConsole() *sliverconsole.SliverClient {
	con := sliverconsole.NewConsole(false)
	if con != nil && con.Settings != nil {
		con.Settings.ConsoleLogs = false
	}
	return con
}
