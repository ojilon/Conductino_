# 07-00 Mirror concept (what + why)

Goal / Non-goal / Where in code / Concept / Flow / Rewrite options / Exercise / Done when.

Goal: mirrors = re-fetchable server cache, never source of truth. Where: `ReaderMode.tsx tree/treeError/locatePath useState`, `.dev-data/` mirror, `release-prep/04-temp-debug-mirror.md`.
Exercise: delete mirror, confirm re-fetch restores.
Done when: you can state what is mirror vs truth for every panel.
