package bloodhound

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// newCypherStatusServer answers the cypher endpoint with status/body and every
// other path with an empty 200 JSON object.
func newCypherStatusServer(t *testing.T, status int, body string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/api/v2/graphs/cypher" {
			w.WriteHeader(status)
			_, _ = w.Write([]byte(body))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	t.Cleanup(srv.Close)
	return srv
}

// assertQueryShape checks that q contains every wanted fragment and none of
// the rejected ones.
func assertQueryShape(t *testing.T, q string, wants, rejects []string) {
	t.Helper()
	for _, want := range wants {
		if !strings.Contains(q, want) {
			t.Errorf("query missing %q in %q", want, q)
		}
	}
	for _, reject := range rejects {
		if strings.Contains(q, reject) {
			t.Errorf("query must not contain %q: %q", reject, q)
		}
	}
}

// runPipelineFailure starts a collection, injects a failure into runner or
// fetcher, and asserts the pipeline reports StageFailed with wantErr.
func runPipelineFailure(t *testing.T, wantErr string, inject func(*fakeRunner, *fakeFetcher)) {
	t.Helper()
	_, srv := newFakeIngestServer(t, `{"id":1,"status":0,"created_at":"2026-08-22T12:00:00Z"}`)
	b := &recordingBus{}
	_, cr, _, runner, fetcher, _ := newTestRunner(t, srv.URL, b)
	inject(runner, fetcher)

	id, err := cr.Start(context.Background(), "sess-1", "session", "windows", defaultOpts())
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	waitForState(t, cr, id, StageFailed)
	st, _ := cr.Status(id)
	if st.Stage != StageFailed || !strings.Contains(st.Err, wantErr) {
		t.Fatalf("state = %+v", st)
	}
}
