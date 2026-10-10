# Multi-provider API plan — Gemini first, then dual / failover

> Branch: `feature/academic-harness-reader-ai`
> Depends on: `02-backend-structure.md` (`services/ai/` package), `01-architecture-harness.md` (context packs, tools).
> Status: **partially landed** — `ModelBackend` + Gemini/Groq/OpenRouter backends + failover + cost classes + semaphore + explain cache + usage/audit telemetry are in (`backend/services/ai/{service,backend,openai_compat,policy,usage}.go`); Settings shows live meters via `App.AIMeters`. Still open: tertiary OpenRouter default, map-reduce multi-source, soft daily budgets, persisted UsageEvents.

## 1. Why this exists

The academic harness will send **large context packs** (selection + document window + other sources + summary snapshot + tool results). That collides with **free-tier** limits on a single provider:

| Pressure | Effect on one free key |
|---|---|
| RPM (requests/minute) | Chat + tool loops + parallel “explain on highlight” burn the budget |
| RPD (requests/day) | Full research day can exhaust daily quota |
| TPM / large inputs | Multi-file context packs hit token-per-minute caps even at low RPM |
| Outages / 429 / 503 | One provider down → entire reader AI dead |

**Gemini remains the primary** (already wired, strong free Flash tier, 1M context). This plan defines:

1. How to run **efficiently on one API** (Gemini).
2. How to optionally run **two (or more)** providers with clear switching.
3. How to **distribute work** and **fail over** without breaking the existing `AIProvider` / event contract.
4. A short inventory of **free / low-cost APIs** worth plugging in later.

Limits below are **indicative (2026)** and change often — implement as **configurable quotas**, not hard-coded constants.

## 2. Operating modes (user-switchable)

Settings (and/or `backend/.ai.env`) expose an explicit mode:

| Mode | Behavior |
|---|---|
| **`single`** | Only the primary provider (default: Gemini). Maximize quality of one quota. |
| **`dual`** | Primary + secondary. Router assigns tasks by policy; both may run **almost concurrently** on independent requests. |
| **`failover`** | Prefer primary; on rate-limit / timeout / 5xx, retry secondary (and optionally tertiary). |
| **`auto`** | Start as dual when two keys exist; degrade to single if secondary missing; apply failover on 429. |

Switching modes must **not** require a rebuild: read config at process start + optional hot reload of non-secret fields.

```
# backend/.ai.env (illustrative)
AI_MODE=auto                    # single | dual | failover | auto
GEMINI_API_KEY=...
GROQ_API_KEY=...                # optional secondary
OPENROUTER_API_KEY=...          # optional tertiary / free-model pool
AI_PRIMARY=gemini
AI_SECONDARY=groq
```

Frontend Settings shows: configured providers, mode, **usage meters** (today’s RPM/RPD/tokens if known).

## 3. Efficient single-API management (Gemini-first)

Before multi-provider complexity, squeeze one key hard. All of this lives in `services/ai/`.

### 3.1 Request classification

Tag every `AIRequest` with a **cost class** before send:

| Class | Examples | Budget posture |
|---|---|---|
| **S** (small) | Explain selection only, revise one sentence | Prefer smallest model / lowest max tokens |
| **M** (medium) | Explain + local window, short chat turn | Default Flash |
| **L** (large) | Multi-source context pack, summary improve, tool loop | Cap context aggressively; maybe queue |

### 3.2 Context budget (already planned in context.go)

- Estimate tokens **before** call (chars/4 heuristic + optional provider count API).
- Drop layers in order: extra open sources → outline → local window → keep selection + user prompt.
- Never send full multi-MB extracts; extract → summarize locally or chunk.
- Cap `maxOutputTokens` per operation (already partially done in `ai.go`).

### 3.3 Concurrency & queue

- Global **in-process semaphore** for outbound model calls (e.g. 1–2 concurrent on free tier).
- Queue excess work with UI phase “Waiting for AI capacity…”.
- Coalesce identical in-flight requests (same selection hash + op) where safe.

### 3.4 Caching

- Short TTL cache for identical `AI_EXPLAIN` / `AI_VERIFY` on same text (workspace-scoped).
- Do **not** cache chat turns or merge proposals that must stay fresh.

### 3.5 Backoff on 429

- Honor `Retry-After` when present.
- Exponential backoff with jitter; surface honest “rate limited, try in Ns” in the panel.
- Track consecutive failures → optional temporary “provider cooling” flag.

### 3.6 Model tiering inside Gemini

When multiple Gemini models are free:

