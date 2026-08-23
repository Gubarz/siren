package bloodhound

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// routedServer routes canned JSON bodies by path prefix and records requests.
type routedServer struct {
	url      string
	requests []string // "METHOD path?query"
	bodies   map[string]func(r *http.Request) string
}

func newRoutedServer(t *testing.T, bodies map[string]func(r *http.Request) string) *routedServer {
	t.Helper()
	f := &routedServer{bodies: bodies}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.requests = append(f.requests, r.Method+" "+r.URL.Path+"?"+r.URL.RawQuery)
		w.Header().Set("Content-Type", "application/json")
		for prefix, body := range f.bodies {
			if strings.HasPrefix(r.URL.Path, prefix) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(body(r)))
				return
			}
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	t.Cleanup(srv.Close)
	f.url = srv.URL
	return f
}

func (f *routedServer) searchRequests() []string {
	var out []string
	for _, req := range f.requests {
		if strings.HasPrefix(req, "GET /api/v2/search?") {
			out = append(out, req)
		}
	}
	return out
}

func TestSearchEntitiesPagination(t *testing.T) {
	f := newRoutedServer(t, map[string]func(r *http.Request) string{
		"/api/v2/search": func(r *http.Request) string { return `{"data":[]}` },
	})
	svc := connectedService(t, f.url)

	page, err := svc.SearchEntities(context.Background(), "adm", 2, 25)
	if err != nil {
		t.Fatalf("SearchEntities: %v", err)
	}
	if page.Offset != 2 || page.Limit != 25 || len(page.Entities) != 0 {
		t.Fatalf("page = %+v", page)
	}

	// Clamping: limit 0 → 25, limit 200 → 100, negative offset → 0.
	if _, err := svc.SearchEntities(context.Background(), "adm", 0, 0); err != nil {
		t.Fatalf("SearchEntities(0,0): %v", err)
	}
	if _, err := svc.SearchEntities(context.Background(), "adm", -5, 200); err != nil {
		t.Fatalf("SearchEntities(-5,200): %v", err)
	}
	if _, err := svc.SearchEntities(context.Background(), "adm", 0, 100); err != nil {
		t.Fatalf("SearchEntities(0,100): %v", err)
	}

	wants := f.searchRequests()
	if len(wants) != 4 {
		t.Fatalf("search request count = %d, want 4 (%v)", len(wants), wants)
	}
	for _, pair := range []struct{ skip, limit string }{
		{"2", "25"}, {"0", "25"}, {"0", "100"}, {"0", "100"},
	} {
		found := false
		for _, w := range wants {
			if strings.Contains(w, "skip="+pair.skip) && strings.Contains(w, "limit="+pair.limit) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("no search request with skip=%s limit=%s in %v", pair.skip, pair.limit, wants)
		}
	}
}

func TestSearchEntitiesMapsOwnedAndTierZero(t *testing.T) {
	f := newRoutedServer(t, map[string]func(r *http.Request) string{
		"/api/v2/search": func(r *http.Request) string {
			return `{"data":[
				{"name":"DA@corp.local","objectid":"S-1-5-999","type":"Group","system_tags":"owned admin_tier_0"},
				{"name":"pc1.corp.local","objectid":"S-1-5-111","type":"Computer"}
			]}`
		},
	})
	svc := connectedService(t, f.url)

	page, err := svc.SearchEntities(context.Background(), "corp", 0, 25)
	if err != nil {
		t.Fatalf("SearchEntities: %v", err)
	}
	if len(page.Entities) != 2 {
		t.Fatalf("len = %d, want 2", len(page.Entities))
	}
	got := page.Entities[0]
	if got.ObjectID != "S-1-5-999" || got.Name != "DA@corp.local" || got.Kind != "Group" || !got.Owned || !got.TierZero {
		t.Fatalf("entities[0] = %+v", got)
	}
	if page.Entities[1].Owned || page.Entities[1].TierZero {
		t.Fatalf("entities[1] should have no flags: %+v", page.Entities[1])
	}
}

func TestListDomains(t *testing.T) {
	f := newRoutedServer(t, map[string]func(r *http.Request) string{
		"/api/v2/available-domains": func(r *http.Request) string {
			return `{"data":[{"id":"S-1-5-21-123","name":"corp.local","type":"ActiveDirectory","collected":true}]}`
		},
	})
	svc := connectedService(t, f.url)

	domains, err := svc.ListDomains(context.Background())
	if err != nil {
		t.Fatalf("ListDomains: %v", err)
	}
	if len(domains) != 1 {
		t.Fatalf("len = %d, want 1", len(domains))
	}
	d := domains[0]
	if d.ObjectID != "S-1-5-21-123" || d.Name != "corp.local" || !d.Collected {
		t.Fatalf("domain = %+v", d)
	}
}

