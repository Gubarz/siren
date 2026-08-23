package bloodhound

import (
	"context"
	"errors"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

type fakeSource struct {
	mu    sync.Mutex
	path  string
	sum   string
	err   error
	calls []string
}

func (f *fakeSource) Download(ctx context.Context, collector, tag string) (string, string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, collector+"@"+tag)
	if f.err != nil {
		return "", "", f.err
	}
	return f.path, f.sum, nil
}

type fakeRunner struct {
	mu       sync.Mutex
	err      error
	output   string
	commands []string
}

func (f *fakeRunner) Run(ctx context.Context, agentID, command string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.commands = append(f.commands, command)
	if f.err != nil {
		return f.output, f.err
	}
	return f.output, nil
}

type fakeFetcher struct {
	mu          sync.Mutex
	uploadErr   error
	downloadErr error
	downloads   []string
	uploads     []string
}

func (f *fakeFetcher) Upload(ctx context.Context, agentID, remotePath, localPath string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.uploads = append(f.uploads, remotePath+"<-"+localPath)
	return f.uploadErr
}

func (f *fakeFetcher) Download(ctx context.Context, agentID, remotePath, localPath string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.downloads = append(f.downloads, localPath+"<-"+remotePath)
	if f.downloadErr != nil {
		return f.downloadErr
	}
	return os.WriteFile(localPath, []byte("FAKE-COLLECTION-ZIP"), 0o644)
}

type fakeLoot struct {
	mu    sync.Mutex
	names []string
	data  [][]byte
}

func (f *fakeLoot) Archive(ctx context.Context, name string, data []byte) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.names = append(f.names, name)
	f.data = append(f.data, data)
	return nil
}

func newTestRunner(t *testing.T, srvURL string, b *recordingBus) (*Service, *CollectionRunner, *fakeSource, *fakeRunner, *fakeFetcher, *fakeLoot) {
	t.Helper()
	svc := connectedServiceWithBus(t, srvURL, b)
	source := &fakeSource{path: "/tmp/sharphound.exe", sum: "abc"}
	runner := &fakeRunner{}
	fetcher := &fakeFetcher{}
	loot := &fakeLoot{}
	cr := NewCollectionRunner(svc, source, runner, fetcher, loot)
	return svc, cr, source, runner, fetcher, loot
}

func defaultOpts() CollectionOptions {
	return CollectionOptions{
		Collector:      "sharphound",
		Methods:        []string{"Default"},
		TimeoutSeconds: 600,
		Ingest:         true,
		Loot:           true,
	}
}

func waitForState(t *testing.T, cr *CollectionRunner, id string, want Stage) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if st, ok := cr.Status(id); ok && st.Stage == want {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("collection %s never reached stage %s", id, want)
}

func TestCollectionPipelineHappyPath(t *testing.T) {
	_, srv := newFakeIngestServer(t, `{"id":1,"status":0,"created_at":"2026-08-22T12:00:00Z"}`)
	b := &recordingBus{}
	_, cr, source, runner, fetcher, loot := newTestRunner(t, srv.URL, b)

	id, err := cr.Start(context.Background(), "sess-1", "session", "windows", defaultOpts())
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	waitForState(t, cr, id, StageDone)

	st, _ := cr.Status(id)
	if st.Err != "" {
		t.Fatalf("state = %+v", st)
	}
	if st.IngestJobID != 1 {
		t.Fatalf("ingest job id = %d, want 1", st.IngestJobID)
	}
	if len(source.calls) != 1 || source.calls[0] != "sharphound@" {
		t.Fatalf("source calls = %v", source.calls)
	}
	if len(runner.commands) != 1 {
		t.Fatalf("commands = %v", runner.commands)
	}
	cmd := runner.commands[0]
	for _, want := range []string{"-c", "Default", "--zipfilename", "sharphound.exe", "C:\\Windows\\Temp"} {
		if !strings.Contains(cmd, want) {
			t.Errorf("command %q missing %q", cmd, want)
		}
	}
	if len(fetcher.uploads) != 1 || len(fetcher.downloads) != 1 {
		t.Fatalf("uploads=%v downloads=%v", fetcher.uploads, fetcher.downloads)
	}
	if len(loot.names) != 1 || !strings.HasPrefix(loot.names[0], "bloodhound-sess-1-") {
		t.Fatalf("loot names = %v", loot.names)
	}
	if len(loot.data) != 1 || len(loot.data[0]) == 0 {
		t.Fatalf("loot data = %d entries", len(loot.data))
	}
	if len(b.ofType("bloodhound.collection."+id+".done")) != 1 {
		t.Fatalf("done event missing")
	}
}

