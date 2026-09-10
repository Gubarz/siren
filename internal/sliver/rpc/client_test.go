package rpc

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// sliver names a client config "<operator>@<lhost> (<first 8 bytes of the
// certificate digest in hex>)", so a test that wants a profile to resolve has to
// build the same key.
func writeTestConfig(t *testing.T, operator, lhost string, lport int, certificate string) string {
	t.Helper()

	rootDir := t.TempDir()
	configDir := filepath.Join(rootDir, "configs")
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		t.Fatal(err)
	}
	body := fmt.Sprintf(
		`{"operator":%q,"lhost":%q,"lport":%d,"ca_certificate":"","private_key":"","certificate":%q}`,
		operator, lhost, lport, certificate,
	)
	if err := os.WriteFile(filepath.Join(configDir, "test.cfg"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SLIVER_CLIENT_ROOT_DIR", rootDir)

	digest := sha256.Sum256([]byte(certificate))
	return fmt.Sprintf("%s@%s (%x)", operator, lhost, digest[:8])
}

// The teardown hook closes resources bound to the outgoing connection. An
// unknown profile fails during config lookup, before there is anything to
// replace, so the hook must not run.
func TestConnectDoesNotRunTeardownWhenProfileIsUnknown(t *testing.T) {
	c := NewClient()

	called := false
	err := c.Connect("profile-that-does-not-exist", func() { called = true })
	if err == nil {
		t.Fatal("Connect() with an unknown profile = nil, want an error")
	}
	if called {
		t.Fatal("teardown ran even though no replacement connection was established")
	}
}

// The same holds when the profile resolves but the dial itself fails, which is
// what an operator hits when the new teamserver is unreachable. The live
// connection and everything bound to it must survive the failed switch.
func TestConnectDoesNotRunTeardownWhenDialFails(t *testing.T) {
	name := writeTestConfig(t, "test", "127.0.0.1", 1, "not-a-real-certificate")

	c := NewClient()

	called := false
	err := c.Connect(name, func() { called = true })
	if err == nil {
		t.Skip("the dial unexpectedly succeeded, so the failure path was not exercised")
	}
	if called {
		t.Fatal("teardown ran even though the dial failed")
	}
	if c.Connected() {
		t.Fatal("client reports connected after a failed dial")
	}
}
