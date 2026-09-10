package gui

import (
	"context"

	"github.com/bishopfox/sliver/protobuf/clientpb"

	"siren/internal/automation"
	"siren/internal/bloodhound"
	"siren/internal/sliver/automationexec"
	"siren/internal/sliver/files"
	"siren/internal/sliver/loot"
)

// bhExecutor runs collector commands on interactive sessions via the
// automation executor.
type bhExecutor struct {
	exec *automationexec.Executor
}

func (e bhExecutor) Run(ctx context.Context, agentID, command string) (string, error) {
	return e.exec.Execute(ctx, agentID, "session", command)
}

// appCollectorStarter adapts the lazy CollectionRunner to the automation
// CollectorStarter interface.
type appCollectorStarter struct {
	app *App
}

func (s appCollectorStarter) StartCollection(ctx context.Context, agentID, agentKind, agentOS string, req automation.CollectorRequest) (string, error) {
	opts := bloodhound.CollectionOptions{
		Collector:      req.Collector,
		Methods:        req.Methods,
		Flags:          req.Flags,
		Domain:         req.Domain,
		TimeoutSeconds: req.TimeoutSeconds,
		Ingest:         req.Ingest,
		Loot:           req.Loot,
	}
	return s.app.collectionRunner().Start(ctx, agentID, agentKind, agentOS, opts)
}

func (s appCollectorStarter) CollectionState(ctx context.Context, id string) (automation.CollectorProgress, bool) {
	runner := s.app.existingCollectionRunner()
	if runner == nil {
		return automation.CollectorProgress{}, false
	}
	st, ok := runner.Status(id)
	return automation.CollectorProgress{Stage: string(st.Stage), Error: st.Err}, ok
}

// bhFetcher moves collector binaries and artifacts without dialogs.
type bhFetcher struct {
	files *files.Service
}

func (f bhFetcher) Upload(ctx context.Context, agentID, remotePath, localPath string) error {
	return f.files.UploadToPath(agentID, remotePath, localPath)
}

func (f bhFetcher) Download(ctx context.Context, agentID, remotePath, localPath string) error {
	return f.files.DownloadToPath(agentID, remotePath, localPath)
}

// bhLoot archives collection artifacts as binary loot.
type bhLoot struct {
	loot *loot.Service
}

func (l bhLoot) Archive(ctx context.Context, name string, data []byte) error {
	_, err := l.loot.Add(ctx, name, clientpb.FileType_BINARY, data)
	return err
}

func (a *App) newCollectionRunner() *bloodhound.CollectionRunner {
	return bloodhound.NewCollectionRunner(
		a.BloodHound,
		a.BloodHound, // CollectorSource
		bhExecutor{exec: automationexec.NewExecutor(a.Console, a.Beacons)},
		bhFetcher{files: a.Files},
		bhLoot{loot: a.Loot},
	)
}

// collectionRunner returns the collector, building it at most once. Two
// goroutines reach this: a Wails binding thread and the automation engine, so
// the nil check has to be guarded.
func (a *App) collectionRunner() *bloodhound.CollectionRunner {
	return a.collectionRunnerWith(a.newCollectionRunner)
}

// collectionRunnerWith is collectionRunner with the builder injected, so the
// guard can be exercised without standing up the whole service graph.
func (a *App) collectionRunnerWith(build func() *bloodhound.CollectionRunner) *bloodhound.CollectionRunner {
	a.bloodhoundMu.Lock()
	defer a.bloodhoundMu.Unlock()
	if a.BloodHoundCollection == nil {
		a.BloodHoundCollection = build()
	}
	return a.BloodHoundCollection
}

// existingCollectionRunner returns the collector if it has been built, without
// building one just to answer a status query.
func (a *App) existingCollectionRunner() *bloodhound.CollectionRunner {
	a.bloodhoundMu.Lock()
	defer a.bloodhoundMu.Unlock()
	return a.BloodHoundCollection
}

func (a *App) BloodHoundStartCollection(agentID, agentKind, agentOS string, opts bloodhound.CollectionOptions) (string, error) {
	return a.collectionRunner().Start(context.Background(), agentID, agentKind, agentOS, opts)
}

func (a *App) BloodHoundCollectionStatus(id string) (bloodhound.CollectionState, error) {
	runner := a.existingCollectionRunner()
	if runner == nil {
		return bloodhound.CollectionState{}, bloodhound.ErrNotConnected
	}
	if st, ok := runner.Status(id); ok {
		return st, nil
	}
	return bloodhound.CollectionState{}, nil
}

func (a *App) BloodHoundCollections() []bloodhound.CollectionState {
	runner := a.existingCollectionRunner()
	if runner == nil {
		return []bloodhound.CollectionState{}
	}
	return runner.List()
}
