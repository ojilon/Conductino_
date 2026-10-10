# 08-10 Rollback + storage migrate on update

Goal / Non-goal / Where in code / Concept / Flow / Rewrite options / Exercise / Done when.

Goal: updates never lose work. Design: backup DB -> stage new -> migrate -> verify -> swap; rollback on fail.
Exercise: simulate failed migrate, verify rollback.
Done when: drill logged.
