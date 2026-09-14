# Lumen — AI integration

> **Where do I plug in the real AI API?** → implement `AIProvider` in
> `src/services/ai.ts` and call `setAIProvider(...)`. That is the entire
> contract. Nothing else in the app changes.

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
           | "AI_MERGE" | "AI_REWRITE" | "AI_VERIFY";
  query?: string;                 // browse query
  documentId?: string;
  sourceId?: string;
  selection?: { blockId: string; text: string };
  changeId?: string;              // for AI_REWRITE (revise a proposal)
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

## Plug-in steps (real API, frontend-side)

1. Create `src/services/ai/openai.ts` (or `anthropic.ts`, `ollama.ts`):

   ```ts
   class OpenAIProvider implements AIProvider {
     readonly name = "OpenAI (gpt-… )";
     readonly configured = true;
     constructor(private apiKey: string, private model: string) {}
     run(req: AIRequest, h: AIHandlers): () => void {
       // 1. translate AIRequest → provider call (fetch / SDK)
       // 2. stream tokens; call h.onPhase(i, label) at coarse steps
       //    (keep the 5 browse phases for AI_SEARCH; they map 1:1 to UI)
       // 3. h.onDone({ explanation | insertion | revision | relatedSources })
       // 4. return a function that aborts the request
     }
   }
   ```

2. At startup (`src/main.tsx` or `App.tsx`):
   `setAIProvider(new OpenAIProvider(key, model))` when a key is configured,
   otherwise leave the `MockAIProvider` in place (current behavior).

3. Key storage: read from the Go side (`backend/services/ai.go`) so the key
   never lives in frontend code — the Go `AIService` can own config + auth.

## Go-side provider (alternative or companion)

`backend/services/ai.go` defines the same boundary in Go:
`AIService.Run(ctx, req, sink)` with `AIEvent { phase | sources | done | error }`.
`app.StreamAIRequest` emits those events to the frontend (`ai://event`).
A frontend provider can be a thin wrapper over that event stream — useful
when the model runs locally (Ollama) or the key must stay server-side.

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

## Mock behavior (today)

`MockAIProvider` (same file) simulates streaming with timed phases
(~3.5 s for browse) and returns canned research answers from
`src/mock/aiContent.ts`. The UI is written to be indistinguishable from a
real provider — replacing the mock must not require touching any feature
file.
