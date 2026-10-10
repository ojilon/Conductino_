# 02-26 Wrapping errors (fmt.Errorf %w, ReasonOf)

Goal / Non-goal / Where in code / Concept / Flow / Rewrite options / Exercise / Done when.

Goal: chain causes. Where: `services.ReasonOf(err)` maps to taxonomy; `%w` preserves cause. Exercise: wrap an os error with path context, verify ReasonOf still classifies. Done when: no bare `err` returns where context helps.
