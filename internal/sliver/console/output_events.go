package console

import (
	"context"

	"siren/internal/bus"
	"siren/internal/execctx"
)

const outputTailMaxBytes = 64 * 1024

func (s *Service) SetBus(b bus.Bus) {
	s.bus = b
}

func (s *Service) publishConsoleOutput(ctx context.Context, line, output string) {
	if s.bus == nil || output == "" {
		return
	}
	tail := output
	if len(tail) > outputTailMaxBytes {
		tail = tail[len(tail)-outputTailMaxBytes:]
	}
	payload := map[string]any{"tail": tail}
	if targetID, targetKind, _ := execctx.Target(ctx); targetID != "" || targetKind != "" {
		payload["targetID"] = targetID
		payload["targetKind"] = targetKind
	}
	s.bus.Publish(bus.Event{
		Type:         "gui.console-output",
		Source:       "gui",
		ConnectionID: s.rpc.ConnectionID(),
		Payload:      payload,
	})
}
