# Lumen — AI integration

> **Where does the real AI live?** → `backend/ai/`
> (`GeminiAIService`, key from `GEMINI_API_KEY` env or git-ignored
> `backend/.ai.env`). The frontend calls it through `WailsAIProvider`
> (`src/services/ai.ts` → `App.StreamAIRequest` → `"ai://event"`).
> Nothing else in the app changes.

## The contract

`src/types/domain.ts`:

```ts
interface AIProvider {
  readonly name: string;          // shown in Settings
  readonly configured: boolean;   // false while the mock is active
  run(request: AIRequest, handlers: AIHandlers): () => void;  // returns cancel
}

interface AIRequest {
  operation: "AI_SEARCH" | "AI_SUMMARIZE" | "AI_EXPLAIN" | "AI_EXPAND"
           | "AI_MERGE" | "AI_REWRITE" | "AI_VERIFY" | "AI_CHAT";
  query?: string;                 // browse query / chat message (@tokens stay in text)
  documentId?: string;
  sourceId?: string;
  selection?: { blockId: string; text: string };  // anchored region (chat included)
  changeId?: string;              // for AI_REWRITE (revise a proposal)
  mentionIds?: string[];          // resolved @doc targets — harness never guesses
  focusedChange?: { id: string; op: "insert"|"modify"|"delete"; blockId: string; oldContent?: string; newContent?: string };  // chat-targeted revise ("shorten this proposal")
}

interface AIHandlers {
  onPhase(index: number, label: string): void;          // streaming progress
  onBrowseSources(sourceIds: string[]): void;           // AI_SEARCH result set
  onDone(result: AIResult): void;                       // explanation / insertion / revision
  onError(message: string): void;
}
```

The handlers are **already wired to state** in `src/state/aiController.ts`:
phases update the progress steps + activity message; results update the
reading companion, propose `DocumentChange`s, or revise pending changes.

## Key setup (do this once, by hand)

1. Open `backend/.ai.env` (created from the checked-in template; the filled
   file is git-ignored, never committed).
2. Paste the Gemini key: `GEMINI_API_KEY=<key>`.
   (A live `GEMINI_API_KEY` env var works too and wins over the file.)
3. Restart `wails dev`. Settings → AI provider shows "Gemini (Go backend)".
   With no key, every AI action fails loudly ("AI key missing…") — no mock
   answers, anywhere.

## Go-side provider (live)

`backend/ai/service.go` implements `AIService` as `GeminiAIService`
(stdlib `net/http` against `generateContent`, model in
`defaultGeminiModel`): `Run(ctx, req, sink)` emits `phase` progress, then
exactly one terminal event — `done` with JSON `{ explanation | insertion |
revision }`, or `error` with a display-safe message. The key travels only in
the request header and never appears in logs or messages.
`App.StreamAIRequest` (`frontend/app.go`) emits those events to the frontend
(`ai://event`); `WailsAIProvider` (`src/services/ai.ts`) demultiplexes them
back to the caller by `requestId`.

Status: WIRED. All `aiController.ts` runners go through the Go provider.
`AI_SEARCH` (browser web search) has no backing service and returns an
`error` event by design — the browse panel shows it, nothing is fabricated.

Boundary note (resolved, do not regress): TS `AIRequest.selection` is
`{ blockId, text }` (`domain.ts`); Go receives it flattened as
`SelectionText`/`BlockID` (`backend/models/models.go`), translated by
`WailsAIProvider`. `RequestID` on request and every event correlates
concurrent calls on the shared channel.

## What each operation must produce

| Operation | Trigger | Expected `onDone` payload |
|---|---|---|
| `AI_SEARCH` | AI Browse query | `onBrowseSources(ids)` → ranked source cards |
| `AI_EXPLAIN` | selection → "Ask AI" | `explanation` (+ `relatedSources`) |
| `AI_VERIFY` | selection → "Explain" | `explanation` (cross-reference verdict) |
| `AI_EXPAND` | selection → "Go deeper" | longer `explanation` |
| `AI_MERGE` | "Include in summary" | `insertion { text, citation }` → becomes a pending `DocumentChange` |
| `AI_REWRITE` | "Revise again" | `revision` (replacement `newContent`) |
| `AI_SUMMARIZE` | (future) summarize whole doc | `explanation` / future structured output |
| `AI_CHAT` | chat composer (`@doc` + selection) | `explanation` + `proposals[] { op, targetBlockId, oldContent, newContent }` → pending `DocumentChange`s (insert/modify/delete) |

`@doc` resolution (`state/mentions.ts`): composer tokens resolve against
workspace documents to `mentionIds`; a mentioned summary overrides the
workspace primary as proposal target. The live text selection rides along
as the region anchor. Proposals are full edit power — insert, modify, or
delete — never append-only; the user gatekeeps every one (accept/revise).

Source ranking (`rank`/`relevance`) is attached by the app from the returned
id order; the provider can send richer source objects later without UI
changes if the `Source` type is extended.

## Activity tracking guarantees

Every `run()` starts with a `AIActivity` record (`state/aiController.ts` →
`activity.start`). The UI answers at any moment:

- *What is the AI doing right now?* → the single activity with `status: "running"` (`ActivityStrip`).
- *What source is it working on?* → `activity.sourceId`.
- *What document will it modify?* → `activity.documentId`.
- *What did it propose?* → `activity.changeIds` → `state.changes`.
- *History?* → `state.aiActivities` (newest first) — surfaced in the AI Reading panel.

## Failure policy (no mock fallback)

There is no mock provider anymore. Missing key, network failure, empty model
response, unknown operation, and browser-mode use (no desktop bridge) all
reach the UI as `onError`, rendered where the answer was expected
("AI unavailable now." / explicit toasts) — see `aiController.ts`. The app
starts empty (no seeded documents) — the initial state in `appState.tsx`
is blank chrome, not an AI fallback.
