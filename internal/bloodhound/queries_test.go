package bloodhound

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
)

func TestAttackPathsCypherRejectsInvalidObjectIDs(t *testing.T) {
	for _, id := range []string{
		"",
		"S-1-5'-x",
		"S-1-5-x; DROP",
		"S-1-5-21-123\nMATCH",
		"not-a-sid",
	} {
		if _, err := AttackPathsCypher(id, 5); err == nil {
			t.Errorf("AttackPathsCypher(%q) = nil error, want rejection", id)
		}
	}
}

func TestAttackPathsCypherAcceptsSIDsAndGUIDs(t *testing.T) {
	for _, id := range []string{
		"S-1-5-21-111",
		"S-1-5-21-111-222",
		"01234567-89ab-cdef-0123-456789abcdef",
	} {
		q, err := AttackPathsCypher(id, 5)
		if err != nil {
			t.Errorf("AttackPathsCypher(%q) = %v, want nil", id, err)
		}
		if !strings.Contains(q, id) {
			t.Errorf("query %q missing object id %q", q, id)
		}
	}
}

func TestAttackPathsCypherClampsAndRestricts(t *testing.T) {
	q, err := AttackPathsCypher("S-1-5-21-111", 0)
	if err != nil {
		t.Fatalf("AttackPathsCypher: %v", err)
	}
	if !strings.Contains(q, "LIMIT 5") {
		t.Errorf("default maxPaths should clamp to 5, got %q", q)
	}
	q, err = AttackPathsCypher("S-1-5-21-111", 50)
	if err != nil {
		t.Fatalf("AttackPathsCypher: %v", err)
	}
	if !strings.Contains(q, "LIMIT 20") {
		t.Errorf("maxPaths > 20 should clamp to 20, got %q", q)
	}
	q, _ = AttackPathsCypher("S-1-5-21-111", 7)
	if !strings.Contains(q, "LIMIT 7") {
		t.Errorf("maxPaths 7 should pass through, got %q", q)
	}
	for _, want := range []string{
		"shortestPath", "admin_tier_0",
		// Edge restriction lives in the relationship-type alternation (the
		// server rejects all(r IN relationships(p) ...) predicates).
		"[:AdminTo|HasSession", "|MemberOf|", "|DCSync|", "|GPLink|TrustedBy*1..15]",
		// The tier-zero target is matched in its own clause so shortestPath
		// never receives start == end rows.
		"MATCH (t) WHERE t <> s",
	} {
		if !strings.Contains(q, want) {
			t.Errorf("AttackPathsCypher missing %q in %q", want, q)
		}
	}
	if strings.Contains(q, "relationships(p)") {
		t.Errorf("query must not use relationships(p) predicates: %q", q)
	}
}

func TestEntityAttackPathsRejectsBeforeHTTP(t *testing.T) {
	f := newRoutedServer(t, map[string]func(r *http.Request) string{
		"/api/v2/graphs/cypher": func(r *http.Request) string { return `{"data":{"nodes":{},"edges":[]}}` },
	})
	svc := connectedService(t, f.url)

	_, err := svc.EntityAttackPaths(context.Background(), "S-1-5'-x", 5)
	if err == nil {
		t.Fatal("EntityAttackPaths with invalid object id should fail")
	}
	for _, req := range f.requests {
		if strings.HasPrefix(req, "POST /api/v2/graphs/cypher") {
			t.Fatalf("no cypher requests expected for invalid input, got %v", f.requests)
		}
	}
}

func TestEntityAttackPathsSendsRestrictedQuery(t *testing.T) {
	var captured string
	f := newRoutedServer(t, map[string]func(r *http.Request) string{
		"/api/v2/graphs/cypher": func(r *http.Request) string {
			buf := make([]byte, r.ContentLength)
			_, _ = r.Body.Read(buf)
			captured = string(buf)
			return `{"data":{"nodes":{
				"u1":{"label":"jane@corp.local","kind":"User","isTierZero":false,"isOwnedObject":true},
				"g1":{"label":"DOMAIN ADMINS@corp.local","kind":"Group","isTierZero":true}},
				"edges":[{"source":"u1","target":"g1","kind":"MemberOf"}]}}`
		},
	})
	svc := connectedService(t, f.url)

	graph, err := svc.EntityAttackPaths(context.Background(), "S-1-5-21-1234", 5)
	if err != nil {
		t.Fatalf("EntityAttackPaths: %v", err)
	}
	// The captured body is the JSON-encoded cypher request; the raw SID and
	// LIMIT survive encoding untouched.
	if !strings.Contains(captured, "S-1-5-21-1234") {
		t.Fatalf("sent query missing object id: %s", captured)
	}
	if !strings.Contains(captured, "LIMIT 5") {
		t.Fatalf("sent query missing limit: %s", captured)
	}
	if len(graph.Nodes) != 2 || len(graph.Edges) != 1 {
		t.Fatalf("graph = %+v", graph)
	}
	var owned bool
	for _, n := range graph.Nodes {
		if n.ID == "u1" && n.Owned {
			owned = true
		}
	}
	if !owned {
		t.Fatalf("projected node should carry owned flag: %+v", graph.Nodes)
	}
}

func TestEntityAttackPathsRequiresConnection(t *testing.T) {
	svc := New(t.TempDir(), nil)
	if _, err := svc.EntityAttackPaths(context.Background(), "S-1-5-21-1", 5); !errors.Is(err, ErrNotConnected) {
		t.Fatalf("EntityAttackPaths = %v, want ErrNotConnected", err)
	}
}

func TestCommunityCypher(t *testing.T) {
	for _, kind := range []CommunityKind{
		CommunityKerberoastable, CommunityASREP, CommunityDCSync, CommunityUnconstrained,
	} {
		q, ok := CommunityCypher(kind)
		if !ok || q == "" {
			t.Errorf("CommunityCypher(%q) = (%q,%v)", kind, q, ok)
		}
	}
	if _, ok := CommunityCypher("bogus"); ok {
		t.Error("unknown kind should report !ok")
	}
}
