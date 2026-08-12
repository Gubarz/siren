package gui

import (
	"embed"
	"fmt"
	"os"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"

	"siren/internal/buildinfo"
	"siren/internal/sliver/console"
)

func Run(frontendAssets embed.FS) {
	if len(os.Args) >= 2 && os.Args[1] == "--version" {
		fmt.Println(buildinfo.String())
		return
	}

	if runConsoleMode() {
		return
	}

	if err := wails.Run(appOptions(frontendAssets, NewApp())); err != nil {
		println("Error:", err.Error())
	}
}

func runConsoleMode() bool {
	if len(os.Args) < 3 || os.Args[1] != console.ConsoleModeFlag {
		return false
	}
	sessionID := ""
	if len(os.Args) >= 4 {
		sessionID = os.Args[3]
	}
	if err := console.RunConsoleSubprocess(os.Args[2], sessionID); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	return true
}

func appOptions(frontendAssets embed.FS, app *App) *options.App {
	return &options.App{
		Title:            "siren",
		Width:            1024,
		Height:           768,
		WindowStartState: options.Maximised,
		Frameless:        true,
		StartHidden:      true,
		AssetServer: &assetserver.Options{
			Assets: frontendAssets,
		},
		BackgroundColour: &options.RGBA{R: 26, G: 26, B: 26, A: 255},
		DragAndDrop:      &options.DragAndDrop{EnableFileDrop: true},
		OnStartup:        app.startup,
		OnShutdown:       app.shutdown,
		Bind: []interface{}{
			app,
		},
	}
}
