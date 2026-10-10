# 04-11 ReaderMode deep dive (owns library-tree cache)

Goal / Non-goal / Where in code / Concept / Flow / Rewrite options / Exercise / Done when.

Goal: own the exception to single-source rule. Where: `ReaderMode.tsx:32 tree/treeError/locatePath useState` (server-mirror cache, re-fetchable), `:61 pickFolder`, `:139 open with metadata.path+rootPath`.
Exercise: add refresh + stale-tag UI per 04-03.
Done when: tree cache documented as mirror, not truth.