- **Flash-Lite / cheapest** → S class, high volume.
- **Flash** → default M/L explanations and chat.
- **Pro** (if free quota exists) → rare, high-stakes summary merges only.

Configurable map: `operation + class → model id`.

## 4. Dual / multi provider: work distribution

### 4.1 Provider interface (Go)

Inside `services/ai/`, every backend implements the same internal contract:

```go
type ModelBackend interface {
    Name() string
    Configured() bool
    // Generate one completion; no streaming required at this layer if
    // the outer Run streams phases itself.
    Generate(ctx context.Context, req CompletionRequest) (CompletionResult, error)
    // Optional: CountTokens, Healthy()
}
```

Gemini stays native REST; Groq / OpenRouter use **OpenAI-compatible** chat completions (one shared client path).

Outer `AIService.Run` / `chat.go` never call Gemini HTTP directly — only via `ModelBackend`.

### 4.2 Routing policy

Router inputs:

- Operating mode
- Cost class (S/M/L)
- Estimated input tokens
- Operation (`AI_EXPLAIN` vs `AI_CHAT` vs tool-heavy)
- Per-provider **usage snapshot** (RPM window, RPD, recent errors)
- Health (last success / cooling until)

Suggested default policy for **`dual`**:

| Condition | Route |
|---|---|
| Class **S**, low tokens | Secondary (e.g. Groq — fast, cheap on free tier) |
| Class **M** explanation / verify | Primary (Gemini — quality) |
| Class **L** / long context / multi-source | Primary if under TPM budget; else **split** (see 4.3) or refuse with “context too large for free tier” |
| Tool loop steps (many small calls) | Secondary for tool-side helpers; Primary for final synthesis |
| Primary in cooling / 429 | Secondary (`failover`) |
| Both cooling | Error event; do not invent answers |

### 4.3 Almost-concurrent multi-API work

True “same user turn, two APIs” cases that are worth supporting:

1. **Map-reduce over sources**  
   - Large folder: for each source (or chunk), secondary does a **short extractive summary** (S class).  
   - Primary merges summaries into one explanation or summary edit.  
   - Parallelism limited by semaphores on each backend.

2. **Race for latency** (optional, off by default)  
   - Fire S-class explain on secondary and primary; take first success; cancel the other.  
   - Wasteful on free tiers — only if user enables “fast race”.

3. **Failover mid-chat**  
   - Thread history stays in **our** storage; next turn can switch provider.  
   - Prompt must be provider-agnostic (no Gemini-only features in the shared template).

**Do not** interleave two providers inside one opaque tool loop without recording which backend produced which tool result (for debugging and usage).

### 4.4 Size-aware decisions (sources)

Before routing:

```
estimateSourceSize(path|id) → bytes, approxTokens, format
```

Rules of thumb (tunable):

| Approx tokens | Behavior |
|---|---|
| &lt; 2k | Full text in context pack; single call |
| 2k–20k | Window + outline; single Primary call |
| 20k–100k | Chunked secondary summaries → Primary merge |
| &gt; 100k | Require explicit user scope (@file section / selection only); refuse full-file dump |

PDF/DOCX: use **extracted text size**, not file bytes on disk.

## 5. Usage tracking (tokens, RPM, RPD)

### 5.1 What to record

Per completed (or failed) call, append a **UsageEvent** (backend storage):

```text
UsageEvent {
  id, ts, workspaceId?, activityId?,
  provider, model,
  operation, costClass,
  inputTokensEst, outputTokensEst,   // exact if API returns usage
  inputTokensExact?, outputTokensExact?,
  latencyMs, status: ok|rate_limited|error|timeout,
  errorCode?
}
```

### 5.2 Live meters

In-memory sliding windows per provider:

- Requests in last 60s (RPM)
- Requests since local midnight or provider reset (RPD approximation)
- Tokens in last 60s (TPM approximation)

Expose to UI via Settings / small strip: “Gemini ~12/15 RPM · ~400 RPD used”.

Exact provider headers (`x-ratelimit-*`) when present should update meters.

### 5.3 Budgets

Optional soft caps in config:

```text
AI_DAILY_TOKEN_BUDGET_PRIMARY=500000
AI_DAILY_REQUEST_BUDGET_SECONDARY=800
```

When exceeded → refuse new **L** class; allow **S** until hard provider 429.

## 6. Failover & “one is down”

Order:

1. Try preferred backend for this route.
2. On **429 / 503 / timeout / network**: mark cooling (e.g. 30–120s), emit phase “Switching provider…”, retry secondary **once** with same logical request (rebuilt prompt).
3. If secondary also fails → `onError` with clear message (which providers tried).
4. Never silently return empty explanation as success.

