# 04-02 Selection toolbar (Ask / Go deeper / Include / Save note)

Goal / Non-goal / Where in code / Concept / Flow / Rewrite options / Exercise / Done when.

Goal: own the toolbar. Where: `DocumentView.tsx` selection -> `TextSelection` -> toolbar actions.
Flow: select -> x/y + blockId + range -> dispatch -> aiController or changes.
Exercise: restyle toolbar, add keyboard shortcut, keep TextSelection shape.
Done when: toolbar works with canvas text layer too.
