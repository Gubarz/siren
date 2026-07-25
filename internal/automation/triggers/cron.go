package triggers

import (
	"context"
	"fmt"
	"time"

	"github.com/robfig/cron/v3"

	"sliver-gui/internal/automation"
)

var cronParser = cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)

type cronTrigger struct{}

func Cron() automation.Trigger { return cronTrigger{} }

func (cronTrigger) Type() string { return "cron" }

func (cronTrigger) ConfigSchema() []automation.FieldSpec {
	return []automation.FieldSpec{{
		Key: "schedule", Label: "Cron schedule", Type: "string", Required: true,
		Default: "0 */6 * * *",
	}}
}

func (cronTrigger) Validate(cfg map[string]any) error {
	schedule, _ := cfg["schedule"].(string)
	if _, err := cronParser.Parse(schedule); err != nil {
		return fmt.Errorf("cron schedule: %w", err)
	}
	return nil
}

func (cronTrigger) Arm(ctx context.Context, cfg map[string]any, fire func(automation.FireEvent)) error {
	schedule, _ := cfg["schedule"].(string)
	sched, err := cronParser.Parse(schedule)
	if err != nil {
		return fmt.Errorf("cron schedule: %w", err)
	}
	timer := time.NewTimer(time.Until(sched.Next(time.Now())))
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case now := <-timer.C:
			fire(automation.FireEvent{Data: map[string]any{"schedule": schedule}})
			timer.Reset(time.Until(sched.Next(now)))
		}
	}
}
