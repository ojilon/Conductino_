/**
 * AI provider boundary.
 *
 * The React application NEVER talks to a concrete model provider.
 * It talks to `AIProvider` (src/types/domain.ts) only, obtained via
 * `getAIProvider()`.
 *
 * The single implementation is `WailsAIProvider`: a thin wrapper over the
 * Go backend (`backend/services/ai.go`, a Gemini-backed AIService). A call
 * goes TS → bound `App.StreamAIRequest` → Go runs the model → progress and
 * results come back as `AIEvent`s on the shared "ai://event" channel, which
 * this file demultiplexes back to the originating caller by `requestId`.
 * The API key lives only in Go (process env or git-ignored backend/.ai.env)
 * and never crosses into JS.
 *
 * Failure policy: there is no mock fallback. No bridge (browser mode), a
 * missing key, a network failure, or an unknown operation all surface as
 * `handlers.onError(...)`, and the UI renders an explicit error where the
 * answer was expected.
 *
 * See docs/ai-integration.md for the full contract.
 */

import type { AIHandlers, AIProvider, AIRequest, AIResult } from "../types/domain";
import { uid } from "../utils/helpers";
import { EventsOn } from "../../wailsjs/runtime/runtime";

/* ------------------------------------------------------------------ */
/* Wire shapes (mirror backend/models/models.go — keep in sync)        */
/* ------------------------------------------------------------------ */

/** Flattened Go AIRequest: TS `selection: {blockId, text}` arrives split. */
interface GoAIRequest {
  operation: string;
  query?: string;
  documentId?: string;
  sourceId?: string;
  changeId?: string;
  selection?: string;
  selectionText?: string;
  blockId?: string;
  requestId?: string;
}

interface GoAIEvent {
  type: string; // "phase" | "sources" | "done" | "error"
  requestId?: string;
  phase?: number;
  label?: string;
  sourceIds?: string[];
  /** JSON-encoded AIResult on "done". */
  payload?: string;
  /** Human-readable reason on "error". */
  message?: string;
}

interface GoBridge {
  StreamAIRequest(req: GoAIRequest): Promise<void>;
}

/** Non-null only inside the desktop build (wails dev / wails build). */
function goBridge(): GoBridge | null {
  const w = window as unknown as { go?: { frontend?: { App?: GoBridge } } };
  return w.go?.frontend?.App ?? null;
}

/* ------------------------------------------------------------------ */
/* Event demultiplexer (one shared "ai://event" channel)               */
/* ------------------------------------------------------------------ */

interface PendingCall {
  handlers: AIHandlers;
  /** Marks the call settled, clears the watchdog, drops the entry. */
  finish: (fn: () => void) => void;
}

const pending = new Map<string, PendingCall>();
let subscribed = false;

/** Subscribes to "ai://event" once per page load; routes by requestId. */
function ensureSubscribed(): void {
  if (subscribed) return;
  subscribed = true;
  EventsOn("ai://event", (ev: GoAIEvent) => {
    if (!ev || ev.requestId == null) return;
    const call = pending.get(ev.requestId);
    // Unknown id = already settled or cancelled: ignore late events.
    if (!call) return;
    switch (ev.type) {
      case "phase":
        call.handlers.onPhase(ev.phase ?? 0, ev.label ?? "Working");
        break;
      case "sources":
        call.handlers.onBrowseSources(ev.sourceIds ?? []);
        break;
      case "done":
        call.finish(() => call.handlers.onDone(parseResult(ev.payload)));
        break;
      case "error":
        call.finish(() => call.handlers.onError(ev.message || "AI unavailable now."));
        break;
      default:
        break;
    }
  });
}

/** Parses a "done" payload into AIResult; garbage becomes an empty result
 * (the controller renders that as "AI unavailable now", never a crash). */
function parseResult(payload: string | undefined): AIResult {
  if (!payload) return {};
  try {
    const parsed = JSON.parse(payload) as AIResult;
    return parsed && typeof parsed === "object" ? parsed : {};
  } catch {
    return {};
  }
}

/* ------------------------------------------------------------------ */
/* Wails provider — the only AIProvider in the app                     */
/* ------------------------------------------------------------------ */

/** Watchdog: a call that never settles (bridge dropped events) fails
 * loudly instead of spinning the activity strip forever. */
const CALL_TIMEOUT_MS = 120_000;

class WailsAIProvider implements AIProvider {
  readonly name = "Gemini (Go backend)";
  /** True when the desktop bridge exists; the Go side separately reports
   * whether a key is configured (a missing key arrives as onError). */
  readonly configured: boolean;

  constructor() {
    this.configured = goBridge() !== null;
  }

  run(request: AIRequest, handlers: AIHandlers): () => void {
    const app = goBridge();
    if (!app) {
      // Browser/mock mode: no bridge, no answers — honest error, and the
      // async shape is preserved so callers need no special case.
      const t = setTimeout(() => handlers.onError("AI needs the desktop app — run `wails dev`."), 0);
      return () => clearTimeout(t);
    }
    ensureSubscribed();
    const requestId = uid("aireq");
    let settled = false;
    const finish = (fn: () => void) => {
      if (settled) return;
      settled = true;
      clearTimeout(watchdog);
      pending.delete(requestId);
      fn();
    };
    const watchdog = setTimeout(
      () => finish(() => handlers.onError("AI timed out — try again.")),
      CALL_TIMEOUT_MS,
    );
    pending.set(requestId, { handlers, finish });
    // StreamAIRequest resolves when Go finishes emitting; results arrive via
    // the event channel above, so only a transport rejection lands here.
    // Note: cancel() drops the pending entry (late events are ignored) but
    // cannot abort the in-flight HTTP call — Go owns that lifetime.
    app
      .StreamAIRequest({
        operation: request.operation,
        query: request.query,
        documentId: request.documentId,
        sourceId: request.sourceId,
        changeId: request.changeId,
        selection: request.selection?.text,
        selectionText: request.selection?.text,
        blockId: request.selection?.blockId,
        requestId,
      })
      .catch((e: unknown) =>
        finish(() => handlers.onError(e instanceof Error ? e.message : "AI request failed to start.")),
      );
    return () => finish(() => undefined);
  }
}

/* ------------------------------------------------------------------ */
/* Provider registry                                                   */
/* ------------------------------------------------------------------ */

let activeProvider: AIProvider = new WailsAIProvider();

export function getAIProvider(): AIProvider {
  return activeProvider;
}

/** Override hook (tests / future providers). Production always uses the
 * Wails provider above. */
export function setAIProvider(provider: AIProvider): void {
  activeProvider = provider;
}
