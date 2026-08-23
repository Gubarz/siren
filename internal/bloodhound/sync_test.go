package bloodhound

import (
	"context"
	"net/http"
	"sync"
	"testing"
	"time"

	"siren/internal/bus"
)

type recordingBus struct {
	mu     sync.Mutex
	events []bus.Event
}

func (b *recordingBus) Publish(ev bus.Event) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.events = append(b.events, ev)
}

func (b *recordingBus) Subscribe(types []string, h bus.Handler) func() { return func() {} }

func (b *recordingBus) ofType(t string) []bus.Event {
	b.mu.Lock()
	defer b.mu.Unlock()
	var out []bus.Event
	for _, ev := range b.events {
		if ev.Type == t {
			out = append(out, ev)
		}
	}
	return out
}

func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", what)
}

func connectedServiceWithBus(t *testing.T, srvURL string, b bus.Bus) *Service {
	t.Helper()
	overrideFactory(t, srvURL)
	svc := New(t.TempDir(), b)
	if err := svc.SaveConfig(Config{ServerURL: srvURL, TokenID: "id", TokenKey: "key"}); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}
	if err := svc.Connect(context.Background()); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	return svc
}

func TestSyncPublishesWhileConnected(t *testing.T) {
	f := newRoutedServer(t, map[string]func(r *http.Request) string{
		"/api/v2/available-domains": func(r *http.Request) string {
			return `{"data":[{"id":"S-1-5-21-123","name":"corp.local","type":"ActiveDirectory","collected":true}]}`
		},
		"/api/v2/search": func(r *http.Request) string {
			return `{"data":[{"name":"JANE@CORP.LOCAL","objectid":"S-1-5-21-77","type":"User"}]}`
		},
		"/api/v2/graphs/cypher": func(r *http.Request) string {
			return threeNodePath
		},
	})
	b := &recordingBus{}
	svc := connectedServiceWithBus(t, f.url, b)

	refs := []AgentRef{{ID: "a1", Username: "jane"}}
	if _, err := svc.Correlate(context.Background(), refs); err != nil {
		t.Fatalf("Correlate: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go svc.StartSync(ctx, 20*time.Millisecond)

	waitFor(t, "bloodhound.synced event", func() bool { return len(b.ofType("bloodhound.synced")) >= 1 })

	ev := b.ofType("bloodhound.synced")[0]
	payload, ok := ev.Payload.(SyncedDTO)
	if !ok {
		t.Fatalf("synced payload = %T, want SyncedDTO", ev.Payload)
	}
	if len(payload.Domains) != 1 || payload.Domains[0].Name != "corp.local" {
		t.Fatalf("synced domains = %+v", payload.Domains)
	}
	if payload.Enrichments["a1"].Entity.ObjectID != "S-1-5-21-77" {
		t.Fatalf("synced enrichments = %+v", payload.Enrichments)
	}
}

func TestSyncSilentWhenDisconnected(t *testing.T) {
	b := &recordingBus{}
	svc := New(t.TempDir(), b)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go svc.StartSync(ctx, 10*time.Millisecond)

	time.Sleep(150 * time.Millisecond)
	if got := len(b.ofType("bloodhound.synced")); got != 0 {
		t.Fatalf("synced events while disconnected = %d, want 0", got)
	}
}

func TestStatusAndEnrichmentEvents(t *testing.T) {
	f := newRoutedServer(t, map[string]func(r *http.Request) string{
		"/api/v2/search": func(r *http.Request) string {
			return `{"data":[{"name":"JANE@CORP.LOCAL","objectid":"S-1-5-21-77","type":"User"}]}`
		},
		"/api/v2/graphs/cypher": func(r *http.Request) string {
			return threeNodePath
		},
	})
	b := &recordingBus{}
	svc := connectedServiceWithBus(t, f.url, b)

	if _, err := svc.Correlate(context.Background(), []AgentRef{{ID: "a1", Username: "jane"}}); err != nil {
		t.Fatalf("Correlate: %v", err)
	}

	statuses := b.ofType("bloodhound.status")
	if len(statuses) < 1 {
		t.Fatal("no bloodhound.status event after connect")
	}
	st, ok := statuses[len(statuses)-1].Payload.(Status)
	if !ok || !st.Connected {
		t.Fatalf("status event payload = %+v", statuses[len(statuses)-1].Payload)
	}

	svc.Disconnect()
	statuses = b.ofType("bloodhound.status")
	if st, _ := statuses[len(statuses)-1].Payload.(Status); st.Connected {
		t.Fatalf("disconnect status event should report disconnected: %+v", st)
	}

	if got := len(b.ofType("bloodhound.enrichment")); got != 1 {
		t.Fatalf("enrichment events = %d, want 1", got)
	}
}
