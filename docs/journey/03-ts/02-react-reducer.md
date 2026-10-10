# 03-02 React reducer (appState.tsx + reducePart*)

Goal / Non-goal / Where in code / Concept / Flow / Rewrite options / Exercise / Done when.

Goal: own the single-reducer rule. Where: `appState.tsx:66 reducer` chaining 7 reducers, `actions.ts`, `selectors.ts`.
Concept: pure (state,action)->state; split by domain; selectors derive.
Exercise: add a trivial action + reducer case + selector.
Done when: you can trace any dispatch to its reducer file.
