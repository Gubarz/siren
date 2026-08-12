package gui

import (
	"embed"
	"fmt"
	"os"

	"github.com/wailsapp/wails/v3/pkg/application"

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

	wailsApp := application.New(application.Options{
		Name: "siren",
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(frontendAssets),
		},
	})

	window := wailsApp.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            "siren",
		Width:            1024,
		Height:           768,
		StartState:       application.WindowStateMaximised,
		Frameless:        true,
		Hidden:           true,
		EnableFileDrop:   true,
		BackgroundColour: application.RGBA{Red: 26, Green: 26, Blue: 26, Alpha: 255},
	})

	wailsApp.RegisterService(application.NewService(NewApp(wailsApp, window)))

	if err := wailsApp.Run(); err != nil {
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
