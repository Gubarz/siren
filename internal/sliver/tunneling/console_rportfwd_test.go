package tunneling

import (
	"strings"
	"testing"
)

func TestHandleConsoleRportfwdAddStartsTrackedListener(t *testing.T) {
	var gotSessionID, gotBindAddr, gotRemoteAddr string
	start := func(sessionID, bindAddr, remoteAddr string) (uint64, error) {
		gotSessionID = sessionID
		gotBindAddr = bindAddr
		gotRemoteAddr = remoteAddr
		return 31, nil
	}

	output, err := handleConsoleRportfwdAdd(start, "session-1", []string{
		"--bind", "4444",
		"--remote", "8080",
	})
	if err != nil {
		t.Fatal(err)
	}
	if gotSessionID != "session-1" || gotBindAddr != ":4444" || gotRemoteAddr != "127.0.0.1:8080" {
		t.Fatalf("start args = (%q, %q, %q)", gotSessionID, gotBindAddr, gotRemoteAddr)
	}
	if !strings.Contains(output, "127.0.0.1:8080 <- :4444") {
		t.Fatalf("output = %q", output)
	}
}

func TestHandleConsoleRportfwdRemoveUsesPinnedSession(t *testing.T) {
	var gotID uint64
	var gotSessionID string
	stop := func(id uint64, sessionID string) error {
		gotID = id
		gotSessionID = sessionID
		return nil
	}

	output, err := handleConsoleRportfwdRemove(stop, "session-1", []string{"--id", "31"})
	if err != nil {
		t.Fatal(err)
	}
	if gotID != 31 || gotSessionID != "session-1" || output != "[*] Removed rportfwd\n" {
		t.Fatalf("stop = (%d, %q), output = %q", gotID, gotSessionID, output)
	}
}

func TestRportfwdSlicePreservesDuplicateIDsAcrossSessions(t *testing.T) {
	seen := map[rportfwdKey]ProxyInfo{
		{sessionID: "session-1", id: 1}: {ID: 1, Kind: "rportfwd", SessionID: "session-1"},
		{sessionID: "session-2", id: 1}: {ID: 1, Kind: "rportfwd", SessionID: "session-2"},
	}
	got := rportfwdSlice(seen)
	if len(got) != 2 {
		t.Fatalf("rportfwdSlice() returned %d rows, want 2", len(got))
	}
}

func TestFormatRportfwdListFiltersPinnedSession(t *testing.T) {
	proxies := []ProxyInfo{
		{ID: 1, Kind: "rportfwd", SessionID: "session-1", BindAddr: ":4444", RemoteAddr: "127.0.0.1:8080"},
		{ID: 1, Kind: "rportfwd", SessionID: "session-2", BindAddr: ":5555", RemoteAddr: "127.0.0.1:9090"},
	}
	output := formatRportfwdList(proxies, "session-1")
	if !strings.Contains(output, ":4444") || strings.Contains(output, ":5555") {
		t.Fatalf("output = %q", output)
	}
}
