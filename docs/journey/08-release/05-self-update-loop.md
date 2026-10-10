# 08-05 Self-update loop (improve it)

Goal / Non-goal / Where in code / Concept / Flow / Rewrite options / Exercise / Done when.

Goal: app updates itself safely. Design: version check -> download -> verify -> stage -> relaunch -> migrate storage.
Where: `backend/version/`, future updater service (you build it).
Exercise: prototype check-for-update against GitHub releases; stage only, no apply yet.
Done when: update never corrupts DB + rollback exists.
