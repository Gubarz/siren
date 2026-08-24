//go:build live

// Live-instance validation for the bloodhound service. Run with:
//
//	BH_URL=... BH_ID=... BH_KEY=... go test -tags live ./internal/bloodhound/ -run TestLive -v
//
// Optional correlation targets (never hardcode environment data in tests):
//
//	BH_TEST_HOSTNAME=pc01.example.local BH_TEST_USERNAME='EXAMPLE\jdoe'
//	BH_TEST_SID=S-1-5-21-...
package bloodhound

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	bh "github.com/Gubarz/bloodhound-sdk-go"
)

func liveService(t *testing.T) *Service {
	t.Helper()
	url, id, key := os.Getenv("BH_URL"), os.Getenv("BH_ID"), os.Getenv("BH_KEY")
	if url == "" || id == "" || key == "" {
		t.Skip("BH_URL/BH_ID/BH_KEY not set")
	}
	svc := New(t.TempDir(), nil)
	if err := svc.SaveConfig(Config{ServerURL: url, TokenID: id, TokenKey: key}); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}
	if err := svc.Connect(context.Background()); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	return svc
}

func TestLive(t *testing.T) {
	svc := liveService(t)
	ctx := context.Background()

	st := svc.Status()
	t.Logf("status: %+v", st)
	if !st.Connected {
		t.Fatal("not connected")
	}

	domains, err := svc.ListDomains(ctx)
	if err != nil {
		t.Fatalf("ListDomains: %v", err)
	}
	for _, d := range domains {
		t.Logf("domain: %+v", d)
	}

	page, err := svc.SearchEntities(ctx, "administrator", 0, 5)
	if err != nil {
		t.Fatalf("SearchEntities: %v", err)
	}
	for _, e := range page.Entities {
		t.Logf("hit: %s (%s) %s tz=%v owned=%v", e.Name, e.Kind, e.ObjectID, e.TierZero, e.Owned)
	}

	var userSID string
	for _, e := range page.Entities {
		if e.Kind == "User" {
			userSID = e.ObjectID
			break
		}
	}
	if userSID != "" {
		ent, err := svc.Entity(ctx, userSID)
		if err != nil {
			t.Fatalf("Entity: %v", err)
		}
		t.Logf("entity props (%d)", len(ent.Properties))

		graph, err := svc.EntityAttackPaths(ctx, userSID, 5)
		if err != nil {
			t.Fatalf("EntityAttackPaths: %v", err)
		}
		t.Logf("attack paths: nodes=%d edges=%d", len(graph.Nodes), len(graph.Edges))
		for _, n := range graph.Nodes {
			t.Logf("  node: %s (%s) tz=%v owned=%v", n.Label, n.Kind, n.TierZero, n.Owned)
		}
	}

	targets, err := svc.KerberoastTargets(ctx)
	if err != nil {
		t.Fatalf("KerberoastTargets: %v", err)
	}
	for _, target := range targets {
		t.Logf("kerberoastable: %s %s", target.Name, target.ObjectID)
	}

	jobs, err := svc.IngestJobs(ctx)
	if err != nil {
		t.Fatalf("IngestJobs: %v", err)
	}
	t.Logf("ingest jobs: %d", len(jobs))

	// Correlation targets come from the environment; the ghost ref is a
	// generic sentinel for the not-found path.
	refs := []AgentRef{{ID: "a3", Hostname: "ghost-does-not-exist"}}
	if h := os.Getenv("BH_TEST_HOSTNAME"); h != "" {
		refs = append([]AgentRef{{ID: "a1", Hostname: h}}, refs...)
	}
	if u := os.Getenv("BH_TEST_USERNAME"); u != "" {
		refs = append([]AgentRef{{ID: "a2", Username: u}}, refs...)
	}
	enrich, err := svc.Correlate(ctx, refs)
	if err != nil {
		t.Fatalf("Correlate: %v", err)
	}
	for _, ref := range refs {
		e := enrich[ref.ID]
		t.Logf("%s: entity=%s (%s) tz=%v owned=%v distance=%d paths=%d",
			ref.ID, e.Entity.Name, e.Entity.Kind, e.TierZero, e.Owned, e.DistanceToTierZero, len(e.Paths))
		for _, p := range e.Paths {
			t.Logf("    path: %s", p.Label)
		}
	}
}

