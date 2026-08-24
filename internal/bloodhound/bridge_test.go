package bloodhound

import (
	"context"
	"errors"
	"net/http"
	"testing"
)

func TestKerberoastTargetsExtractsUsers(t *testing.T) {
	f := newRoutedServer(t, map[string]func(r *http.Request) string{
		"/api/v2/graphs/cypher": func(r *http.Request) string {
			return `{"data":{"nodes":{
				"u1":{"label":"svc_sql@corp.local","kind":"User","objectId":"S-1-5-21-1","isOwnedObject":false},
				"u2":{"label":"svc_web@corp.local","kind":"User","objectId":"S-1-5-21-2"},
				"u3":{"label":"dup@corp.local","kind":"User","objectId":"S-1-5-21-1"},
				"c1":{"label":"PC1.CORP.LOCAL","kind":"Computer","objectId":"S-1-5-21-9"}},
				"edges":[]}}`
		},
	})
	svc := connectedService(t, f.url)

	got, err := svc.KerberoastTargets(context.Background())
	if err != nil {
		t.Fatalf("KerberoastTargets: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2 (computer filtered, dup deduped): %+v", len(got), got)
	}
	ids := map[string]bool{}
	for _, target := range got {
		if target.Kind != "User" || target.ObjectID == "" || target.Name == "" {
			t.Fatalf("target = %+v", target)
		}
		ids[target.ObjectID] = true
	}
	if !ids["S-1-5-21-1"] || !ids["S-1-5-21-2"] {
		t.Fatalf("object ids = %v, want S-1-5-21-1 and S-1-5-21-2", ids)
	}
}

func TestKerberoastTargetsEmptyGraph(t *testing.T) {
	f := newRoutedServer(t, map[string]func(r *http.Request) string{
		"/api/v2/graphs/cypher": func(r *http.Request) string {
			return `{"data":{"nodes":{},"edges":[]}}`
		},
	})
	svc := connectedService(t, f.url)

	got, err := svc.KerberoastTargets(context.Background())
	if err != nil {
		t.Fatalf("KerberoastTargets: %v", err)
	}
	if got == nil || len(got) != 0 {
		t.Fatalf("empty graph should yield empty non-nil slice, got %+v", got)
	}
}

func TestKerberoastTargetsRequiresConnection(t *testing.T) {
	svc := New(t.TempDir(), nil)
	if _, err := svc.KerberoastTargets(context.Background()); !errors.Is(err, ErrNotConnected) {
		t.Fatalf("KerberoastTargets = %v, want ErrNotConnected", err)
	}
}
