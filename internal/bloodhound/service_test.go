package bloodhound

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	bh "github.com/Gubarz/bloodhound-sdk-go"
)

func fakeBHServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func overrideFactory(t *testing.T, srvURL string) {
	t.Helper()
	orig := clientFactory
	clientFactory = func(cfg Config) (*bh.Client, error) {
		return bh.NewClient(srvURL, cfg.TokenID, cfg.TokenKey)
	}
	t.Cleanup(func() { clientFactory = orig })
}

func TestSaveConfigConnectsWithoutExplicitConnect(t *testing.T) {
	srv := fakeBHServer(t)
	overrideFactory(t, srv.URL)
	svc := New(t.TempDir(), nil)
	if err := svc.SaveConfig(Config{ServerURL: srv.URL, TokenID: "id", TokenKey: "key"}); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}
	st := svc.Status()
	if !st.Connected {
		t.Fatalf("saving a configured server should establish the live connection: %+v", st)
	}
}

func TestConnectIfConfiguredRestoresConnectionAfterRestart(t *testing.T) {
	srv := fakeBHServer(t)
	overrideFactory(t, srv.URL)
	dir := t.TempDir()
	if err := New(dir, nil).SaveConfig(Config{ServerURL: srv.URL, TokenID: "id", TokenKey: "key"}); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}
	svc := New(dir, nil) // fresh instance, as after an app restart
	svc.ConnectIfConfigured(context.Background())
	if !svc.Status().Connected {
		t.Fatalf("ConnectIfConfigured should connect from the saved config: %+v", svc.Status())
	}
}

func TestConnectIfConfiguredWithoutConfigStaysIdle(t *testing.T) {
	svc := New(t.TempDir(), nil)
	svc.ConnectIfConfigured(context.Background())
	if svc.Status().Connected {
		t.Fatal("ConnectIfConfigured connected without a saved config")
	}
}

func TestConnectRequiresConfig(t *testing.T) {
	svc := New(t.TempDir(), nil)
	err := svc.Connect(context.Background())
	if !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("Connect() = %v, want ErrNotConfigured", err)
	}
	st := svc.Status()
	if st.Configured || st.Connected || st.Error == "" {
		t.Fatalf("Status after failed connect = %+v", st)
	}
}

func TestConnectDisconnectStatus(t *testing.T) {
	srv := fakeBHServer(t)
	overrideFactory(t, srv.URL)

	dir := t.TempDir()
	svc := New(dir, nil)
	if err := svc.SaveConfig(Config{ServerURL: srv.URL, TokenID: "id", TokenKey: "key"}); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}
	if err := svc.Connect(context.Background()); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	st := svc.Status()
	if !st.Configured || !st.Connected || st.Error != "" || st.ServerURL != srv.URL {
		t.Fatalf("Status = %+v", st)
	}
	view := svc.GetConfig()
	if !view.HasTokenKey || view.TokenID != "id" {
		t.Fatalf("GetConfig lost fields: %+v", view)
	}
	svc.Disconnect()
	if svc.Status().Connected {
		t.Fatal("still connected after Disconnect")
	}
}

func TestSaveConfigReconnectsAndReportsFailure(t *testing.T) {
	srv := fakeBHServer(t)
	overrideFactory(t, srv.URL)
	svc := New(t.TempDir(), nil)
	if err := svc.SaveConfig(Config{ServerURL: srv.URL, TokenID: "id", TokenKey: "key"}); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}
	if err := svc.Connect(context.Background()); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	// Point the factory at a dead server; saving while connected triggers a reconnect.
	overrideFactory(t, "http://127.0.0.1:1")
	err := svc.SaveConfig(Config{ServerURL: "http://127.0.0.1:1", TokenID: "id", TokenKey: "key"})
	if err == nil {
		t.Fatal("SaveConfig should report reconnect failure")
	}
	if svc.Status().Connected {
		t.Fatal("should be disconnected after failed reconnect")
	}
	if !errors.Is(err, ErrNotConfigured) && svc.Status().Error == "" {
		t.Fatalf("Status.Error should carry the failure: %+v", svc.Status())
	}
}

