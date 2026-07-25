package console

import (
	"context"

	"sliver-gui/internal/bus"
	"sliver-gui/internal/journal"
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
	if overlay, ok := journal.OverlayFrom(ctx); ok {
		payload["targetID"] = overlay.TargetID
		payload["targetKind"] = overlay.TargetKind
	}
	s.bus.Publish(bus.Event{
		Type:         "gui.console-output",
		Source:       "gui",
		ConnectionID: s.rpc.ConnectionID(),
		Payload:      payload,
	})
}
