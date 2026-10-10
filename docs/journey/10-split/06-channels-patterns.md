# 10-06 Channels + patterns (fan-out/fan-in, cancel)

Goal / Non-goal / Where in code / Concept / Flow / Rewrite options / Exercise / Done when.

Goal: cancel via context. Where: `Run(ctx,...)` already takes ctx; propagate to heavy workers.
Exercise: cancel a bulk extract mid-flight, verify prompt return.
Done when: ctx cancel tested.
