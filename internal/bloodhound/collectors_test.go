package bloodhound

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const payload = "FAKE-SHARPHOUND-BINARY-BYTES"

func payloadSum() string {
	sum := sha256.Sum256([]byte(payload))
	return hex.EncodeToString(sum[:])
}

// fakeCollectorServer serves manifest/checksum/download endpoints for the
// sharphound collector, with an optional bad checksum to force failure.
func fakeCollectorServer(t *testing.T, badChecksum bool) (*httptest.Server, *int) {
	t.Helper()
	requests := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		switch {
		case strings.HasSuffix(r.URL.Path, "/checksum"):
			w.Header().Set("Content-Type", "text/plain")
			// CE returns sha256sum-style output with a trailing filename.
			if badChecksum {
				_, _ = w.Write([]byte("0" + payloadSum()[1:] + " *SharpHound_v2.14.0_windows_x86.zip"))
			} else {
				_, _ = w.Write([]byte(payloadSum() + " *SharpHound_v2.14.0_windows_x86.zip"))
			}
		case r.URL.Path == "/api/v2/collectors/sharphound":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"data":{"latest":"v2.1.0","versions":[{"version":"v2.1.0","sha256sum":"` + payloadSum() + `"}]}}`))
		case strings.HasPrefix(r.URL.Path, "/api/v2/collectors/sharphound/"):
			w.Header().Set("Content-Type", "application/octet-stream")
			_, _ = w.Write([]byte(payload))
		default:
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))
		}
	}))
	t.Cleanup(srv.Close)
	return srv, &requests
}

func TestCollectorDownloadVerifiesAndCaches(t *testing.T) {
	srv, requests := fakeCollectorServer(t, false)
	svc := connectedService(t, srv.URL)

	path, checksum, err := svc.CollectorDownload(context.Background(), "sharphound", "")
	if err != nil {
		t.Fatalf("CollectorDownload: %v", err)
	}
	if checksum != payloadSum() {
		t.Fatalf("checksum = %s, want %s", checksum, payloadSum())
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != payload {
		t.Fatalf("downloaded data = %q, want %q", data, payload)
	}
	if !strings.HasSuffix(path, filepath.Join("collectors", "sharphound", "v2.1.0", "sharphound.exe")) {
		t.Fatalf("path = %s", path)
	}
	first := *requests

	// Second call resolves the latest tag (one manifest request) then hits
	// the on-disk cache: no checksum or download requests.
	if _, _, err := svc.CollectorDownload(context.Background(), "sharphound", ""); err != nil {
		t.Fatalf("CollectorDownload (cached): %v", err)
	}
	if *requests != first+1 {
		t.Fatalf("cached download made %d requests, want 1 (manifest only): %d -> %d", *requests-first, first, *requests)
	}
}

func TestCollectorDownloadChecksumMismatch(t *testing.T) {
	srv, _ := fakeCollectorServer(t, true)
	dir := t.TempDir()
	overrideFactory(t, srv.URL)
	svc := New(dir, nil)
	if err := svc.SaveConfig(Config{ServerURL: srv.URL, TokenID: "id", TokenKey: "key"}); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}
	if err := svc.Connect(context.Background()); err != nil {
		t.Fatalf("Connect: %v", err)
	}

	_, _, err := svc.CollectorDownload(context.Background(), "sharphound", "")
	if err == nil || !strings.Contains(err.Error(), "checksum") {
		t.Fatalf("CollectorDownload = %v, want checksum mismatch error", err)
	}
	target := filepath.Join(dir, "collectors", "sharphound", "v2.1.0", "sharphound.exe")
	if _, err := os.Stat(target); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("mismatched download should be removed, stat = %v", err)
	}
}

func TestCollectorDownloadUnknownType(t *testing.T) {
	srv, _ := fakeCollectorServer(t, false)
	svc := connectedService(t, srv.URL)

	if _, _, err := svc.CollectorDownload(context.Background(), "metasploit", ""); err == nil {
		t.Fatal("unknown collector type should fail")
	}
}