func TestMergeSaveConfigKeepsExistingKeyWhenBlank(t *testing.T) {
	srv := fakeBHServer(t)
	overrideFactory(t, srv.URL)
	svc := New(t.TempDir(), nil)
	full := Config{ServerURL: srv.URL, TokenID: "id", TokenKey: "key"}
	if err := svc.SaveConfig(full); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}
	if err := svc.MergeSaveConfig(Config{ServerURL: srv.URL, TokenID: "id2"}); err != nil {
		t.Fatalf("MergeSaveConfig: %v", err)
	}
	if got := svc.GetConfig(); got.TokenID != "id2" || !got.HasTokenKey {
		t.Fatalf("merged view = %+v, want tokenId id2 with key retained", got)
	}
}

func TestTestConnectionUsesStoredKeyWhenBlank(t *testing.T) {
	srv := fakeBHServer(t)
	overrideFactory(t, srv.URL)
	svc := New(t.TempDir(), nil)
	if err := svc.SaveConfig(Config{ServerURL: srv.URL, TokenID: "id", TokenKey: "key"}); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}
	if err := svc.Connect(context.Background()); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	// Blank key must fall back to the stored one (the overridden factory
	// rejects an empty token key), and the probe must leave the live
	// connection and config untouched.
	if err := svc.TestConnection(Config{TokenID: "id"}); err != nil {
		t.Fatalf("TestConnection with blank key = %v, want nil", err)
	}
	if !svc.Status().Connected {
		t.Fatal("probe must not disturb the live connection")
	}
	if got := svc.GetConfig(); got.TokenID != "id" || !got.HasTokenKey {
		t.Fatalf("GetConfig changed by probe: %+v", got)
	}
}

func TestTestConnectionFailsWithBadURL(t *testing.T) {
	svc := New(t.TempDir(), nil) // real clientFactory, no override
	err := svc.TestConnection(Config{ServerURL: "http://127.0.0.1:1", TokenID: "id", TokenKey: "key"})
	if err == nil {
		t.Fatal("TestConnection against a dead server should fail")
	}
	if st := svc.Status(); st.Error != "" || st.Connected {
		t.Fatalf("probe leaked into service state: %+v", st)
	}
}

func TestCloseIsSafeWhenNeverUsed(t *testing.T) {
	svc := New(t.TempDir(), nil)
	svc.Close() // must not panic
}

// blockingSearchServer serves everything like fakeBHServer except
// /search, whose handler signals entry and then waits for a release
// channel before responding.
func blockingSearchServer(t *testing.T) (*httptest.Server, <-chan struct{}, chan struct{}) {
	t.Helper()
	entered := make(chan struct{})
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v2/search" {
			close(entered)
			<-release
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	t.Cleanup(srv.Close)
	return srv, entered, release
}

func TestQueriesDoNotBlockConnect(t *testing.T) {
	srv, entered, release := blockingSearchServer(t)
	var releaseOnce sync.Once
	unblock := func() { releaseOnce.Do(func() { close(release) }) }
	defer unblock() // must run before t.Cleanup closes the server
	overrideFactory(t, srv.URL)

	svc := New(t.TempDir(), nil)
	if err := svc.SaveConfig(Config{ServerURL: srv.URL, TokenID: "id", TokenKey: "key"}); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}
	if err := svc.Connect(context.Background()); err != nil {
		t.Fatalf("Connect: %v", err)
	}

	// A search blocks inside the HTTP handler; it must not pin the
	// service lock, so a reconnect remains possible while it is in flight.
	searchDone := make(chan error, 1)
	go func() {
		_, err := svc.SearchEntities(context.Background(), "x", 0, 10)
		searchDone <- err
	}()

	select {
	case <-entered:
	case <-time.After(5 * time.Second):
		t.Fatal("search request never reached the fake server")
	}

	connectDone := make(chan error, 1)
	go func() {
		connectDone <- svc.Connect(context.Background())
	}()

	select {
	case err := <-connectDone:
		if err != nil {
			t.Fatalf("Connect() = %v, want nil", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Connect() blocked behind an in-flight search query")
	}

	unblock()
	if err := <-searchDone; err != nil {
		t.Fatalf("SearchEntities = %v, want nil", err)
	}
}
