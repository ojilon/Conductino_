# 03-11 null vs undefined (bridges + legacy)

Goal / Non-goal / Where in code / Concept / Flow / Rewrite options / Exercise / Done when.

Goal: nil->null, missing->undefined. Where: `ListLibraryTree()->FileTreeNode|null`, `workspaceId?` legacy unscoped.
Exercise: audit one boundary for null handling.
Done when: no `!` non-null assertions added.
