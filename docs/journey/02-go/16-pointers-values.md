# 02-16 Pointers vs values (*T vs T, when Backend uses each)

Goal / Non-goal / Where in code / Concept / Flow / Rewrite options / Exercise / Done when.

Goal: read `*Backend` vs `FileTreeNode` value. Where: `backend/main.go:52 ListTree returns []models.FileTreeNode (values)`, `:69 ListLibraryTree returns *models.FileTreeNode (pointer, nil = no folder)`, `filesystem.go:156 walkDir Children []FileTreeNode values appended dereferenced`.
Concept: pointer = nullable + shared; value = copy. Tree children are values (no aliasing); root pointer allows nil empty-state.
Exercise: change one return from value to pointer, fix callers, note `go vet` output.
Done when: you choose correctly without guessing.
