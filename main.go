package main

import (
	"embed"
	"log"

	"github.com/kbinani/screenshot"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	bounds := screenshot.GetDisplayBounds(0)
	screenWidth := bounds.Dx()
	screenHeight := bounds.Dy()

	windowWidth := int(float64(screenWidth) * 0.5)
	windowHeight := int(float64(screenHeight) * 0.5)

	app := NewApp()

	err := wails.Run(&options.App{
		Title:  "ppatcher",
		Width:  windowWidth,
		Height: windowHeight,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		OnStartup: app.startup,
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		log.Fatal(err)
	}
}
