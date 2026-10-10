# 01-02 Frontend data flow: event -> dispatch -> reducer -> render

Goal / Non-goal / Where in code / Concept / Flow / Rewrite options / Exercise / Done when.

Goal: trace one click end-to-end. Non-goal: AI streaming yet.
Where in code: `frontend/src/App.tsx:39 dispatch(mode.set)`, `state/appState.tsx:66 reducer` chains reduceWorkspace/reduceChat/reducePart1a..4, `state/selectors.ts`.
Concept (TS/React): `useReducer` = pure function (state, action) -> state; `createContext` shares it.
Flow: React event -> dispatch(action) -> reducer returns new AppState -> re-render from state.
Exercise: add a console.log in reducer, click mode switch, observe.
Done when: you can narrate mode switch without looking.
