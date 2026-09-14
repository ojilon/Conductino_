/**
 * AI provider boundary.
 *
 * The React application NEVER talks to a concrete model provider.
 * It talks to `AIProvider` (src/types/domain.ts) only, obtained via
 * `getAIProvider()`.
 *
 * To connect a real provider:
 *   1. Implement `AIProvider` (e.g. `OpenAIProvider`) in this file or a
 *      sibling file — call your API from `run()`, report progress via
 *      `handlers.onPhase(...)`, deliver results via `handlers.onDone(...)`.
 *   2. Call `setAIProvider(new OpenAIProvider(apiKey))` once at startup
 *      (App.tsx) when configuration is available.
 *   3. In a Wails build, the provider can instead forward requests to the
 *      Go backend (backend/services/ai.go) — same handlers, different transport.
 *
 * See docs/ai-integration.md for the full contract.
 */

import type { AIHandlers, AIProvider, AIRequest } from "../types/domain";
import {
  browseResultIds,
  explanationFor,
  expansionFor,
  verificationFor,
  relatedSourcesFor,
  insertionFor,
  revisionFor,
} from "../mock/aiContent";

/* ------------------------------------------------------------------ */
/* Mock provider — simulates streaming so the UI behaves for real      */
/* ------------------------------------------------------------------ */

interface Step {
  label: string;
  delay: number;
}

const BROWSE_STEPS: Step[] = [
  { label: "Searching", delay: 650 },
  { label: "Finding sources", delay: 750 },
  { label: "Comparing sources", delay: 850 },
  { label: "Ranking results", delay: 650 },
  { label: "Preparing useful sources", delay: 550 },
];

const EXPLAIN_STEPS: Step[] = [
  { label: "Reading selection", delay: 500 },
  { label: "Locating context", delay: 700 },
  { label: "Drafting explanation", delay: 800 },
];

const MERGE_STEPS: Step[] = [
  { label: "Extracting claim", delay: 550 },
  { label: "Aligning with summary", delay: 750 },
  { label: "Drafting insertion", delay: 700 },
];

const SIMPLE_STEPS: Step[] = [{ label: "Working", delay: 900 }];

function stepsFor(request: AIRequest): Step[] {
  switch (request.operation) {
    case "AI_SEARCH":
      return BROWSE_STEPS;
    case "AI_EXPAND":
    case "AI_VERIFY":
      return EXPLAIN_STEPS;
    case "AI_MERGE":
      return MERGE_STEPS;
    case "AI_REWRITE":
      return [{ label: "Revising draft", delay: 950 }];
    case "AI_EXPLAIN":
      return EXPLAIN_STEPS;
    case "AI_SUMMARIZE":
      return SIMPLE_STEPS;
  }
}

class MockAIProvider implements AIProvider {
  readonly name = "Mock provider (built-in)";
  readonly configured = false;

  run(request: AIRequest, handlers: AIHandlers): () => void {
    const timers: ReturnType<typeof setTimeout>[] = [];
    const steps = stepsFor(request);
    let elapsed = 0;

    steps.forEach((step, i) => {
      elapsed += step.delay;
      timers.push(
        setTimeout(() => handlers.onPhase(i, step.label), elapsed),
      );
    });

    timers.push(
      setTimeout(() => {
        try {
          if (request.operation === "AI_SEARCH") {
            handlers.onBrowseSources(browseResultIds());
          }
          const text = request.selection?.text ?? request.query ?? "";
          switch (request.operation) {
            case "AI_SEARCH":
              handlers.onDone({});
              break;
            case "AI_EXPAND":
              handlers.onDone({
                explanation: expansionFor(text),
                relatedSources: relatedSourcesFor(text),
              });
              break;
            case "AI_VERIFY":
              handlers.onDone({ explanation: verificationFor(text) });
              break;
            case "AI_MERGE":
              handlers.onDone({
                insertion: insertionFor(text, request.documentId),
              });
              break;
            case "AI_REWRITE":
              handlers.onDone({ revision: revisionFor(text) });
              break;
            default:
              handlers.onDone({
                explanation: explanationFor(text),
                relatedSources: relatedSourcesFor(text),
              });
          }
        } catch (err) {
          handlers.onError(err instanceof Error ? err.message : "AI request failed");
        }
      }, elapsed + 120),
    );

    return () => timers.forEach(clearTimeout);
  }
}

/* ------------------------------------------------------------------ */
/* Provider registry                                                   */
/* ------------------------------------------------------------------ */

let activeProvider: AIProvider = new MockAIProvider();

export function getAIProvider(): AIProvider {
  return activeProvider;
}

export function setAIProvider(provider: AIProvider): void {
  activeProvider = provider;
}
