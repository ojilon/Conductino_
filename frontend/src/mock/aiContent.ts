/**
 * Mock AI response content, consumed by MockAIProvider
 * (src/services/ai.ts). Replaced automatically when a real provider
 * is registered — these functions are never called on that path.
 */

import { documents } from "./data";

export function browseResultIds(): string[] {
  return ["ws-1", "ws-2", "ws-3", "ws-4", "ws-5"];
}

function headWords(text: string, n: number): string {
  const words = text.replace(/\s+/g, " ").trim().split(" ");
  return words.slice(0, n).join(" ");
}

export function relatedSourcesFor(_text: string): { title: string; meta: string }[] {
  return [
    {
      title: "Oxidative Phosphorylation and ATP Synthesis",
      meta: "Nelson et al. (2021) · Nature Reviews Molecular Cell Biology",
    },
    {
      title: "Structure and Mechanism of ATP Synthase",
      meta: "Petersen & Junge (2019) · Biochimica et Biophysica Acta",
    },
  ];
}

export function explanationFor(text: string): string {
  if (/proton gradient/i.test(text)) {
    return (
      "The proton gradient creates a flow of H+ ions back into the mitochondrial matrix (or stroma) through ATP synthase. " +
      "This flow provides the energy for ATP synthase to convert ADP and inorganic phosphate into ATP, a process called chemiosmosis."
    );
  }
  return (
    `The selected passage — “${headWords(text, 12)}…” — states a specific mechanism. In the context of the document, ` +
    "it supports the central claim that an electrochemical gradient, rather than a high-energy intermediate, carries the coupling. " +
    "ATP synthase is the membrane protein that spends that gradient to phosphorylate ADP."
  );
}

export function expansionFor(text: string): string {
  return (
    explanationFor(text) +
    " Two quantitative details worth noting: the gradient has a chemical component (ΔpH) and an electrical component (Δψ), " +
    "and the free energy available to the synthase is set by ΔG = 2FΔψ − 2.303RT·n·ΔpH. Recent cryo-EM work shows that the " +
    "proton stoichiometry per ATP varies with c-ring size, which is one reason efficiency differs across species."
  );
}

export function verificationFor(_text: string): string {
  return (
    "Cross-referenced against the sources open in this session. The claim is consistent with Mitchell (1961) and with " +
    "Nelson et al. (2021); no conflicting statement was found in the AI Browse results. " +
    "Confidence: high — 2 supporting sources, 0 contradicting."
  );
}

function citationFor(documentId: string | undefined): string {
  const doc = documentId ? documents[documentId] : undefined;
  if (doc?.metadata.author && doc?.metadata.year) {
    return `(${doc.metadata.author.replace(/\s?et al\.$/i, " et al.")}, ${doc.metadata.year})`;
  }
  return "(session source)";
}

export function insertionFor(text: string, documentId: string | undefined): { text: string; citation: string } {
  const doc = documentId ? documents[documentId] : undefined;
  const from = doc ? `— From ${doc.metadata.title}: “${headWords(text, 12)}…”` : `— “${headWords(text, 12)}…”`;
  return { text: from, citation: citationFor(documentId) };
}

export function revisionFor(text: string): string {
  let out = text;
  if (out.includes("Recent research suggests")) {
    out = out.replace("Recent research suggests", "Recent work further indicates");
  } else if (out.startsWith("— From")) {
    out = out.replace("— From", "— Selected from") + " (revised phrasing)";
  } else {
    out = out + " The authors note that this coupling remains an active area of investigation.";
  }
  return out === text ? `Revised: ${text}` : out;
}
