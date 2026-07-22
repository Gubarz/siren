package tunneling

import (
	"errors"
	"net"
	"strings"
	"testing"
)

func TestHandleConsolePortfwdAddStartsTrackedProxy(t *testing.T) {
	var gotSessionID, gotBindAddr, gotRemoteAddr string
	start := func(sessionID, bindAddr, remoteAddr string) (uint64, error) {
		gotSessionID = sessionID
		gotBindAddr = bindAddr
		gotRemoteAddr = remoteAddr
		return 23, nil
	}

	output, err := handleConsolePortfwdAdd(start, "session-1", []string{
		"--bind", "9000",
		"--remote", "10.0.0.5:80",
	})
	if err != nil {
		t.Fatal(err)
	}
	if gotSessionID != "session-1" || gotBindAddr != "127.0.0.1:9000" || gotRemoteAddr != "10.0.0.5:80" {
		t.Fatalf("start args = (%q, %q, %q)", gotSessionID, gotBindAddr, gotRemoteAddr)
	}
	if !strings.Contains(output, "127.0.0.1:9000 -> 10.0.0.5:80") {
		t.Fatalf("output = %q, want bind and remote addresses", output)
	}
}

func TestHandleConsolePortfwdAddDoesNotRefreshAfterFailure(t *testing.T) {
	wantErr := errors.New("bind failed")
	start := func(string, string, string) (uint64, error) { return 0, wantErr }
	if _, err := handleConsolePortfwdAdd(start, "session-1", []string{"--remote", "10.0.0.5:80"}); !errors.Is(err, wantErr) {
		t.Fatalf("handleConsolePortfwdAdd() error = %v, want %v", err, wantErr)
	}

	svc := &Service{}
	result := svc.handleConsolePortfwdArgs("session-1", []string{"portfwd", "add"})
	if !result.Handled || result.Refresh || !strings.HasPrefix(result.Output, "[!] ") {
		t.Fatalf("result = %#v, want handled error without refresh", result)
	}
}

func TestRenderPortfwdListUsesParentServiceState(t *testing.T) {
	svc := &Service{
		pfwd: map[uint64]*portfwdProxy{
			7: {info: ProxyInfo{ID: 7, Kind: "portfwd", SessionID: "session-12345678", BindAddr: "127.0.0.1:9000", RemoteAddr: "10.0.0.5:80"}},
			8: {info: ProxyInfo{ID: 8, Kind: "portfwd", SessionID: "other-session", BindAddr: "127.0.0.1:9001", RemoteAddr: "10.0.0.6:443"}},
		},
		socks: map[uint64]*socksProxy{},
	}

	output := svc.renderPortfwdList("session-12345678")
	if !strings.Contains(output, "127.0.0.1:9000") || !strings.Contains(output, "10.0.0.5:80") {
		t.Fatalf("output = %q, want matching port forward", output)
	}
	if strings.Contains(output, "127.0.0.1:9001") {
		t.Fatalf("output = %q, contains another session's port forward", output)
	}
}

func TestHandleConsolePortfwdRemoveStopsParentProxy(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	svc := &Service{pfwd: map[uint64]*portfwdProxy{
		23: {listener: listener, quit: make(chan struct{})},
	}}

	result := svc.handleConsolePortfwdArgs("session-1", []string{"portfwd", "rm", "--id", "23"})
	if !result.Handled || !result.Refresh || result.Output != "[*] Removed portfwd\n" {
		t.Fatalf("result = %#v", result)
	}
	if _, ok := svc.pfwd[23]; ok {
		t.Fatal("portfwd 23 remains in parent service state")
	}
}

func TestHandleConsoleTunnelCommandRoutesPortfwd(t *testing.T) {
	svc := &Service{pfwd: map[uint64]*portfwdProxy{}, socks: map[uint64]*socksProxy{}}
	result := svc.HandleConsoleTunnelCommand("session-1", "portfwd")
	if !result.Handled || result.Output != "[*] No port forwards\n" {
		t.Fatalf("result = %#v", result)
	}
}