Chat threads: store `provider` on the assistant message for transparency (“Answered via Groq after Gemini rate limit”).

## 7. Candidate APIs to add (inventory)

Prioritize **no credit card**, OpenAI-compatible where possible, and suitability for **academic text** (not only code).

| Provider | Role suggestion | Free-tier notes (verify before ship) | API shape |
|---|---|---|
| **Google Gemini** (current) | **Primary** — quality, long context, multimodal later | Free Flash-class models; RPM/RPD/TPM vary by model (~10–15 RPM, hundreds–1.5k RPD typical); 1M context on Flash | Native generateContent (already) |
| **Groq** | **Secondary** — speed, small/medium calls, parallel map steps | No card; ~30 RPM; per-model RPD/TPM (often ~1k RPD on larger models); very fast | OpenAI-compatible `api.groq.com/openai/v1` |
| **OpenRouter** (`:free` models) | **Tertiary / pool** — many models, one key | Free model routes; low RPD (often ~50/day unless credits); good for experiments | OpenAI-compatible |
| **Cloudflare Workers AI** | Optional secondary | Daily neuron budget; edge latency | Provider-specific / OpenAI-ish |
| **Cerebras** | Optional speed tier | Free tier with RPM/token caps | OpenAI-compatible often |
| **Mistral** (experiment / free tier) | Optional quality alt | Account limits vary | Mistral / OpenAI-compatible |
| **NVIDIA NIM** | Optional model zoo | Phone verify; evaluation limits | OpenAI-compatible |
| **Local Ollama** (future) | Offline primary | Unlimited local; user hardware | Local HTTP |

**Not free API (but document for paid path):** OpenAI, Anthropic Claude — excellent quality; add only when user supplies paid keys.

**Avoid as “primary free production”:** scrapers, unstable anonymous gateways, keys that ban commercial use without review.

### Recommended adoption order

1. **Gemini only** + efficiency (Phase 1–3).
2. **Groq secondary** + `failover` / `dual` routing (after `ModelBackend` split).
3. **OpenRouter free** as optional tertiary when both rate-limit.
4. Local Ollama when offline research matters.

## 8. Package layout touchpoints

```
backend/services/ai/
├── service.go          # Run, mode, outer streaming
├── router.go           # NEW: mode + class + health → backend
├── usage.go            # NEW: meters, UsageEvent, budgets
├── backends/
│   ├── gemini.go       # move current client here
│   ├── openai_compat.go  # shared Groq/OpenRouter client
│   └── registry.go     # Configured backends from env
├── context.go
├── prompts.go
├── chat.go
└── tools*.go
```

Frontend:

- Settings: list providers, mode toggle, usage meters.
- No vendor SDKs in React — still only `AIProvider` / Wails events.

## 9. Security & privacy notes

- Keys only in env / `.ai.env` (git-ignored); never in renderer.
- Free tiers often allow **prompt logging for product improvement** (Gemini free). Document this in Settings; prefer paid tier later for “no training” if required.
- Do not send whole disk paths or PII beyond what’s in the research files the user opened.

## 10. Phasing relative to other plans

| When | What |
|---|---|
| With Phase 1 (`ai/` split) | Introduce `ModelBackend` + Gemini-only implementation; usage events optional |
| With Phase 3 (context packs) | Cost class + token estimates + single-provider queue |
| After Phase 4–5 (chat/tools) | Dual routing becomes valuable (many small tool calls) |
| Separate small PR | Groq backend + `AI_MODE=failover` |
| Later | OpenRouter tertiary, Settings meters UI, map-reduce multi-source |

## 11. Success criteria

1. **Single mode:** app runs only on Gemini; large contexts are budgeted; 429s show honest wait/retry; usage visible.
2. **Failover mode:** killing/rate-limiting Gemini still allows S/M answers via secondary when configured.
3. **Dual mode:** map-style multi-source jobs can use secondary for partial summaries and primary for merge, with both usage tracked.
4. **Switching modes** is config-only; no frontend rewrite per vendor.
5. Missing secondary key → behave as single without errors on startup.

## 12. Open decisions

1. Default secondary: **Groq** vs OpenRouter free pool?
2. Soft daily token budget default numbers for academic day use?
3. Whether map-reduce multi-source is automatic or only when user selects ≥N files?
4. Persist UsageEvents in SQLite vs ring buffer in memory until Phase 6 storage?
