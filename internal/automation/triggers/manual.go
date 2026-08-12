package triggers

import (
	"context"

	"siren/internal/automation"
)

type manual struct{}

func Manual() automation.Trigger { return manual{} }

func (manual) Type() string                         { return "manual" }
func (manual) ConfigSchema() []automation.FieldSpec { return nil }

func (manual) Arm(ctx context.Context, _ map[string]any, _ func(automation.FireEvent)) error {
	<-ctx.Done()
	return ctx.Err()
}
