# 01-13 Transfer catalog (every handoff)

Goal / Non-goal / Where in code / Concept / Flow / Rewrite options / Exercise / Done when.

Goal: list every transfer: event->action, action->reducer, reducer->render, TS->Go, Go->event, dump->TS, blocks->docx, extract->cache->dump. Each entry: producer, payload shape, consumer, failure mode.
Exercise: pick 3, document failure mode. Start with: OpenFile->dump (Go writes, UI reads back by name; missing dump = honest toast), cache hit->dump (same file rewritten, same name), save->dump (snapshot of what was written).
Done when: catalog has 15+ entries.
