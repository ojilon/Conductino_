# 05-07 Wails shell (frontend/app.go + main.go)

Goal / Non-goal / Where in code / Concept / Flow / Rewrite options / Exercise / Done when.

Goal: ALL Wails imports isolated. Where: `frontend/app.go` Bind, `frontend/main.go` embed+Run, root `main.go` router.
Rule: `backend/` never imports wailsapp.
Exercise: add a bound method without touching backend/.
Done when: `go vet` + desktop open still pass.
