package main

import (
	"embed"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
)

//go:embed frontend/dist
var assets embed.FS

//go:embed build/appicon.png
var icon []byte

var app *application.App
var version = "v0.1.0"
var commit = ""

func main() {
	verInfo := resolveVersionInfo(version, commit)
	twsnmp := &Twsnmp{}
	twsnmp.load()
	stop := twsnmp.checkSiteState()

	app = application.New(application.Options{
		Name:        "TWSNMP MW",
		Description: fmt.Sprintf("TWSNMP MW %s\nTWSNMP Multi Window viewer", verInfo.Display),
		Services: []application.Service{
			application.NewService(twsnmp),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
		Icon: icon,
	})

	twsnmp.updater = newUpdateManager(app, verInfo)
	if err := twsnmp.updater.init(); err != nil {
		log.Printf("updater init err=%v", err)
	}

	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title: "TWSNMP MV",
		Mac: application.MacWindow{
			Backdrop: application.MacBackdropTranslucent,
		},
		URL: "/",
	})
	if os.Getenv("TWSNMPMW_DISABLE_AUTO_UPDATE") != "1" {
		twsnmp.updater.startStartupCheck(1500 * time.Millisecond)
	}
	if err := app.Run(); err != nil {
		log.Fatalf("app.Run err=%v", err)
	}
	stop <- true
	close(stop)
	twsnmp.close()
}
