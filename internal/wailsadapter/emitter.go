package wailsadapter

import (
	"context"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type Emitter struct {
	ctx context.Context
}

func New(ctx context.Context) *Emitter {
	return &Emitter{ctx: ctx}
}

func (e *Emitter) Emit(name string, payload any) {
	runtime.EventsEmit(e.ctx, name, payload)
}
