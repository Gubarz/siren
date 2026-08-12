// Package wailsadapter adapts Wails v3 application services (events, file
// dialogs) to the narrow interfaces consumed by internal packages.
package wailsadapter

import (
	"github.com/wailsapp/wails/v3/pkg/application"
)

// Bridge exposes the Wails application services that internal packages need:
// event emission and file dialogs.
type Bridge struct {
	app *application.App
}

func New(app *application.App) *Bridge {
	return &Bridge{app: app}
}

// Emit satisfies the Emitter interfaces defined by internal consumers
// (console, automation, and the sliver services).
func (b *Bridge) Emit(name string, payload any) {
	b.app.Event.Emit(name, payload)
}

// SaveFileDialog prompts for a save location and returns the chosen path.
// It returns an empty string when the user cancels.
func (b *Bridge) SaveFileDialog(opts *application.SaveFileDialogOptions) (string, error) {
	return b.app.Dialog.SaveFileWithOptions(opts).PromptForSingleSelection()
}

// OpenFileDialog prompts for a file to open and returns the chosen path.
// It returns an empty string when the user cancels.
func (b *Bridge) OpenFileDialog(opts *application.OpenFileDialogOptions) (string, error) {
	return b.app.Dialog.OpenFileWithOptions(opts).PromptForSingleSelection()
}
