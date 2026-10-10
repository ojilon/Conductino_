# 03-15 Why reducer is split (reducePart1a..4, Workspace, Chat)

Goal / Non-goal / Where in code / Concept / Flow / Rewrite options / Exercise / Done when.

Goal: navigate splits. Where: `appState.tsx:66` chain `reduceWorkspace ?? reduceChat ?? reducePart1a ... ?? state`.
Concept: each returns new state or null (not-mine); order matters; pure + synchronous.
Exercise: move one case to the correct file, verify.
Done when: you know where each action lives.
