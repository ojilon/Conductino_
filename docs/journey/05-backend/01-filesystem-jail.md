# 05-01 Filesystem jail deep dive

Goal / Non-goal / Where in code / Concept / Flow / Rewrite options / Exercise / Done when.

Goal: own Resolve/walk/reveal. Where: `filesystem.go:66 ListRoot`, `:90 Resolve`, `:111 ShowContainingFolder`, `:140 walkDir`.
Exercise: add tests for `..`, absolute outsider, symlink dir.
Done when: jail has tests + you can explain it blind.
