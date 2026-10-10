# 10-12 Step 3 wiring (what actually landed in Go)

Goal: prove the heavy/network split compiles. Non-goal: switching default dispatch yet.

## What was added (Step 3 commit)

- `backend/bridge/bridge.go` — `Version=1`, Reason constants mirroring
  `models.OpenFailureReason` as plain strings (heavy stays dependency-free),
  `ParseRequest{PathToken,RelKey,KindHint}` + `ParseResult{BlocksJSON,Title,Kind,Reason,Detail}`.
  Raw bytes travel as a separate arg so the contract stays JSON-serializable for
  a future Rust sidecar (see `11-08`).
- `backend/internal/heavy/doc.go` + `parse_text.go` — pure `ParseText(relKey, raw)`:
  same caps (2 MiB, 1000 blocks), same blank-line paragraphs, same `#/##/###`
  headings, same FNV-1a content-addressed IDs (duplicated from
  `services/blockids.go` on purpose — heavy must never import services).
  No `os`, no `net/http`, no `exec`.
- `backend/internal/heavy/parse_text_test.go` — headings, stable IDs, too_large, empty.
- `backend/internal/netops/doc.go` — what STAYS network (HTTP, failover, streaming,
  prompts with live context, tool loop, meters).
- `backend/services/documents_heavy.go` — `openTextViaHeavy`: IO + caps here,
  parse delegated to `heavy.ParseText`, errors mapped to `OpenError` taxonomy.
  Old `openTextFile` untouched; default dispatch unchanged.
- `backend/services/documents_heavy_test.go` — `TestTextPathsAgree`: both txt paths
  byte-agree on a corpus file.

## Your next moves (do them yourself)

1. Run `go test ./backend/internal/heavy/ ./backend/services/ -run 'TestParse|TestTextPaths' -v`.
2. Extend corpus: your own .md files — both paths must still agree.
3. Flip dispatch for txt/md only when agreement holds for a week (one-line change
   in `documents.go:99` + keep legacy as fallback for one release).
4. Next split: docx paragraphs (heavy, pure ZIP+XML) — see `10-04`.

Done when: `go vet ./...` + `go build ./...` + `go test ./backend/...` green with
both paths present.
