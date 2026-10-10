# 02-07 File IO + Resolve jail

Goal / Non-goal / Where in code / Concept / Flow / Rewrite options / Exercise / Done when.

Goal: own containment. Non-goal: new FS features.
Where: `filesystem.go:90 Resolve` (Join+Cleans+Rel check rejects `..` escape), `walkDir` skips unreadable dirs, symlinks as files (no cycles).
Exercise: attempt `..` escape, confirm rejection; add a test.
Done when: you can explain the jail in 3 sentences.
