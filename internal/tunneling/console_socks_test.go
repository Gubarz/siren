package tunneling

import (
	"fmt"
	"strings"
	"testing"
)

func TestHandleConsoleSocksStartStartsTrackedProxy(t *testing.T) {
	var gotSessionID, gotBindAddr, gotUsername, gotPassword string
	start := func(sessionID, bindAddr, username, password string) (uint64, error) {
		gotSessionID = sessionID
		gotBindAddr = bindAddr
		gotUsername = username
		gotPassword = password
		return 17, nil
	}

	output, err := handleConsoleSocksStart(start, "session-1", []string{
		"--host", "::1",
		"--port", "1080",
		"--user", "operator",
	})
	if err != nil {
		t.Fatal(err)
	}
	if gotSessionID != "session-1" || gotBindAddr != "[::1]:1080" || gotUsername != "operator" {
		t.Fatalf("start args = (%q, %q, %q), want (session-1, [::1]:1080, operator)", gotSessionID, gotBindAddr, gotUsername)
	}
	if gotPassword == "" {
		t.Fatal("start password is empty for an authenticated proxy")
	}
	if !strings.Contains(output, fmt.Sprintf("operator %s", gotPassword)) {
		t.Fatalf("output = %q, want generated credentials", output)
	}
}

func TestHandleConsoleSocksStartRejectsEmptyBindParts(t *testing.T) {
	svc := &Service{}

	for _, line := range []string{
		"socks5 start --host '' --port 1080",
		"socks5 start --host 127.0.0.1 --port ''",
	} {
		result := svc.HandleConsoleSocksCommand("session-1", line)
		if !result.Handled {
			t.Fatalf("HandleConsoleSocksCommand(%q) was not handled", line)
		}
		if result.Refresh {
			t.Fatalf("HandleConsoleSocksCommand(%q) requested a refresh after failing", line)
		}
		if !strings.HasPrefix(result.Output, "[!] ") {
			t.Fatalf("HandleConsoleSocksCommand(%q) output = %q, want an error", line, result.Output)
		}
	}
}
