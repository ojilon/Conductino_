# 01-05 Library tree flow: dialog -> walk -> opaque tokens

Goal / Non-goal / Where in code / Concept / Flow / Rewrite options / Exercise / Done when.

Goal: understand folder-open pipe. Non-goal: multi-folder yet (see tasks.md 1.1).
Where in code: `ReaderMode.tsx:61 pickFolder`, `filesystem.go:49 SetRoot`, `:66 ListRoot`, `:140 walkDir`, `models.FileTreeNode` Path = root-relative opaque token.
Flow: SelectFolder -> SetRoot(abs) -> ListLibraryTree -> nested root node -> UI displays label, sends path back, never builds paths.
Exercise: choose folder, inspect one node's {id,label,path} in devtools.
Done when: you can explain why UI must never join paths.
