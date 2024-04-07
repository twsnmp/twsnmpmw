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

var app *application.App
var version = "v1.0.0"
var commit = ""

func main() {
	twsnmp := &Twsnmp{}
	twsnmp.load()
	stop := twsnmp.checkSiteState()

	app = application.New(application.Options{
		Name:        "TWSNMP Multi Window",
		Description: "Multi Window Viewer for TWSNMP FC",
		Bind: []any{
			twsnmp,
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
		Title: "TWSNMP MV Main",
		Mac:   application.MacWindow{
			//Backdrop: application.MacBackdropTranslucent,
		},
		URL: "/?page=main",
	})
	if err := app.Run(); err != nil {
		log.Fatalf("app.Run err=%v", err)
	}
	stop <- true
	close(stop)
}
