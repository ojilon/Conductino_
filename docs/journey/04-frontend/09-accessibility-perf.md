# 04-09 Accessibility + perf on low-spec (4GB Celeron)

Goal / Non-goal / Where in code / Concept / Flow / Rewrite options / Exercise / Done when.

Goal: fast on weak hardware. Where: virtualized lists, canvas zoom, bundle audit (`pnpm build` size).
Exercise: profile opening a large doc, note long tasks, move work off main thread or to Go.
Done when: no jank on target machine.
