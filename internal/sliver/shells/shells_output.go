package shells

import (
	"bytes"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

const (
	maxShellOutputBytes = 4 * 1024 * 1024
	shellTrimTarget     = 3 * 1024 * 1024
	shellEventBatchSize = 512 * 1024
	shellEventInterval  = 100 * time.Millisecond
)

// emitShellOutput batches shell chunks onto the wails event bus, flushing
// on size or interval so fast output doesn't flood the frontend.
func (s *Service) emitShellOutput(id string, chunks <-chan []byte, done chan<- struct{}) {
	defer close(done)
	ticker := time.NewTicker(shellEventInterval)
	defer ticker.Stop()

	pending := make([]byte, 0, shellEventBatchSize)
	flush := func() {
		if len(pending) == 0 {
			return
		}
		runtime.EventsEmit(s.ctx, "shell-output", map[string]interface{}{
			"id": id, "data": string(pending),
		})
		pending = pending[:0]
	}

	for {
		select {
		case chunk, ok := <-chunks:
			if !ok {
				flush()
				return
			}
			pending = append(pending, chunk...)
			if len(pending) >= shellEventBatchSize {
				flush()
			}
		case <-ticker.C:
			flush()
		}
	}
}

// appendBoundedShellOutput appends chunk and trims the head (on a newline
// boundary when possible) once the buffer exceeds maxShellOutputBytes.
func appendBoundedShellOutput(output, chunk []byte) []byte {
	output = append(output, chunk...)
	if len(output) <= maxShellOutputBytes {
		return output
	}

	start := len(output) - shellTrimTarget
	if newline := bytes.IndexByte(output[start:], '\n'); newline >= 0 {
		start += newline + 1
	}
	trimmed := make([]byte, len(output)-start)
	copy(trimmed, output[start:])
	return trimmed
}
