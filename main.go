package main

// Repo entrypoint — thin router only, no Wails and no business logic.
//
// Where everything moved (and why):
//   - All Wails concerns live in frontend/ (package frontend, importable):
//     frontend/main.go owns asset embedding + wails.Run (Run), and
//     frontend/app.go owns the bound shell (App: Startup/Shutdown plus
//     thin forwarders that wrap the bridge and call runtime.EventsEmit).
//     They are `package frontend` rather than `package main` because Go
//     cannot import a `package main` — the root router below must be able
//     to import the shell.
//   - All pure Go lives in backend/ (package backend, zero Wails imports):
//     backend/main.go is ONLY the bridge — type Backend aggregates one
//     instance per submodule (models/ wire types + services/ capabilities)
//     and forwards each call to exactly one service with ctx passed
//     explicitly, so it stays testable with plain `go test`.
//   - This file just builds the shell and runs it. To add a capability,
//     add a method on backend.Backend, expose it via frontend.App, and it
//     is automatically bound through Run. Binding name in the UI is
//     window.go.frontend.App.* (see frontend/src/services/backend.ts).
//
// `wails build`/`wails dev` still run from the repo root (wails.json:
// frontend:dir `frontend`, wailsjs:dir `./frontend/wailsjs`); nothing about
// the CLI workflow changes.
import (
	// `Conductino` matches `module Conductino` in go.mod.
	"Conductino/frontend"
)

func main() {
	// Build the Wails shell (which in turn builds the pure-Go backend
	// bridge behind it) and start the desktop app. All options, hooks
	// and bindings are configured inside frontend.Run.
	app := frontend.NewApp()
	if err := frontend.Run(app); err != nil {
		println("Error:", err.Error())
	}
}
