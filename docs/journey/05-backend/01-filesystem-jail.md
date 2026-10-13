# 05-01 Filesystem jail deep dive

Goal / Non-goal / Where in code / Concept / Flow / Rewrite options / Exercise / Done when.

Goal: own Resolve/walk/reveal. Where: `backend/read/resolve.go` Resolve, `backend/list/list.go` walk, `backend/list/library.go` Library.Resolve, `backend/list/reveal.go` ShowContainingFolder.
Exercise: add tests for `..`, absolute outsider, `\`-rooted token, symlink dir.
Done when: jail has tests + you can explain it blind.
