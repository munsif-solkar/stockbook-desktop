package main

import (
	"embed"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/linux"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/appicon.png
var appIcon []byte

func main() {
	restrictFilePermissions()
	app := NewApp()

	err := wails.Run(&options.App{
		Title:             "Stockbook",
		Width:             1280,
		Height:            840,
		MinWidth:          1000,
		MinHeight:         660,
		BackgroundColour:  &options.RGBA{R: 245, G: 246, B: 245, A: 1},
		AssetServer:       &assetserver.Options{Assets: assets},
		OnStartup:         app.startup,
		OnShutdown:        app.shutdown,
		Bind:              []interface{}{app},
		Linux:             &linux.Options{Icon: appIcon, ProgramName: "stockbook"},
		HideWindowOnClose: false,
	})
	if err != nil {
		println("Error:", err.Error())
	}
}
