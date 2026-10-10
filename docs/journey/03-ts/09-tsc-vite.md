# 03-09 tsc + Vite + singlefile build

Goal / Non-goal / Where in code / Concept / Flow / Rewrite options / Exercise / Done when.

Goal: own the build. Where: `tsconfig.json`, `vite.config.ts`, `frontend/dist` embedded via `//go:embed`.
Exercise: run `pnpm exec tsc --noEmit` + `pnpm build`, inspect dist size.
Done when: you can explain embed to a friend.
