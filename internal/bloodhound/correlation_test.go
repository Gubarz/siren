package bloodhound

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
)

// correlationServer serves search + cypher bodies with call counting.
type correlationServer struct {
	f          *routedServer
	searchBody func(q string) string
	cypherBody string
}

func newCorrelationServer(t *testing.T, searchBody func(q string) string, cypherBody string) *correlationServer {
	t.Helper()
	c := &correlationServer{searchBody: searchBody, cypherBody: cypherBody}
	c.f = newRoutedServer(t, map[string]func(r *http.Request) string{
		"/api/v2/search": func(r *http.Request) string {
			return searchBody(r.URL.Query().Get("q"))
		},
		"/api/v2/graphs/cypher": func(r *http.Request) string {
			return cypherBody
		},
	})
	return c
}

func (c *correlationServer) searchCalls() int {
	n := 0
	for _, req := range c.f.requests {
		if strings.HasPrefix(req, "GET /api/v2/search?") {
			n++
		}
	}
	return n
}

func (c *correlationServer) cypherCalls() int {
	n := 0
	for _, req := range c.f.requests {
		if strings.HasPrefix(req, "POST /api/v2/graphs/cypher") {
			n++
		}
	}
	return n
}

const threeNodePath = `{"data":{"nodes":{
	"u1":{"label":"jane@corp.local","kind":"User","isTierZero":false,"isOwnedObject":true},
	"g1":{"label":"IT ADMINS@corp.local","kind":"Group","isTierZero":false},
	"dc1":{"label":"DC1.CORP.LOCAL","kind":"Computer","isTierZero":true}},
	"edges":[
		{"source":"u1","target":"g1","kind":"MemberOf"},
		{"source":"g1","target":"dc1","kind":"AdminTo"}]}}`

func computerHit(q string) string {
	return `{"data":[{"name":"PC1.CORP.LOCAL","objectid":"S-1-5-21-999","type":"Computer"}]}`
}

func TestCorrelatePrefersComputerOverContainerTwins(t *testing.T) {
	// CE search can return a Container twin for a hostname (e.g.
	// DC01@DOMAIN vs DC01.DOMAIN); correlation must resolve the Computer.
	c := newCorrelationServer(t, func(q string) string {
		return `{"data":[
			{"name":"PC01-CA@CORP.LOCAL","objectid":"G1","type":"EnterpriseCA"},
			{"name":"PC01.CORP.LOCAL","objectid":"S-1-5-21-1000","type":"Computer"},
			{"name":"PC01@CORP.LOCAL","objectid":"G2","type":"Container"}]}`
	}, threeNodePath)
	svc := connectedService(t, c.f.url)

	got, err := svc.Correlate(context.Background(), []AgentRef{{ID: "a1", Hostname: "PC01.CORP.LOCAL"}})
	if err != nil {
		t.Fatalf("Correlate: %v", err)
	}
	e := got["a1"]
	if e.Entity.Kind != "Computer" || e.Entity.ObjectID != "S-1-5-21-1000" {
		t.Fatalf("enrichment = %+v, want the Computer twin", e.Entity)
	}
}

func userHit(q string) string {
	return `{"data":[{"name":"JANE@CORP.LOCAL","objectid":"S-1-5-21-77","type":"User","system_tags":"owned"}]}`
}

func TestCorrelateBatchesSharedCandidates(t *testing.T) {
	c := newCorrelationServer(t, computerHit, threeNodePath)
	svc := connectedService(t, c.f.url)

	refs := []AgentRef{
		{ID: "a1", Hostname: "PC1.corp.local"},
		{ID: "a2", Hostname: "pc1"},
	}
	got, err := svc.Correlate(context.Background(), refs)
	if err != nil {
		t.Fatalf("Correlate: %v", err)
	}
	if c.searchCalls() != 1 {
		t.Fatalf("search calls = %d, want 1 (candidates must be batched)", c.searchCalls())
	}
	if c.cypherCalls() != 1 {
		t.Fatalf("cypher calls = %d, want 1", c.cypherCalls())
	}
	for _, id := range []string{"a1", "a2"} {
		e, ok := got[id]
		if !ok {
			t.Fatalf("no enrichment for %s", id)
		}
		if e.Entity.ObjectID != "S-1-5-21-999" || e.Entity.Name != "PC1.CORP.LOCAL" {
			t.Fatalf("%s enrichment entity = %+v", id, e.Entity)
		}
		if e.DistanceToTierZero != 2 {
			t.Fatalf("%s distance = %d, want 2", id, e.DistanceToTierZero)
		}
		if len(e.Paths) != 3 {
			t.Fatalf("%s paths = %+v", id, e.Paths)
		}
		if e.Paths[0].ID != "u1" || e.Paths[2].ID != "dc1" {
			t.Fatalf("%s path order = %+v", id, e.Paths)
		}
	}
}

