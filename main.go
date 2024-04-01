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
			InvisibleTitleBarHeight: 50,
			Backdrop:                application.MacBackdropTranslucent,
			TitleBar:                application.MacTitleBarHiddenInset,
		},
		BackgroundColour: application.NewRGB(27, 38, 54),
		URL:              "/",
	})

	err := app.Run()

	if err != nil {
		log.Fatal(err)
	}
}
