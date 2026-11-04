// Hypo is an open-source Windows maintenance GUI that drives the Mole CLI
// engine. It is a community project, GPL-3.0 like the engine, and is not
// affiliated with the Mole project (see README.md).
package main

import (
	"embed"
	"log"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/wuqingwei/hypo/internal/app"
)

// Wails embeds the built frontend into the binary.
//
//go:embed all:frontend/dist
var assets embed.FS

func main() {
	svc := app.New()

	wailsApp := application.New(application.Options{
		Name:        "Hypo",
		Description: "Open-source Windows maintenance GUI powered by the Mole CLI engine",
		Services: []application.Service{
			application.NewService(svc),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})

	wailsApp.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            "Hypo",
		Width:            1180,
		Height:           780,
		MinWidth:         940,
		MinHeight:        620,
		BackgroundColour: application.NewRGB(13, 17, 23),
		URL:              "/",
	})

	if err := wailsApp.Run(); err != nil {
		log.Fatal(err)
	}
}
