package bloodhound

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

// ownedServer fakes the asset-group-tags endpoints the SDK's owned helpers
// hit, recording mutations for assertions.
type ownedServer struct {
	mu sync.Mutex

	selectors string // JSON body for GET /asset-group-tags/{id}/selectors

	posts    []string // recorded POST paths
	postBody string   // last recorded POST body
	deletes  []string // recorded DELETE paths
	created  string   // JSON returned for POST (201)
	tagCount int
}

func (s *ownedServer) handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		defer s.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v2/asset-group-tags":
			_, _ = w.Write([]byte(`{"data":{"tags":[{"id":2,"type":3,"name":"Owned"},{"id":1,"type":1,"name":"Tier Zero"}]}}`))
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/api/v2/asset-group-tags/") && strings.HasSuffix(r.URL.Path, "/selectors"):
			if s.selectors == "" {
				_, _ = w.Write([]byte(`{"data":{"selectors":[]}}`))
				return
			}
			_, _ = w.Write([]byte(s.selectors))
		case r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/api/v2/asset-group-tags/") && strings.HasSuffix(r.URL.Path, "/selectors"):
			body, _ := io.ReadAll(r.Body)
			_ = r.Body.Close()
			s.posts = append(s.posts, r.URL.Path)
			s.postBody = string(body)
			w.WriteHeader(http.StatusCreated)
			resp := s.created
			if resp == "" {
				resp = `{"data":{"id":9,"asset_group_tag_id":2,"name":"owned-seed","seeds":[{"type":1,"value":"S-1-5-21-999"}]}}`
			}
			_, _ = w.Write([]byte(resp))
		case r.Method == http.MethodDelete:
			s.deletes = append(s.deletes, r.URL.Path)
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))
		}
	})
}

func newOwnedService(t *testing.T) (*Service, *ownedServer) {
	t.Helper()
	state := &ownedServer{}
	srv := httptest.NewServer(state.handler())
	t.Cleanup(srv.Close)
	overrideFactory(t, srv.URL)
	svc := New(t.TempDir(), nil)
	if err := svc.SaveConfig(Config{ServerURL: srv.URL, TokenID: "id", TokenKey: "key"}); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}
	return svc, state
}

func TestMarkOwnedAddsSelectorSeed(t *testing.T) {
	svc, state := newOwnedService(t)

	if err := svc.MarkOwned(context.Background(), "S-1-5-21-999"); err != nil {
		t.Fatalf("MarkOwned: %v", err)
	}
	state.mu.Lock()
	defer state.mu.Unlock()
	if len(state.posts) != 1 || state.posts[0] != "/api/v2/asset-group-tags/2/selectors" {
		t.Fatalf("posts = %v, want one selector POST on the Owned tag", state.posts)
	}
}

func TestMarkOwnedPostsObjectIDSeedBody(t *testing.T) {
	svc, state := newOwnedService(t)

	if err := svc.MarkOwned(context.Background(), "S-1-5-21-999"); err != nil {
		t.Fatalf("MarkOwned: %v", err)
	}
	state.mu.Lock()
	body := state.postBody
	state.mu.Unlock()
	var decoded struct {
		Name  string `json:"name"`
		Seeds []struct {
			Type  int    `json:"type"`
			Value string `json:"value"`
		} `json:"seeds"`
	}
	if err := json.Unmarshal([]byte(body), &decoded); err != nil {
		t.Fatalf("POST body %q: %v", body, err)
	}
	if len(decoded.Seeds) != 1 || decoded.Seeds[0].Type != 1 || decoded.Seeds[0].Value != "S-1-5-21-999" {
		t.Fatalf("body = %+v, want one object-id seed", decoded)
	}
}

func TestUnmarkOwnedDeletesSoleSeedSelector(t *testing.T) {
	svc, state := newOwnedService(t)
	state.selectors = `{"data":{"selectors":[{"id":7,"asset_group_tag_id":2,"name":"S-1-5-21-999","seeds":[{"type":1,"value":"S-1-5-21-999"}]}]}}`

	if err := svc.UnmarkOwned(context.Background(), "S-1-5-21-999"); err != nil {
		t.Fatalf("UnmarkOwned: %v", err)
	}
	state.mu.Lock()
	defer state.mu.Unlock()
	if len(state.deletes) != 1 || state.deletes[0] != "/api/v2/asset-group-tags/2/selectors/7" {
		t.Fatalf("deletes = %v, want the selector holding only this seed", state.deletes)
	}
}