func TestEntityMergesTypedProps(t *testing.T) {
	var sawAuth, sawDate, sawSig bool
	f := newRoutedServer(t, map[string]func(r *http.Request) string{
		"/api/v2/search": func(r *http.Request) string {
			return `{"data":[{"name":"jane@corp.local","objectid":"S-1-5-21-77","type":"User","system_tags":"owned"}]}`
		},
		"/api/v2/users/": func(r *http.Request) string {
			sawAuth = r.Header.Get("Authorization") != ""
			sawDate = r.Header.Get("RequestDate") != ""
			sawSig = r.Header.Get("Signature") != ""
			// Real CE shape: flat scalar props (the SDK's nested model
			// cannot unmarshal this, which is why we fetch it ourselves).
			return `{"data":{"kinds":["User","Base"],"props":{"admincount":true,"hasspn":false,"description":"svc","serviceprincipalnames":[]}}}`
		},
	})
	svc := connectedService(t, f.url)

	ent, err := svc.Entity(context.Background(), "S-1-5-21-77")
	if err != nil {
		t.Fatalf("Entity: %v", err)
	}
	if ent.ObjectID != "S-1-5-21-77" || ent.Kind != "User" || !ent.Owned || ent.TierZero {
		t.Fatalf("entity = %+v", ent)
	}
	if ent.Properties["admincount"] != "true" || ent.Properties["hasspn"] != "false" || ent.Properties["description"] != "svc" {
		t.Fatalf("properties = %+v", ent.Properties)
	}
	if _, hasArray := ent.Properties["serviceprincipalnames"]; hasArray {
		t.Fatalf("array props must be skipped: %+v", ent.Properties)
	}
	if !sawAuth || !sawDate || !sawSig {
		t.Fatalf("detail request must carry HMAC headers (auth=%v date=%v sig=%v)", sawAuth, sawDate, sawSig)
	}
	// The request path must carry the leading slash: the HMAC is computed
	// over the exact path sent, and a mismatched slash yields a 401.
	for _, req := range f.requests {
		if strings.HasPrefix(req, "GET /api/v2/users/") {
			return
		}
	}
	t.Fatalf("no detail request with rooted path in %v", f.requests)
}

func TestEntitySurvivesDetailFailure(t *testing.T) {
	f := newRoutedServer(t, map[string]func(r *http.Request) string{
		"/api/v2/search": func(r *http.Request) string {
			return `{"data":[{"name":"jane@corp.local","objectid":"S-1-5-21-77","type":"User"}]}`
		},
	})
	svc := connectedService(t, f.url)

	// The routed server answers the detail path with an empty 200 body, so
	// the decode fails; Entity must still return identity data.
	ent, err := svc.Entity(context.Background(), "S-1-5-21-77")
	if err != nil {
		t.Fatalf("Entity: %v", err)
	}
	if ent.Name != "jane@corp.local" || len(ent.Properties) != 0 {
		t.Fatalf("entity = %+v", ent)
	}
}

func TestEntityNotFound(t *testing.T) {
	f := newRoutedServer(t, map[string]func(r *http.Request) string{
		"/api/v2/search": func(r *http.Request) string { return `{"data":[]}` },
	})
	svc := connectedService(t, f.url)

	if _, err := svc.Entity(context.Background(), "S-1-5-404"); !errors.Is(err, ErrEntityNotFound) {
		t.Fatalf("Entity = %v, want ErrEntityNotFound", err)
	}
}

func TestEntityOperationsRequireConnection(t *testing.T) {
	svc := New(t.TempDir(), nil)
	ctx := context.Background()
	if _, err := svc.SearchEntities(ctx, "x", 0, 10); !errors.Is(err, ErrNotConnected) {
		t.Errorf("SearchEntities = %v, want ErrNotConnected", err)
	}
	if _, err := svc.ListDomains(ctx); !errors.Is(err, ErrNotConnected) {
		t.Errorf("ListDomains = %v, want ErrNotConnected", err)
	}
	if _, err := svc.Entity(ctx, "S-1-5"); !errors.Is(err, ErrNotConnected) {
		t.Errorf("Entity = %v, want ErrNotConnected", err)
	}
}
