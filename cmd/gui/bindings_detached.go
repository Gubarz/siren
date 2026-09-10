package gui

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/wailsapp/wails/v3/pkg/application"
	wailsevents "github.com/wailsapp/wails/v3/pkg/events"
)

const detachedAgentTabEvent = "agent-tab-window-closed"

const (
	detachedWindowWidth  = 900
	detachedWindowHeight = 650
)

type detachedAgentTabEnvelope struct {
	Tab struct {
		ID    string `json:"id"`
		Type  string `json:"type"`
		Label string `json:"label"`
	} `json:"tab"`
	Shell json.RawMessage `json:"shell,omitempty"`
}

type detachedAgentTab struct {
	payload string
	tabType string
	// window and reattach are guarded by App.detachedMu. reattach records that
	// the tab was reattached while its window was still being created, so the
	// detach path closes it instead of showing it.
	window   *application.WebviewWindow
	reattach bool
}

// DetachAgentTab creates a standalone Wails window that loads the frontend in
// agent-tab mode. The payload remains in Go so large task results and
// screenshots do not need to be placed in the window URL.
func (a *App) DetachAgentTab(payload string, x, y int) (string, error) {
	var envelope detachedAgentTabEnvelope
	if err := json.Unmarshal([]byte(payload), &envelope); err != nil {
		return "", fmt.Errorf("invalid agent tab: %w", err)
	}
	if strings.TrimSpace(envelope.Tab.ID) == "" || strings.TrimSpace(envelope.Tab.Type) == "" {
		return "", fmt.Errorf("invalid agent tab: id and type are required")
	}

	token := uuid.NewString()
	record := &detachedAgentTab{payload: payload, tabType: envelope.Tab.Type}
	a.detachedMu.Lock()
	a.detachedTabs[token] = record
	a.detachedMu.Unlock()

	options := a.detachedWindowOptions(token, envelope, x, y)
	window := a.wails.Window.NewWithOptions(options)
	a.detachedMu.Lock()
	record.window = window
	closeEarly := record.reattach
	a.detachedMu.Unlock()
	window.OnWindowEvent(wailsevents.Common.WindowClosing, func(_ *application.WindowEvent) {
		a.detachedAgentTabClosed(token)
	})
	a.registerFileDropWindow(window)
	if closeEarly {
		// A reattach landed while the window was being created; without this it
		// would be left open with its record already gone.
		window.Close()
		return token, nil
	}
	window.Show()
	return token, nil
}

// detachedWindowOptions builds the standalone window for a detached tab.
func (a *App) detachedWindowOptions(token string, envelope detachedAgentTabEnvelope, x, y int) application.WebviewWindowOptions {
	options := application.WebviewWindowOptions{
		Name:             "agent-tab-" + token,
		Title:            envelope.Tab.Label,
		Width:            detachedWindowWidth,
		Height:           detachedWindowHeight,
		MinWidth:         480,
		MinHeight:        320,
		URL:              "/?detachedAgentTab=" + url.QueryEscape(token),
		Frameless:        true,
		Hidden:           true,
		EnableFileDrop:   true,
		BackgroundColour: application.RGBA{Red: 26, Green: 26, Blue: 26, Alpha: 255},
	}
	if x != 0 || y != 0 {
		options.InitialPosition = application.WindowXY
		options.X, options.Y = a.detachedWindowPosition(x, y)
	}
	return options
}

func (a *App) detachedWindowPosition(cursorX, cursorY int) (int, int) {
	x := cursorX - detachedWindowWidth/2
	y := cursorY - 20
	screen := a.wails.Screen.ScreenNearestDipPoint(application.Point{X: cursorX, Y: cursorY})
	if screen == nil || screen.WorkArea.IsEmpty() {
		return x, y
	}
	area := screen.WorkArea
	maxX := max(area.X, area.X+area.Width-detachedWindowWidth)
	maxY := max(area.Y, area.Y+area.Height-detachedWindowHeight)
	return max(area.X, min(x, maxX)), max(area.Y, min(y, maxY))
}

// GetDetachedAgentTab returns the tab envelope associated with a standalone
// window. It is intentionally one-shot only at window close, not at read, so a
// frontend reload can recover the same tab.
func (a *App) GetDetachedAgentTab(token string) (string, error) {
	a.detachedMu.Lock()
	defer a.detachedMu.Unlock()
	record := a.detachedTabs[token]
	if record == nil {
		return "", fmt.Errorf("detached agent tab was not found")
	}
	return record.payload, nil
}

// takeDetachedTab removes token from the registry and reports the window that
// owns it. window is nil when the window had not been created yet, in which
// case the record is marked so the detach path closes it as soon as it exists.
func (a *App) takeDetachedTab(token string) (record *detachedAgentTab, window *application.WebviewWindow) {
	a.detachedMu.Lock()
	defer a.detachedMu.Unlock()
	record = a.detachedTabs[token]
	if record == nil {
		return nil, nil
	}
	delete(a.detachedTabs, token)
	if record.window == nil {
		record.reattach = true
		return record, nil
	}
	return record, record.window
}

// ReattachAgentTab returns a standalone tab to the main workspace and closes
// its auxiliary window. Removing the record first distinguishes this from a
// normal close, which releases the detached tab instead.
func (a *App) ReattachAgentTab(token string) error {
	record, window := a.takeDetachedTab(token)
	if record == nil {
		return fmt.Errorf("detached agent tab was not found")
	}

	a.bridge.Emit("agent-tab-reattach", record.payload)
	if window == nil {
		return nil
	}
	// Close the pointer captured under the lock: reading record.window here
	// raced with the detach path assigning it.
	go func() {
		time.Sleep(50 * time.Millisecond)
		window.Close()
	}()
	return nil
}

func (a *App) detachedAgentTabClosed(token string) {
	a.detachedMu.Lock()
	record := a.detachedTabs[token]
	if record != nil {
		delete(a.detachedTabs, token)
	}
	a.detachedMu.Unlock()
	if record != nil {
		a.bridge.Emit(detachedAgentTabEvent, map[string]string{"type": record.tabType})
	}
}
