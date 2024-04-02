package main

import (
	"embed"
	"log"

	"github.com/wailsapp/wails/v3/pkg/application"
)

//go:embed frontend/dist
var assets embed.FS

//go:embed build/appicon.png
var icon []byte

var version = "v1.0.0"
var commit = ""

func main() {

	app := application.New(application.Options{
		Name:        "TWSNMP Multi Window",
		Description: "Multi Window Viewer for TWSNMP FC",
		Bind: []any{
			&TwsnmpMWService{},
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
		Icon: icon,
	})

	app.NewWebviewWindowWithOptions(application.WebviewWindowOptions{
		Title: "TWSNMP Multi Window",
		Mac: application.MacWindow{
			Backdrop: application.MacBackdropTranslucent,
		},
		URL: "/",
	})

	err := app.Run()

	if err != nil {
		log.Fatal(err)
	}
}
