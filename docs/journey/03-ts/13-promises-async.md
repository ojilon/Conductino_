# 03-13 Promises + async/await (backend.ts)

Goal / Non-goal / Where in code / Concept / Flow / Rewrite options / Exercise / Done when.

Goal: own async boundary. Where: `WailsFilesystem.openFile` returns Promise<OpenedFile|null>, `delay()` mocks, `catch->null` (tasks.md 1.2 hazard).
Concept: `async` = Promise; `await` = suspend; rejection = Go error (unexpected), value-with-reason = classified.
Exercise: convert one `.then` chain to async/await + typed reason handling.
Done when: no silent `catch {return null}` remains where reason matters.
