package triggers

import (
	"context"

	"siren/internal/automation"
	"siren/internal/bus"
)

type MapFunc func(ev bus.Event, cfg map[string]any) (automation.FireEvent, bool)

type busEventTrigger struct {
	triggerType string
	busType     string
	schema      []automation.FieldSpec
	b           bus.Bus
	mapFn       MapFunc
}

func BusEvent(triggerType, busType string, schema []automation.FieldSpec, b bus.Bus, mapFn MapFunc) automation.Trigger {
	return &busEventTrigger{triggerType: triggerType, busType: busType, schema: schema, b: b, mapFn: mapFn}
}

func (t *busEventTrigger) Type() string { return t.triggerType }

func (t *busEventTrigger) ConfigSchema() []automation.FieldSpec { return t.schema }

func (t *busEventTrigger) Arm(ctx context.Context, cfg map[string]any, fire func(automation.FireEvent)) error {
	fired := make(chan automation.FireEvent, 16)
	unsub := t.b.Subscribe([]string{t.busType}, func(ev bus.Event) {
		fe, ok := t.mapFn(ev, cfg)
		if !ok {
			return
		}
		select {
		case fired <- fe:
		default:
		}
	})
	defer unsub()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case fe := <-fired:
			fire(fe)
		}
	}
}
