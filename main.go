package main

import (
	"embed"
	"log"

	"github.com/wailsapp/wails/v3/pkg/application"
)

//go:embed frontend/dist
var assets embed.FS

var version = "v1.0.0"
var commit = ""
var app *application.App

func main() {
	app = application.New(application.Options{
		Name:        "twsnmpmw",
		Description: "TWSNMMP FC Multi Windows",
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
		Assets: application.AssetOptions{
			FS: assets,
		},
		Bind: []interface{}{
			&TwsnmpMWService{},
		},
	})
	// Create window
	app.NewWebviewWindowWithOptions(&application.WebviewWindowOptions{
		Title: "TWSNMP FC Multi Window",
		URL:   "/",
	})

	err := app.Run()

	if err != nil {
		log.Fatal(err)
	}
}
