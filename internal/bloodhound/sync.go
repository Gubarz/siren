package bloodhound

import (
	"context"
	"time"

	"siren/internal/bus"
)

// Event types published by this package on the app bus.
const (
	EventStatus     = "bloodhound.status"
	EventEnrichment = "bloodhound.enrichment"
	EventSynced     = "bloodhound.synced"
)

// SyncedDTO is the payload of bloodhound.synced: fresh domain list and
// recomputed enrichment for the agents last seen by Correlate.
type SyncedDTO struct {
	Domains     []DomainDTO           `json:"domains"`
	Enrichments map[string]Enrichment `json:"enrichments"`
	At          int64                 `json:"at"`
}

func (s *Service) publish(eventType string, payload any) {
	if s.bus == nil {
		return
	}
	s.bus.Publish(bus.Event{Type: eventType, Source: "bloodhound", Payload: payload})
}

func (s *Service) publishStatus() {
	s.publish(EventStatus, s.Status())
}

// StartSync runs the background refresh loop: every interval, while
// connected, it reloads the domain list and recomputes enrichment for the
// retained agent refs, then publishes bloodhound.synced. Cancelled via ctx.
func (s *Service) StartSync(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = 5 * time.Minute
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.syncOnce(ctx)
		}
	}
}

func (s *Service) syncOnce(ctx context.Context) {
	st := s.Status()
	if !st.Connected {
		return
	}
	domains, err := s.ListDomains(ctx)
	if err != nil {
		s.publish(EventStatus, st)
		return
	}
	refs := s.corr.Refs()
	enrichments := map[string]Enrichment{}
	if len(refs) > 0 {
		s.InvalidateCorrelation()
		if m, err := s.Correlate(ctx, refs); err == nil {
			enrichments = m
		}
	}
	s.publish(EventSynced, SyncedDTO{
		Domains:     domains,
		Enrichments: enrichments,
		At:          time.Now().UnixMilli(),
	})
}