// TestLiveRawEntityShape proxies the signed entity-detail request and dumps
// the raw response + outgoing headers, for diagnosing server/schema
// mismatches. Requires BH_TEST_SID pointing at a resolvable entity.
func TestLiveRawEntityShape(t *testing.T) {
	url, id, key := os.Getenv("BH_URL"), os.Getenv("BH_ID"), os.Getenv("BH_KEY")
	sid := os.Getenv("BH_TEST_SID")
	if url == "" || sid == "" {
		t.Skip("BH_URL and BH_TEST_SID not set")
	}
	var captured []string
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		target := url + r.URL.Path
		if r.URL.RawQuery != "" {
			target += "?" + r.URL.RawQuery
		}
		req, err := http.NewRequest(r.Method, target, r.Body)
		if err != nil {
			t.Errorf("new request: %v", err)
			return
		}
		req.Header = r.Header
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Errorf("forward: %v", err)
			return
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		if strings.Contains(r.URL.Path, "/api/v2/users/") || strings.Contains(r.URL.Path, "/api/v2/search") {
			hdr := ""
			for k, vv := range r.Header {
				for _, v := range vv {
					hdr += k + ": " + v + "\n"
				}
			}
			captured = append(captured, "STATUS="+resp.Status+" PATH="+r.URL.Path+"\n"+hdr)
		}
		for k, vv := range resp.Header {
			for _, v := range vv {
				w.Header().Add(k, v)
			}
		}
		w.WriteHeader(resp.StatusCode)
		_, _ = w.Write(body)
	}))
	defer proxy.Close()

	svc := New(t.TempDir(), nil)
	if err := svc.SaveConfig(Config{ServerURL: proxy.URL, TokenID: id, TokenKey: key}); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}
	if err := svc.Connect(context.Background()); err != nil {
		t.Fatalf("Connect: %v", err)
	}

	_, _ = svc.Entity(context.Background(), sid)
	// Reference: the SDK's own signed request to the same endpoint.
	_, _ = svc.SearchEntities(context.Background(), "administrator", 0, 2)
	client := svc.snapshotClient()
	if client != nil {
		_, _ = client.Community().ADUsers().UserEntity(context.Background(), sid)
	}
	for _, c := range captured {
		t.Logf("CAPTURED:\n%s", c)
	}
}

// TestLiveSignatureParity recomputes the HMAC both ways for identical inputs
// to prove the siren signer matches the SDK's algorithm.
func TestLiveSignatureParity(t *testing.T) {
	key := os.Getenv("BH_KEY")
	if key == "" {
		t.Skip("BH_KEY not set")
	}
	method := "GET"
	path := "/api/v2/users/S-1-5-21-EXAMPLE-SID"
	now := time.Now().UTC().Truncate(time.Minute)

	sdkOp := hmac.New(sha256.New, []byte(key))
	sdkOp.Write([]byte(method + path))
	sdkDigest := sdkOp.Sum(nil)
	sdkDate := hmac.New(sha256.New, sdkDigest)
	sdkDate.Write([]byte(now.Format("2006-01-02T15")))
	sdkFinal := hmac.New(sha256.New, sdkDate.Sum(nil))
	sdkSig := base64.StdEncoding.EncodeToString(sdkFinal.Sum(nil))

	req, _ := http.NewRequest(method, "http://x"+path, nil)
	signHMAC("tok", key, method, path, nil, now, req)
	ourSig := req.Header.Get("Signature")

	if sdkSig != ourSig {
		t.Fatalf("signature mismatch: sdk=%s ours=%s", sdkSig, ourSig)
	}
}

// snapshotClient exposes the live SDK client for comparison probes (live
// build only).
func (s *Service) snapshotClient() *bh.Client {
	c, _ := s.snapshot()
	return c
}
