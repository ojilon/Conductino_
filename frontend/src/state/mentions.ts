/**
 * @doc mention resolution (chat composer → explicit document ids).
 *
 * The user types `@something`; on send we resolve each token against the
 * workspace documents (title, then label/path match, case-insensitive) so
 * the harness never guesses which document was meant. The raw `@token`
 * stays in the query text — these ids ride alongside as ground truth.
 * A mentioned summary overrides the workspace primary as proposal target.
 */

import type { Document, ID } from "../types/domain";

/** Raw @tokens in message order (`@foo-bar` → `foo-bar`). */
export function parseMentionTokens(text: string): string[] {
  const out: string[] = [];
  const re = /@([\p{L}\p{N}][\p{L}\p{N}._-]*)/gu;
  for (const m of text.matchAll(re)) {
    if (m[1]) out.push(m[1]);
  }
  return out;
}

function docHaystack(d: Document, labelOf: (d: Document) => string): string[] {
  const meta = d.metadata;
  return [
    meta.title,
    labelOf(d),
    meta.path ?? "",
    d.id,
  ]
    .map((s) => s.toLowerCase())
    .filter(Boolean);
}

/**
 * Resolve @tokens to document ids. First case-insensitive substring hit
 * wins per token (title preferred via haystack order); unknown tokens are
 * skipped so a typo never misdirects the harness.
 */
export function resolveMentions(
  text: string,
  docs: Document[],
  labelOf: (d: Document) => string,
): ID[] {
  const tokens = parseMentionTokens(text);
  if (!tokens.length || !docs.length) return [];
  const ids: ID[] = [];
  for (const tok of tokens) {
    const needle = tok.toLowerCase();
    const hit = docs.find((d) => docHaystack(d, labelOf).some((h) => h.includes(needle)));
    if (hit && !ids.includes(hit.id)) ids.push(hit.id);
  }
  return ids;
}
