# 05-17 Sketch: new capability end-to-end

Goal / Non-goal / Where in code / Concept / Flow / Rewrite options / Exercise / Done when.

Goal: copy-paste checklist in code. Steps with files:
1 reader arm or store method in its small package 2 `backend/main.go` orchestrate (no new interfaces) 3 `frontend/app.go` bind 4 `services/backend.ts` WailsApp+helper 5 Settings row if user-facing 6 test both modes.
Exercise: add `ReadDumpFile`-style readback through all steps (already done once — re-read that diff).
Done when: new call works desktop, degrades honestly in browser.
