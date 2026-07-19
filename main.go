package main

import (
	"embed"
	"fmt"
	"os"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"

	"sliver-gui/internal/buildinfo"
	"sliver-gui/internal/console"
)

//go:embed all:frontend/dist
var frontendAssets embed.FS

func main() {
	if len(os.Args) >= 2 && os.Args[1] == "--version" {
		fmt.Println(buildinfo.String())
		return
	}

	// Console-mode re-exec: this process was spawned by a running GUI to
	// host a real sliver client console for a specific session. Skip the
	// wails app entirely and hand control to the sliver console loop.
	if len(os.Args) >= 3 && os.Args[1] == console.ConsoleModeFlag {
		sessionID := ""
		if len(os.Args) >= 4 {
			sessionID = os.Args[3]
		}
		if err := console.RunConsoleSubprocess(os.Args[2], sessionID); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}

	// Create an instance of the app structure
	app := NewApp()

	// Create application with options
	err := wails.Run(&options.App{
		Title:            "sliver-gui",
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
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
