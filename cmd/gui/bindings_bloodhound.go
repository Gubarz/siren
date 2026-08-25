package gui

import (
	"context"
	"time"

	"siren/internal/bloodhound"
)

func (a *App) BloodHoundGetConfig() bloodhound.ConfigView {
	return a.BloodHound.GetConfig()
}

// BloodHoundSaveConfig persists cfg; a blank token key keeps the stored one.
func (a *App) BloodHoundSaveConfig(cfg bloodhound.Config) error {
	return a.BloodHound.MergeSaveConfig(cfg)
}

// BloodHoundTestConnection verifies credentials without persisting them;
// a blank token key falls back to the locally stored one.
func (a *App) BloodHoundTestConnection(cfg bloodhound.Config) error {
	return a.BloodHound.TestConnection(cfg)
}

func (a *App) BloodHoundConnect() error {
	return a.BloodHound.Connect(context.Background())
}

func (a *App) BloodHoundMarkOwned(objectID string) error {
	return a.BloodHound.MarkOwned(context.Background(), objectID)
}

func (a *App) BloodHoundUnmarkOwned(objectID string) error {
	return a.BloodHound.UnmarkOwned(context.Background(), objectID)
}

func (a *App) BloodHoundDisconnect() {
	a.BloodHound.Disconnect()
}

func (a *App) BloodHoundStatus() bloodhound.Status {
	return a.BloodHound.Status()
}

func (a *App) BloodHoundSearchEntities(query string, offset, limit int) (bloodhound.SearchPage, error) {
	return a.BloodHound.SearchEntities(context.Background(), query, offset, limit)
}

func (a *App) BloodHoundListDomains() ([]bloodhound.DomainDTO, error) {
	return a.BloodHound.ListDomains(context.Background())
}

func (a *App) BloodHoundEntity(objectID string) (*bloodhound.Entity, error) {
	return a.BloodHound.Entity(context.Background(), objectID)
}

func (a *App) BloodHoundCorrelate(agents []bloodhound.AgentRef) (map[string]bloodhound.Enrichment, error) {
	return a.BloodHound.Correlate(context.Background(), agents)
}

func (a *App) BloodHoundKerberoastTargets() ([]bloodhound.Entity, error) {
	return a.BloodHound.KerberoastTargets(context.Background())
}

func (a *App) BloodHoundIngestJobs() ([]bloodhound.IngestJobDTO, error) {
	return a.BloodHound.IngestJobs(context.Background())
}

func (a *App) BloodHoundIngestJob(id int64) (bloodhound.IngestJobDTO, error) {
	return a.BloodHound.IngestJob(context.Background(), id)
}

func (a *App) BloodHoundIngestLocalFile(path string) (bloodhound.IngestJobDTO, error) {
	return a.BloodHound.IngestLocalFile(context.Background(), path)
}

func (a *App) BloodHoundWatchIngestJob(id int64) error {
	return a.BloodHound.WatchIngestJob(context.Background(), id, 2*time.Second)
}

func (a *App) BloodHoundAttackPaths(objectID string, maxPaths int) (bloodhound.GraphDTO, error) {
	return a.BloodHound.EntityAttackPaths(context.Background(), objectID, maxPaths)
}

func (a *App) BloodHoundCommunityQuery(kind string) (bloodhound.GraphDTO, error) {
	return a.BloodHound.CommunityQuery(context.Background(), bloodhound.CommunityKind(kind))
}
