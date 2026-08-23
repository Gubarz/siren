package bloodhound

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

// fakeIngestServer emulates the collection-upload endpoints with a mutable
// job record, recording what the client sent.
type fakeIngestServer struct {
	mu      sync.Mutex
	job     string // JSON for the job record served by start/list
	status  int    // job status int served by list scans
	uploads []fakeUpload
	reqs    int
}

type fakeUpload struct {
	jobID       string
	contentType string
	name        string
	body        string
}

func (f *fakeIngestServer) setStatus(s int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.status = s
}

func (f *fakeIngestServer) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.reqs
}

func (f *fakeIngestServer) jobJSON() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return strings.Replace(f.job, `"status":0`, `"status":`+strconv.Itoa(f.status), 1)
}

func newFakeIngestServer(t *testing.T, jobJSON string) (*fakeIngestServer, *httptest.Server) {
	t.Helper()
	f := &fakeIngestServer{job: jobJSON}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		f.reqs++
		f.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v2/file-upload/start":
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"data":{"id":1,"status":0}}`))
		case r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/api/v2/file-upload/") && strings.HasSuffix(r.URL.Path, "/end"):
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))
		case r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/api/v2/file-upload/"):
			buf := make([]byte, r.ContentLength)
			_, _ = r.Body.Read(buf)
			f.mu.Lock()
			f.uploads = append(f.uploads, fakeUpload{
				jobID:       strings.TrimPrefix(r.URL.Path, "/api/v2/file-upload/"),
				contentType: r.Header.Get("Content-Type"),
				name:        r.Header.Get("X-File-Upload-Name"),
				body:        string(buf),
			})
			f.mu.Unlock()
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/completed-tasks"):
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"data":[{"file_name":"sharphound.zip","errors":[]},{"file_name":"bad.zip","errors":["unparseable"]}]}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v2/file-upload":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"data":[` + f.jobJSON() + `]}`))
		default:
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))
		}
	}))
	t.Cleanup(srv.Close)
	return f, srv
}

func TestIngestBytesUploadsWithContentType(t *testing.T) {
	f, srv := newFakeIngestServer(t, `{"id":1,"status":0,"created_at":"2026-08-22T12:00:00Z","total_files":1}`)
	svc := connectedService(t, srv.URL)

	job, err := svc.IngestBytes(context.Background(), "sharphound.zip", "application/zip", []byte("PK\x03\x04"))
	if err != nil {
		t.Fatalf("IngestBytes: %v", err)
	}
	if job.ID != 1 || job.Status != "ready" {
		t.Fatalf("job = %+v", job)
	}
	if len(f.uploads) != 1 {
		t.Fatalf("uploads = %d, want 1", len(f.uploads))
	}
	u := f.uploads[0]
	if u.jobID != "1" || u.contentType != "application/zip" || u.name != "sharphound.zip" || u.body != "PK\x03\x04" {
		t.Fatalf("upload = %+v", u)
	}
}

func TestIngestJobMergesCompletedTasks(t *testing.T) {
	_, srv := newFakeIngestServer(t, `{"id":1,"status":2,"created_at":"2026-08-22T12:00:00Z","total_files":2,"failed_files":1}`)
	svc := connectedService(t, srv.URL)

	job, err := svc.IngestJob(context.Background(), 1)
	if err != nil {
		t.Fatalf("IngestJob: %v", err)
	}
	if job.Status != "complete" || job.Total != 2 || job.Failed != 1 {
		t.Fatalf("job = %+v", job)
	}
	if len(job.Files) != 2 || job.Files[0].Name != "sharphound.zip" || len(job.Files[1].Errors) != 1 {
		t.Fatalf("files = %+v", job.Files)
	}
}

func TestWatchIngestJobCompletes(t *testing.T) {
	f, srv := newFakeIngestServer(t, `{"id":1,"status":0,"created_at":"2026-08-22T12:00:00Z"}`)
	f.setStatus(6)
	b := &recordingBus{}
	svc := connectedServiceWithBus(t, srv.URL, b)

	done := make(chan error, 1)
	go func() {
		done <- svc.WatchIngestJob(context.Background(), 1, 10*time.Millisecond)
	}()

	// First polls observe "ingesting" (6), then flip to complete (2).
	waitFor(t, "watch progress event", func() bool {
		return len(b.ofType("bloodhound.ingest.job.progress")) >= 1
	})
	f.setStatus(2)

	if err := <-done; err != nil {
		t.Fatalf("WatchIngestJob: %v", err)
	}
	if len(b.ofType("bloodhound.ingest.job.completed")) != 1 {
		t.Fatalf("completed events = %d, want 1", len(b.ofType("bloodhound.ingest.job.completed")))
	}
}

func TestWatchIngestJobFailurePublishesFailed(t *testing.T) {
	f, srv := newFakeIngestServer(t, `{"id":1,"status":0,"created_at":"2026-08-22T12:00:00Z"}`)
	f.setStatus(5)
	b := &recordingBus{}
	svc := connectedServiceWithBus(t, srv.URL, b)

	err := svc.WatchIngestJob(context.Background(), 1, 5*time.Millisecond)
	if err == nil {
		t.Fatal("WatchIngestJob should return an error for failed jobs")
	}
	if len(b.ofType("bloodhound.ingest.job.failed")) != 1 {
		t.Fatalf("failed events = %d, want 1", len(b.ofType("bloodhound.ingest.job.failed")))
	}
	if len(b.ofType("bloodhound.ingest.job.completed")) != 0 {
		t.Fatal("failed jobs must not publish completed")
	}
}

func TestWatchIngestJobInvalidatesCorrelation(t *testing.T) {
	f, srv := newFakeIngestServer(t, `{"id":1,"status":0,"created_at":"2026-08-22T12:00:00Z"}`)
	f.setStatus(2)
	b := &recordingBus{}
	svc := connectedServiceWithBus(t, srv.URL, b)
	refs := []AgentRef{{ID: "a1", Username: "jane"}}

	if _, err := svc.Correlate(context.Background(), refs); err != nil {
		t.Fatalf("Correlate: %v", err)
	}
	first := f.count()
	if _, err := svc.Correlate(context.Background(), refs); err != nil {
		t.Fatalf("Correlate: %v", err)
	}
	if f.count() != first {
		t.Fatal("second Correlate should hit the cache")
	}

	if err := svc.WatchIngestJob(context.Background(), 1, 5*time.Millisecond); err != nil {
		t.Fatalf("WatchIngestJob: %v", err)
	}
	if _, err := svc.Correlate(context.Background(), refs); err != nil {
		t.Fatalf("Correlate: %v", err)
	}
	if f.count() == first {
		t.Fatal("ingest completion should invalidate the correlation cache")
	}
}

func TestIngestRequiresConnection(t *testing.T) {
	svc := New(t.TempDir(), nil)
	ctx := context.Background()
	if _, err := svc.IngestBytes(ctx, "x.zip", "application/zip", []byte("x")); err == nil {
		t.Fatal("IngestBytes without connection should fail")
	}
	if _, err := svc.IngestJobs(ctx); err == nil {
		t.Fatal("IngestJobs without connection should fail")
	}
	if _, err := svc.IngestJob(ctx, 1); err == nil {
		t.Fatal("IngestJob without connection should fail")
	}
}
