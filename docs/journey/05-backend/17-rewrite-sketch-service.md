# 05-17 Rewrite sketch: new service end-to-end

Goal / Non-goal / Where in code / Concept / Flow / Rewrite options / Exercise / Done when.

Goal: copy-paste checklist in code. Steps with files:
1 `services/mine.go` interface+impl 2 `backend/main.go` aggregate+delegate 3 `frontend/app.go` bind 4 `services/backend.ts` WailsApp+impl 5 Settings row 6 test both modes.
Exercise: add `Ping() string` through all 6 steps.
Done when: Ping works desktop, mocks browser.
