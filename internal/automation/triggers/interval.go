package triggers

import (
	"context"
	"time"

	"sliver-gui/internal/automation"
)

var MinInterval = 10 * time.Second

type interval struct{}

func Interval() automation.Trigger { return interval{} }

func (interval) Type() string { return "interval" }

func (interval) ConfigSchema() []automation.FieldSpec {
	return []automation.FieldSpec{{
		Key: "intervalSeconds", Label: "Interval (seconds)", Type: "number",
		Required: true, Default: 300,
	}}
}

func (interval) Arm(ctx context.Context, cfg map[string]any, fire func(automation.FireEvent)) error {
	seconds := cfgNumber(cfg, "intervalSeconds", 300)
	d := time.Duration(seconds) * time.Second
	if d < MinInterval {
		d = MinInterval
	}
	ticker := time.NewTicker(d)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			fire(automation.FireEvent{Data: map[string]any{"intervalSeconds": seconds}})
		}
	}
}

func cfgNumber(cfg map[string]any, key string, fallback float64) float64 {
	switch v := cfg[key].(type) {
	case float64:
		return v
	case int:
		return float64(v)
	case int64:
		return float64(v)
	}
	return fallback
}
