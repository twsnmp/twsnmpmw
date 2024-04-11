package main

import (
	"embed"
	"fmt"
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
		Name:        "TWSNMP MW",
		Description: fmt.Sprintf("TWSNMP MW %s(%s)\nTWSNMP Multi Window viewer", version, commit),
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
		Title: "TWSNMP MV",
		Mac: application.MacWindow{
			Backdrop: application.MacBackdropTranslucent,
		},
		URL: "/",
	})
	if err := app.Run(); err != nil {
		log.Fatalf("app.Run err=%v", err)
	}
	stop <- true
	close(stop)
}
