package main

import (
	"embed"

	"siren/cmd/gui"
)

//go:embed all:frontend/dist
var frontendAssets embed.FS

func main() {
	gui.Run(frontendAssets)
}
