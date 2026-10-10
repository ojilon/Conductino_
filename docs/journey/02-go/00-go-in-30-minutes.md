# 02-00 Go in 30 minutes (via this repo)

Goal / Non-goal / Where in code / Concept / Flow / Rewrite options / Exercise / Done when.

Goal: read Go files without fear. Non-goal: concurrency yet (see 10-split).
Concepts: package, import, struct, method on *T, interface, error return, zero values.
Where in code: `backend/main.go:19 Backend struct`, `:27 NewBackend()`, `:52 ListTree (T,error)`.
Snippet: `func (b *Backend) OpenFile(...) (*models.OpenedDocument, error)` = method on *Backend returning pointer+error.
Exercise: annotate `filesystem.go:37 Filesystem struct` + each method receiver.
Done when: you can parse any signature in backend/.
