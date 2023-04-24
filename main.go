package main

import (
	"embed"
	"fmt"
	"log"

	"github.com/wailsapp/wails/v3/pkg/application"
)

//go:embed frontend/dist
var assets embed.FS

type App struct{}

var verison = "v1.0.0"
var commit = ""

func (a *App) GetVersion() string {
	return fmt.Sprintf("%s(%s)", verison, commit)
}

func main() {
	app := application.New(application.Options{
		Name:        "twsnmpmw",
		Description: "A demo of using raw HTML & CSS",
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
		Assets: application.AssetOptions{
			FS: assets,
		},
		Bind: []any{
			&App{},
		},
	})
	// Create window
	app.NewWebviewWindowWithOptions(&application.WebviewWindowOptions{
		Title: "Plain Bundle",
		CSS:   `body { background-color: rgba(255, 255, 255, 0); } .main { color: white; margin: 20%; }`,
		Mac: application.MacWindow{
			InvisibleTitleBarHeight: 50,
			Backdrop:                application.MacBackdropTranslucent,
			// TitleBar:                application.MacTitleBarHiddenInset,
		},

		URL: "/",
	})

	err := app.Run()

	if err != nil {
		log.Fatal(err)
	}
}
