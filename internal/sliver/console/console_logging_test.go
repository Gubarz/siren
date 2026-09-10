package console

import (
	"os"
	"path/filepath"
	"testing"

	"siren/internal/sliver/rpc"
)

// The console log directory lives under the sliver client root. When sliver
// cannot create or open it, its logger calls log.Fatalf and the GUI exits with
// no diagnostic, so init() must not depend on that directory at all.
func TestInitDoesNotUseSliverConsoleLogs(t *testing.T) {
	rootDir := t.TempDir()
	t.Setenv("SLIVER_CLIENT_ROOT_DIR", rootDir)

	svc := New(rpc.NewClient())
	if _, err := svc.RunLine("", "help"); err != nil {
		t.Fatalf("RunLine(help) = %v, want success", err)
	}
	if svc.sliverCon == nil || svc.sliverCon.Settings == nil {
		t.Fatal("console was not initialised with settings")
	}
	if svc.sliverCon.Settings.ConsoleLogs {
		t.Fatal("sliver console logging is enabled; an unwritable log dir would exit the process")
	}
	if _, err := os.Stat(filepath.Join(rootDir, "logs")); !os.IsNotExist(err) {
		t.Fatalf("sliver created its log tree under %s, want it untouched", rootDir)
	}
}
