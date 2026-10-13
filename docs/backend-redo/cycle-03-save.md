# Cycle 3: save/write-back in `backend/save` (txt/md only)

Goal: close the loop — blocks JSON back to a file on disk. Mirror image of
`read`: same jail idea, same caps, same dump dir. Two files, zero edits to
old code. Borrowed from `extract/docx.go` WriteSummaryDOCXPath (jail-on-write)
and `mirror/mirror.go` (MkdirAll-then-write, never crash on missing dirs).

## Files you create

1. `backend/save/save.go` — `SaveText()` + `resolveTarget()` + block renderer.
2. `backend/save/save_test.go` — round-trip, list rendering, escape, ext, bad JSON.

## The model

- `read` = file → blocks. `save` = blocks → file. Same token world.
- Difference from `read.Resolve`: the file may NOT exist yet (new file!),
  so we jail-check the token, then `MkdirAll` the PARENT instead of
  Statting the file. Missing folders appear; existing neighbours untouched.
- Rendering (blocks → text): heading → `# ` prefix, list → `- ` rows,
  `page` breaks skipped (render anchors, not content), blank parts dropped
  (the parser never stores blanks, so dropping keeps files clean).
- Same `MaxWriteBytes` cap as read: render first, measure, THEN write —
  never half-write a giant file.
- After writing, dump the saved snapshot to the same `extract_dump` dir:
  what you wrote is what a reopen must return (round-trip test proves it).

## Go concepts you meet

- `json.Unmarshal` into a small local struct: unknown fields ignored, so
  frontend extras (`em`, `strong`) pass through harmlessly.
- `switch` with `continue`: `page` blocks contribute nothing.
- `strings.Join(parts, "\n\n")`: build-then-join beats print-as-you-go —
  no trailing-separator bugs.
- Cross-package reuse: `save` imports `read` (`*read.Fail`) AND uses
  `read.ParseText`/`read.OpenFile` in tests. New packages compose; old ones frozen.

## Run

```
go vet ./backend/save/
go test ./backend/save/ -v
```

## Done when

- 5 tests pass, including round-trip: `ParseText → SaveText → OpenFile`
  returns byte-identical blocks (stable IDs make this possible — Cycle 1 pays off).
- You can explain why saving reuses the dump dir (answer: the dump is the
  contract — "what left Go" — for both directions).
- Next: Cycle 4 (Wails wire-up, append-only).
