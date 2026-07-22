package console

import (
	"bytes"
	"fmt"
	"io"
	"reflect"
	"sync"
	"unsafe"

	sliverconsole "github.com/bishopfox/sliver/client/console"
	"github.com/spf13/cobra"
)

type outputSink struct {
	mu sync.Mutex
	w  io.Writer
}

func installOutputSink(con *sliverconsole.SliverClient, sink *outputSink) {
	field := reflect.ValueOf(con).Elem().FieldByName("printf")
	// Sliver keeps its print sink private; hooking it keeps GUI commands off global stdout.
	reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem().Set(
		reflect.ValueOf(sink.printf),
	)
}

func (s *outputSink) printf(format string, args ...any) (int, error) {
	msg := fmt.Sprintf(format, args...)
	return s.Write([]byte(msg))
}

func (s *outputSink) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.w == nil {
		return len(p), nil
	}
	return s.w.Write(p)
}

func (s *outputSink) capture(fn func() error) (string, error) {
	var buf bytes.Buffer

	s.mu.Lock()
	prev := s.w
	s.w = &buf
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		s.w = prev
		s.mu.Unlock()
	}()

	runErr := fn()
	return buf.String(), runErr
}

func routeCommandOutput(cmd *cobra.Command, writer io.Writer) {
	if cmd == nil {
		return
	}
	cmd.SetOut(writer)
	cmd.SetErr(writer)
	for _, child := range cmd.Commands() {
		routeCommandOutput(child, writer)
	}
}
