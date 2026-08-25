package bloodhound

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// fakeQueryServer serves canned search + cypher responses and records requests.
type fakeQueryServer struct {
	searchBody string
	cypherBody string
	searchReqs []string // captured limit query params
	cypherReqs []string // captured query strings
}

func newFakeQueryServer(t *testing.T, f *fakeQueryServer) *httptest.Server {
	t.Helper()
	if f.searchBody == "" {
		f.searchBody = `{"data":[]}`
	}
	if f.cypherBody == "" {
		f.cypherBody = `{"data":{"nodes":{},"edges":[]}}`
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(r.URL.Path, "/search"):
			f.searchReqs = append(f.searchReqs, r.URL.Query().Get("limit"))
			_, _ = w.Write([]byte(f.searchBody))
		case strings.HasSuffix(r.URL.Path, "/graphs/cypher"):
			buf := make([]byte, r.ContentLength)
			_, _ = r.Body.Read(buf)
			f.cypherReqs = append(f.cypherReqs, string(buf))
			_, _ = w.Write([]byte(f.cypherBody))
		default:
			_, _ = w.Write([]byte(`{}`)) // meta/ping endpoints
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func connectedService(t *testing.T, srvURL string) *Service {
	t.Helper()
	overrideFactory(t, srvURL)
	svc := New(t.TempDir(), nil)
	if err := svc.SaveConfig(Config{ServerURL: srvURL, TokenID: "id", TokenKey: "key"}); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}
	if err := svc.Connect(context.Background()); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	return svc
}

func TestQueriesRequireConnection(t *testing.T) {
	svc := New(t.TempDir(), nil)
	ctx := context.Background()
	if _, err := svc.SearchEntities(ctx, "x", 0, 10); !errors.Is(err, ErrNotConnected) {
		t.Errorf("SearchEntities = %v, want ErrNotConnected", err)
	}
	if _, err := svc.EntityAttackPaths(ctx, "S-1-5-21-1", 5); !errors.Is(err, ErrNotConnected) {
		t.Errorf("EntityAttackPaths = %v, want ErrNotConnected", err)
	}
	if _, err := svc.CommunityQuery(ctx, CommunityDCSync); !errors.Is(err, ErrNotConnected) {
		t.Errorf("CommunityQuery = %v, want ErrNotConnected", err)
	}
}

func TestSearchEntities(t *testing.T) {
	fake := &fakeQueryServer{searchBody: `{"data":[{"name":"DA@corp.local","objectid":"S-1-5-999","type":"Group","system_tags":"admin_tier_0"}]}`}
	svc := connectedService(t, newFakeQueryServer(t, fake).URL)

	page, err := svc.SearchEntities(context.Background(), "Domain Admins", 0, 10)
	if err != nil {
		t.Fatalf("SearchEntities: %v", err)
	}
	if len(page.Entities) != 1 || page.Entities[0].Name != "DA@corp.local" || page.Entities[0].ObjectID != "S-1-5-999" {
		t.Fatalf("page %+v", page)
	}
}

func TestSearchEntitiesClampsLimit(t *testing.T) {
	cases := []struct {
		limit int
		want  string
	}{
		{-1, "25"},
		{0, "25"},
		{101, "100"},
		{1, "1"},
		{100, "100"},
	}
	for _, tc := range cases {
		fake := &fakeQueryServer{}
		svc := connectedService(t, newFakeQueryServer(t, fake).URL)
		if _, err := svc.SearchEntities(context.Background(), "x", 0, tc.limit); err != nil {
			t.Fatalf("limit %d: SearchEntities: %v", tc.limit, err)
		}
		got := fake.searchReqs[0]
		if got != tc.want {
			t.Errorf("limit %d: sent %q, want %q", tc.limit, got, tc.want)
		}
	}
}

func TestEntityAttackPaths(t *testing.T) {
	fake := &fakeQueryServer{cypherBody: `{"data":{"nodes":{
		"u1":{"label":"jane@corp.local","kind":"User","isTierZero":false},
		"g1":{"label":"DOMAIN ADMINS@corp.local","kind":"Group","isTierZero":true}},
		"edges":[{"source":"u1","target":"g1","kind":"MemberOf"}]}}`}
	svc := connectedService(t, newFakeQueryServer(t, fake).URL)

	graph, err := svc.EntityAttackPaths(context.Background(), "S-1-5-21-1234", 5)
	if err != nil {
		t.Fatalf("EntityAttackPaths: %v", err)
	}
	if len(graph.Nodes) != 2 || len(graph.Edges) != 1 {
		t.Fatalf("graph = %+v", graph)
	}
	if len(fake.cypherReqs) != 1 || !strings.Contains(fake.cypherReqs[0], "S-1-5-21-1234") {
		t.Fatalf("cypher request = %v", fake.cypherReqs)
	}
}

func TestEntityAttackPathsTreats404AsEmpty(t *testing.T) {
	// BloodHound CE returns HTTP 404 for cypher queries with zero rows;
	// the service must translate that into an empty graph, not an error.
	f := newRoutedServer(t, map[string]func(r *http.Request) string{
		"/api/v2/graphs/cypher": func(r *http.Request) string { return `` },
	})
	_ = f
	// The routed server helper always answers 200; build a raw 404 server.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v2/graphs/cypher" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"http_status":404,"errors":[{"context":"query","message":"resource not found"}]}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	t.Cleanup(srv.Close)
	svc := connectedService(t, srv.URL)

	graph, err := svc.EntityAttackPaths(context.Background(), "S-1-5-21-1234", 5)
	if err != nil {
		t.Fatalf("EntityAttackPaths = %v, want nil with empty graph", err)
	}
	if graph.Nodes == nil || graph.Edges == nil || len(graph.Nodes) != 0 || len(graph.Edges) != 0 {
		t.Fatalf("graph = %+v, want empty non-nil", graph)
	}

	// A real non-404 failure still surfaces.
	srvBad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/api/v2/graphs/cypher" {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"http_status":500,"errors":[{"context":"query","message":"boom"}]}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	t.Cleanup(srvBad.Close)
	svcBad := connectedService(t, srvBad.URL)
	if _, err := svcBad.EntityAttackPaths(context.Background(), "S-1-5-21-1234", 5); err == nil {
		t.Fatal("non-404 cypher errors must propagate")
	}
}

func TestCommunityQueryUnknownKind(t *testing.T) {
	svc := connectedService(t, newFakeQueryServer(t, &fakeQueryServer{}).URL)
	_, err := svc.CommunityQuery(context.Background(), "nope")
	if err == nil || !strings.Contains(err.Error(), "unknown community query kind") {
		t.Fatalf("err = %v, want unknown-kind error", err)
	}
}

func TestCommunityQueryRunsCypher(t *testing.T) {
	fake := &fakeQueryServer{}
	svc := connectedService(t, newFakeQueryServer(t, fake).URL)
	graph, err := svc.CommunityQuery(context.Background(), CommunityKerberoastable)
	if err != nil {
		t.Fatalf("CommunityQuery: %v", err)
	}
	if graph.Nodes == nil || graph.Edges == nil {
		t.Fatalf("expected empty non-nil graph, got %+v", graph)
	}
	if len(fake.cypherReqs) != 1 {
		t.Fatalf("expected 1 cypher request, got %d", len(fake.cypherReqs))
	}
}

func TestEntityRelationsRequireConnection(t *testing.T) {
	svc := New(t.TempDir(), nil)
	ctx := context.Background()
	if _, err := svc.EntitySessions(ctx, "S-1-5-21-1", "Computer"); !errors.Is(err, ErrNotConnected) {
		t.Errorf("EntitySessions = %v, want ErrNotConnected", err)
	}
	if _, err := svc.EntityLocalAdmins(ctx, "S-1-5-21-1", "Computer"); !errors.Is(err, ErrNotConnected) {
		t.Errorf("EntityLocalAdmins = %v, want ErrNotConnected", err)
	}
}

func TestEntityRelationsRejectInvalidIDsBeforeHTTP(t *testing.T) {
	fake := &fakeQueryServer{}
	svc := connectedService(t, newFakeQueryServer(t, fake).URL)
	if _, err := svc.EntitySessions(context.Background(), "nope", "Computer"); err == nil {
		t.Error("EntitySessions with invalid id should fail")
	}
	if _, err := svc.EntityLocalAdmins(context.Background(), "nope", "User"); err == nil {
		t.Error("EntityLocalAdmins with invalid id should fail")
	}
	if len(fake.cypherReqs) != 0 {
		t.Fatalf("no cypher requests expected, got %v", fake.cypherReqs)
	}
}

func TestEntitySessionsRunsTypedQuery(t *testing.T) {
	fake := &fakeQueryServer{cypherBody: `{"data":{"nodes":{
		"u1":{"label":"jane@corp.local","kind":"User","isTierZero":false,"isOwnedObject":true}},
		"edges":[{"source":"u1","target":"c1","kind":"HasSession"}]}}`}
	svc := connectedService(t, newFakeQueryServer(t, fake).URL)

	graph, err := svc.EntitySessions(context.Background(), "S-1-5-21-1234", "Computer")
	if err != nil {
		t.Fatalf("EntitySessions: %v", err)
	}
	if len(graph.Nodes) != 1 || len(graph.Edges) != 1 || !graph.Nodes[0].Owned {
		t.Fatalf("graph = %+v", graph)
	}
	if !strings.Contains(fake.cypherReqs[0], "c.objectid") || !strings.Contains(fake.cypherReqs[0], "S-1-5-21-1234") {
		t.Fatalf("cypher request = %s", fake.cypherReqs[0])
	}
}

func TestEntityLocalAdminsRunsExpandedQuery(t *testing.T) {
	fake := &fakeQueryServer{cypherBody: `{"data":{"nodes":{
		"g1":{"label":"IT ADMINS@corp.local","kind":"Group"}},
		"edges":[{"source":"g1","target":"c1","kind":"AdminTo"}]}}`}
	svc := connectedService(t, newFakeQueryServer(t, fake).URL)

	graph, err := svc.EntityLocalAdmins(context.Background(), "S-1-5-21-5678", "Computer")
	if err != nil {
		t.Fatalf("EntityLocalAdmins: %v", err)
	}
	if len(graph.Nodes) != 1 || len(graph.Edges) != 1 {
		t.Fatalf("graph = %+v", graph)
	}
	if !strings.Contains(fake.cypherReqs[0], "MemberOf*0..5") {
		t.Fatalf("cypher request missing group expansion: %s", fake.cypherReqs[0])
	}
}

func TestEntityRelationsTreat404AsEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/api/v2/graphs/cypher" {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"http_status":404,"errors":[{"context":"query","message":"resource not found"}]}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	t.Cleanup(srv.Close)
	svc := connectedService(t, srv.URL)

	graph, err := svc.EntitySessions(context.Background(), "S-1-5-21-1", "Computer")
	if err != nil || graph.Nodes == nil || graph.Edges == nil || len(graph.Nodes) != 0 {
		t.Fatalf("EntitySessions = (%+v, %v), want empty non-nil", graph, err)
	}
	admins, err := svc.EntityLocalAdmins(context.Background(), "S-1-5-21-1", "User")
	if err != nil || admins.Nodes == nil || admins.Edges == nil || len(admins.Edges) != 0 {
		t.Fatalf("EntityLocalAdmins = (%+v, %v), want empty non-nil", admins, err)
	}
}
