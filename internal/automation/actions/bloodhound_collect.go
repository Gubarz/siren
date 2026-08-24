package actions

import (
	"context"
	"fmt"
	"strings"
	"time"

	"siren/internal/automation"
)

// bloodhoundCollect runs a SharpHound/AzureHound collection on the rule's
// target agent, then tracks the pipeline to completion. Combine with the
// session-connected/beacon-checkin/cron triggers for automated collection.
type bloodhoundCollect struct{}

func BloodHoundCollect() automation.Action { return bloodhoundCollect{} }

func (bloodhoundCollect) Type() string { return "bloodhound_collect" }

func (bloodhoundCollect) ConfigSchema() []automation.FieldSpec {
	return []automation.FieldSpec{
		{Key: "collector", Label: "Collector", Type: "string", Default: "sharphound"},
		{Key: "methods", Label: "Collection methods (comma-separated)", Type: "string", Default: "Default"},
		{Key: "flags", Label: "Extra flags (space-separated)", Type: "string"},
		{Key: "domain", Label: "Domain", Type: "string"},
		{Key: "timeoutMinutes", Label: "Timeout (minutes)", Type: "number", Default: float64(15)},
		{Key: "autoIngest", Label: "Ingest into BloodHound", Type: "bool", Default: true},
		{Key: "archiveLoot", Label: "Archive artifact to loot", Type: "bool", Default: true},
	}
}

func (bloodhoundCollect) Execute(rc *automation.RunContext) error {
	if rc.Deps.Collector == nil {
		return fmt.Errorf("bloodhound: collection is not available")
	}
	cfg := rc.Action.Config
	collector := cfgString("collector", cfg)
	if strings.TrimSpace(collector) == "" {
		collector = "sharphound"
	}
	req := automation.CollectorRequest{
		Collector:      strings.ToLower(strings.TrimSpace(collector)),
		Methods:        splitCommaList(cfgString("methods", cfg), "Default"),
		Flags:          strings.Fields(cfgString("flags", cfg)),
		Domain:         strings.TrimSpace(cfgString("domain", cfg)),
		TimeoutSeconds: int(cfgFloat("timeoutMinutes", cfg, 15)) * 60,
		Ingest:         cfgBoolOr(cfg, "autoIngest", true),
		Loot:           cfgBoolOr(cfg, "archiveLoot", true),
	}

	id, err := rc.Deps.Collector.StartCollection(rc.Ctx, rc.Target.ID, rc.Target.Kind, rc.Target.OS, req)
	if err != nil {
		return err
	}
	rc.Log("bloodhound collection started: ", id)

	timeout := time.Duration(req.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 15 * time.Minute
	}
	ctx, cancel := context.WithTimeout(rc.Ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	lastStage := ""
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			progress, ok := rc.Deps.Collector.CollectionState(ctx, id)
			if !ok {
				return fmt.Errorf("bloodhound: collection %s disappeared", id)
			}
			if progress.Stage != lastStage {
				lastStage = progress.Stage
				rc.Log("stage: ", progress.Stage)
			}
			switch progress.Stage {
			case "done":
				return nil
			case "failed":
				return fmt.Errorf("bloodhound collection failed: %s", progress.Error)
			}
		}
	}
}

func splitCommaList(raw string, fallback string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		out = append(out, fallback)
	}
	return out
}

// cfgBoolOr reads a bool config value, returning fallback when unset.
func cfgBoolOr(cfg map[string]any, key string, fallback bool) bool {
	if v, ok := cfg[key].(bool); ok {
		return v
	}
	return fallback
}
