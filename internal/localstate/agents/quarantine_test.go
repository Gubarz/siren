package agents

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Corrupt state is deliberately treated as empty, but the next Observe would
// then overwrite the only copy of the operator's known-agent history.
func TestCorruptKnownStateIsQuarantined(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, persistPrefix+".json")
	if err := os.WriteFile(path, []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}

	New(dir)

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("corrupt state still at %s, want it moved aside", path)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.Contains(entry.Name(), ".corrupt-") {
			return
		}
	}
	t.Fatalf("no quarantined copy of the corrupt state in %v", entries)
}
