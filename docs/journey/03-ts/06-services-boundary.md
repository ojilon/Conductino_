# 03-06 Services boundary (backend.ts mock vs wails)

Goal / Non-goal / Where in code / Concept / Flow / Rewrite options / Exercise / Done when.

Goal: never touch OS from React. Where: `services/backend.ts:102 BackendServices`, `createBackend()`, Wails* impls.
Exercise: add a new backend method end-to-end (Go + WailsApp + impl).
Done when: browser still mocks, desktop still live.