func TestMarkOwnedInvalidatesCorrelationCache(t *testing.T) {
	state := &ownedServer{}
	searchCalls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/api/v2/search":
			searchCalls++
			_, _ = w.Write([]byte(`{"data":[{"name":"PC1.CORP.LOCAL","objectid":"S-1-5-21-999","type":"Computer"}]}`))
		case r.URL.Path == "/api/v2/graphs/cypher":
			_, _ = w.Write([]byte(`{"data":{"nodes":{},"edges":[]}}`))
		default:
			state.handler().ServeHTTP(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	overrideFactory(t, srv.URL)
	svc := New(t.TempDir(), nil)
	if err := svc.SaveConfig(Config{ServerURL: srv.URL, TokenID: "id", TokenKey: "key"}); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}

	refs := []AgentRef{{ID: "a1", Hostname: "PC1"}}
	if _, err := svc.Correlate(context.Background(), refs); err != nil {
		t.Fatalf("Correlate: %v", err)
	}
	if _, err := svc.Correlate(context.Background(), refs); err != nil {
		t.Fatalf("Correlate (cached): %v", err)
	}
	if searchCalls != 1 {
		t.Fatalf("search calls = %d, want 1 (second must hit cache)", searchCalls)
	}

	if err := svc.MarkOwned(context.Background(), "S-1-5-21-999"); err != nil {
		t.Fatalf("MarkOwned: %v", err)
	}
	if _, err := svc.Correlate(context.Background(), refs); err != nil {
		t.Fatalf("Correlate (after mark): %v", err)
	}
	if searchCalls != 2 {
		t.Fatalf("search calls after mark = %d, want 2 (cache must be invalidated)", searchCalls)
	}
}

func TestMarkOwnedRequiresConnection(t *testing.T) {
	svc := New(t.TempDir(), nil)
	if err := svc.MarkOwned(context.Background(), "S-1-5-21-999"); !errors.Is(err, ErrNotConnected) {
		t.Fatalf("MarkOwned = %v, want ErrNotConnected", err)
	}
	if err := svc.UnmarkOwned(context.Background(), "S-1-5-21-999"); !errors.Is(err, ErrNotConnected) {
		t.Fatalf("UnmarkOwned = %v, want ErrNotConnected", err)
	}
}

// TestEnrichmentOwnedFromSelectorMembership proves enrichment.owned tracks
// the Owned tag's selectors (synchronous with mark/unmark) rather than the
// search system_tags or the lagging graph members list.
func TestEnrichmentOwnedFromSelectorMembership(t *testing.T) {
	var selectors string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/api/v2/search":
			_, _ = w.Write([]byte(`{"data":[{"name":"PC1.CORP.LOCAL","objectid":"S-1-5-21-999","type":"Computer"}]}`))
		case r.URL.Path == "/api/v2/graphs/cypher":
			_, _ = w.Write([]byte(`{"data":{"nodes":{},"edges":[]}}`))
		case r.URL.Path == "/api/v2/asset-group-tags":
			_, _ = w.Write([]byte(`{"data":{"tags":[{"id":2,"type":3,"name":"Owned"}]}}`))
		case strings.HasPrefix(r.URL.Path, "/api/v2/asset-group-tags/2/selectors"):
			_, _ = w.Write([]byte(selectors))
		default:
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))
		}
	}))
	t.Cleanup(srv.Close)
	overrideFactory(t, srv.URL)
	svc := New(t.TempDir(), nil)
	if err := svc.SaveConfig(Config{ServerURL: srv.URL, TokenID: "id", TokenKey: "key"}); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}

	refs := []AgentRef{{ID: "a1", Hostname: "PC1"}}

	selectors = `{"data":{"selectors":[]}}`
	enr, err := svc.Correlate(context.Background(), refs)
	if err != nil {
		t.Fatalf("Correlate: %v", err)
	}
	if enr["a1"].Owned {
		t.Fatalf("enrichment = %+v, want not owned with no selectors", enr["a1"])
	}

	selectors = `{"data":{"selectors":[{"id":7,"asset_group_tag_id":2,"name":"S-1-5-21-999","seeds":[{"type":1,"value":"S-1-5-21-999"}]}]}}`
	svc.InvalidateCorrelation()
	enr, err = svc.Correlate(context.Background(), refs)
	if err != nil {
		t.Fatalf("Correlate: %v", err)
	}
	if !enr["a1"].Owned {
		t.Fatalf("enrichment = %+v, want owned via selector membership", enr["a1"])
	}
}
