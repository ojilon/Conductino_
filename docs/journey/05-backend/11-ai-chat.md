# 05-11 ai/chat.go (tool loop)

Goal / Non-goal / Where in code / Concept / Flow / Rewrite options / Exercise / Done when.

Goal: multi-turn with tools. Where: host FS+Docs+SummaryText, truncate/propose from BlocksJSON (no format conversion in Backend.RunAI).
Exercise: cap loop iterations + log them.
Done when: loop can't spin forever.