func TestCorrelateUsernameSamAccount(t *testing.T) {
	var searched []string
	c := newCorrelationServer(t, func(q string) string {
		searched = append(searched, q)
		return userHit(q)
	}, threeNodePath)
	svc := connectedService(t, c.f.url)

	got, err := svc.Correlate(context.Background(), []AgentRef{{ID: "a1", Username: `CORP\jane`}})
	if err != nil {
		t.Fatalf("Correlate: %v", err)
	}
	if len(searched) != 1 || searched[0] != "jane" {
		t.Fatalf("search queries = %v, want [jane]", searched)
	}
	e := got["a1"]
	if e.Entity.ObjectID != "S-1-5-21-77" || !e.Owned {
		t.Fatalf("enrichment = %+v", e)
	}
}

func TestCorrelateFallsBackToHostnameWhenUsernameUnresolved(t *testing.T) {
	var searched []string
	c := newCorrelationServer(t, func(q string) string {
		searched = append(searched, q)
		if q == "srv01" {
			return `{"data":[{"name":"SRV01.CORP.LOCAL","objectid":"S-1-5-21-4615","type":"Computer"}]}`
		}
		return `{"data":[]}`
	}, threeNodePath)
	svc := connectedService(t, c.f.url)

	ref := AgentRef{ID: "a1", Hostname: "SRV01", Username: `CORP\SRV01$`}
	got, err := svc.Correlate(context.Background(), []AgentRef{ref})
	if err != nil {
		t.Fatalf("Correlate: %v", err)
	}
	e := got["a1"]
	if e.Entity.ObjectID != "S-1-5-21-4615" || e.Entity.Kind != "Computer" {
		t.Fatalf("enrichment = %+v, want the hostname computer fallback", e.Entity)
	}
	if len(searched) != 2 || searched[0] != "srv01$" || searched[1] != "srv01" {
		t.Fatalf("searched = %v, want username first then hostname", searched)
	}
}

func TestCorrelateUnreachableEntity(t *testing.T) {
	c := newCorrelationServer(t, userHit, `{"data":{"nodes":{},"edges":[]}}`)
	svc := connectedService(t, c.f.url)

	got, err := svc.Correlate(context.Background(), []AgentRef{{ID: "a1", Username: "jane"}})
	if err != nil {
		t.Fatalf("Correlate: %v", err)
	}
	e := got["a1"]
	if e.DistanceToTierZero != -1 || len(e.Paths) != 0 {
		t.Fatalf("unreachable enrichment = %+v", e)
	}
}

func TestCorrelateNotFoundSkipsPathQuery(t *testing.T) {
	c := newCorrelationServer(t, func(q string) string { return `{"data":[]}` }, threeNodePath)
	svc := connectedService(t, c.f.url)

	got, err := svc.Correlate(context.Background(), []AgentRef{{ID: "a1", Hostname: "ghost"}})
	if err != nil {
		t.Fatalf("Correlate: %v", err)
	}
	e := got["a1"]
	if e.Entity.ObjectID != "" || e.DistanceToTierZero != -1 {
		t.Fatalf("not-found enrichment = %+v", e)
	}
	if c.cypherCalls() != 0 {
		t.Fatalf("cypher calls = %d, want 0 for unresolved agents", c.cypherCalls())
	}
}

func TestCorrelateCacheAndInvalidate(t *testing.T) {
	c := newCorrelationServer(t, userHit, threeNodePath)
	svc := connectedService(t, c.f.url)
	refs := []AgentRef{{ID: "a1", Username: "jane"}}

	if _, err := svc.Correlate(context.Background(), refs); err != nil {
		t.Fatalf("Correlate: %v", err)
	}
	afterFirst := c.searchCalls() + c.cypherCalls()

	// Second call with identical refs must hit the cache: no new HTTP.
	if _, err := svc.Correlate(context.Background(), refs); err != nil {
		t.Fatalf("Correlate: %v", err)
	}
	if afterSecond := c.searchCalls() + c.cypherCalls(); afterSecond != afterFirst {
		t.Fatalf("cached call made requests: %d -> %d", afterFirst, afterSecond)
	}

	// Changed hostname/username is a new key.
	if _, err := svc.Correlate(context.Background(), []AgentRef{{ID: "a1", Username: "other"}}); err != nil {
		t.Fatalf("Correlate: %v", err)
	}

	svc.InvalidateCorrelation()
	if _, err := svc.Correlate(context.Background(), refs); err != nil {
		t.Fatalf("Correlate: %v", err)
	}
	if afterInvalidate := c.searchCalls() + c.cypherCalls(); afterInvalidate <= afterFirst {
		t.Fatalf("invalidation should force refetch: %d -> %d", afterFirst, afterInvalidate)
	}
}

func TestCorrelateRequiresConnection(t *testing.T) {
	svc := New(t.TempDir(), nil)
	if _, err := svc.Correlate(context.Background(), []AgentRef{{ID: "a1"}}); !errors.Is(err, ErrNotConnected) {
		t.Fatalf("Correlate = %v, want ErrNotConnected", err)
	}
}
