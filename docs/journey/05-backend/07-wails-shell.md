# 05-07 Wails shell (frontend/app.go + main.go)

Goal / Non-goal / Where in code / Concept / Flow / Rewrite options / Exercise / Done when.

Goal: ALL Wails imports isolated. Where: `frontend/app.go` Bind, `frontend/main.go` embed+Run, root `main.go` router. Bound = exactly what `backend.ts` calls (ListLibraryTree, OpenFile, ReadDumpFile, ReadRawFile, SelectFolder, ShowContainingFolder, LibraryRoot, AIMeters, StreamAIRequest, SetPrimarySummary, WriteSummaryDOCX, SaveThread, AppendChatMessage, ListThreads, ListMessages, BuildContextPack, MatchSkills). Dead surface (ListTree, RunAI, doc/change persistence wrappers) deleted in the remake.
Rule: `backend/` never imports wailsapp.
Exercise: add a bound method without touching backend/.
Done when: `go vet` + desktop open still pass.
