# 04-07 Panels: resize/collapse, widths in AppState

Goal / Non-goal / Where in code / Concept / Flow / Rewrite options / Exercise / Done when.

Goal: widths survive mode switches. Where: `domain.ts:402 ReaderUIState aiPanelWidth`, TopBar dispatch reader.ui.
Exercise: persist widths per workspace (not global).
Done when: switch modes, widths restore.