func TestCollectionPipelineStripsLoopFlag(t *testing.T) {
	_, srv := newFakeIngestServer(t, `{"id":1,"status":0,"created_at":"2026-08-22T12:00:00Z"}`)
	b := &recordingBus{}
	_, cr, _, runner, _, _ := newTestRunner(t, srv.URL, b)

	opts := defaultOpts()
	opts.Flags = []string{"--Loop", "--Stealth"}
	id, err := cr.Start(context.Background(), "sess-1", "session", "windows", opts)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	waitForState(t, cr, id, StageDone)
	if len(runner.commands) != 1 {
		t.Fatalf("commands = %v", runner.commands)
	}
	if strings.Contains(runner.commands[0], "--Loop") {
		t.Fatalf("--Loop must be stripped: %q", runner.commands[0])
	}
	if !strings.Contains(runner.commands[0], "--Stealth") {
		t.Fatalf("--Stealth should pass through: %q", runner.commands[0])
	}
}

func TestCollectionPipelineRunFailure(t *testing.T) {
	_, srv := newFakeIngestServer(t, `{"id":1,"status":0,"created_at":"2026-08-22T12:00:00Z"}`)
	b := &recordingBus{}
	_, cr, _, runner, _, _ := newTestRunner(t, srv.URL, b)
	runner.err = errors.New("exit status 1")

	id, err := cr.Start(context.Background(), "sess-1", "session", "windows", defaultOpts())
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	waitForState(t, cr, id, StageFailed)
	st, _ := cr.Status(id)
	if st.Stage != StageFailed || !strings.Contains(st.Err, "exit status 1") {
		t.Fatalf("state = %+v", st)
	}
}

func TestCollectionPipelineDownloadFailure(t *testing.T) {
	_, srv := newFakeIngestServer(t, `{"id":1,"status":0,"created_at":"2026-08-22T12:00:00Z"}`)
	b := &recordingBus{}
	_, cr, _, _, fetcher, _ := newTestRunner(t, srv.URL, b)
	fetcher.downloadErr = errors.New("chunk failed")

	id, err := cr.Start(context.Background(), "sess-1", "session", "windows", defaultOpts())
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	waitForState(t, cr, id, StageFailed)
	st, _ := cr.Status(id)
	if st.Stage != StageFailed || !strings.Contains(st.Err, "chunk failed") {
		t.Fatalf("state = %+v", st)
	}
}

func TestCollectionStartRejectsBeaconsAndNonWindows(t *testing.T) {
	_, srv := newFakeIngestServer(t, `{"id":1,"status":0,"created_at":"2026-08-22T12:00:00Z"}`)
	b := &recordingBus{}
	_, cr, _, _, _, _ := newTestRunner(t, srv.URL, b)

	if _, err := cr.Start(context.Background(), "b1", "beacon", "windows", defaultOpts()); err == nil {
		t.Fatal("beacons must be rejected in v2")
	}
	if _, err := cr.Start(context.Background(), "s1", "session", "linux", defaultOpts()); err == nil {
		t.Fatal("non-windows agents must be rejected in v2")
	}
	if _, err := cr.Start(context.Background(), "s1", "session", "windows", CollectionOptions{Collector: "metasploit"}); err == nil {
		t.Fatal("unknown collector must be rejected")
	}
}

func TestCollectionStartRequiresConnection(t *testing.T) {
	svc := New(t.TempDir(), nil)
	cr := NewCollectionRunner(svc, &fakeSource{}, &fakeRunner{}, &fakeFetcher{}, &fakeLoot{})
	if _, err := cr.Start(context.Background(), "s1", "session", "windows", defaultOpts()); !errors.Is(err, ErrNotConnected) {
		t.Fatalf("Start = %v, want ErrNotConnected", err)
	}
}
