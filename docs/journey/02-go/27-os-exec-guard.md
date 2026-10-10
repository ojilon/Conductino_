# 02-27 os/exec guardrails (reveal + future shell)

Goal / Non-goal / Where in code / Concept / Flow / Rewrite options / Exercise / Done when.

Goal: exec without injection. Where: `filesystem.go:121 exec.Command("explorer","/select,",full)` (args, not shell). Rule: Resolve first, arg list, never sh -c with input. Exercise: audit every exec call. Done when: allowlist documented.
