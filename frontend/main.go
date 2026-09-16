// Wails bootstrap for package frontend (importable shell, not main).
// 
// All Wails concerns — asset embedding, wails.Run options, lifecycle wiring —
// live in this package (app.go: App plus Run below). The repo-root main.go
// is then a ~5-line thin router that calls NewApp + Run, and backend/
// never imports Wails at all.
//
// NOTE on the embed path: because this file sits INSIDE frontend/, the
// pattern is `all:dist` (relative to this directory). The old root-level
// `all:frontend/dist` pattern would break here — embed paths are always
// relative to the file that declares them. wails.json still points
// `frontend:dir` at `frontend` and `wailsjs:dir` at `./frontend/wailsjs`,
// and `wails build` still runs from the repo root (where the real
// `package main` in root main.go lives), so no CLI changes are needed.
package frontend

import (
	"embed"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

// assets serves the built web UI (Vite output) inside the desktop webview.
// Rebuilt by the frontend pipeline (`npm run build` → frontend/dist).
//
//go:embed all:dist
var assets embed.FS

// Run starts the Wails application with the given shell. The shell is built
// by NewApp (app.go) and holds the *backend.Backend bridge; Run binds it so
// the UI can call window.go.frontend.App.* and wires Startup/Shutdown.
func Run(app *App) error {
	return wails.Run(&options.App{
		Title:     "Conductino",
		Width:     1024,
		Height:    768,
		Frameless: true,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        app.Startup,
		OnShutdown:       app.Shutdown,
		Bind: []interface{}{
			app,
		},
	})
}
