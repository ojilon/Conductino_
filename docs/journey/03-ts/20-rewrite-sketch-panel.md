# 03-20 Rewrite sketch: panel React -> TS helper

Goal / Non-goal / Where in code / Concept / Flow / Rewrite options / Exercise / Done when.

Goal: copy-paste starter. Before: logic inside JSX. After:
```ts
// features/reader/libraryTree.ts
export interface TreeViewModel { id:string; label:string; depth:number; kind:'folder'|'file' }
export function toViewModel(root: FileTreeNode|null): TreeViewModel[] // pure, testable
// ReaderSidebar.tsx: thin wrapper maps viewModel -> <ul>
```
Exercise: apply to one panel; measure lines moved out of JSX.
Done when: helper unit-checkable without rendering.
