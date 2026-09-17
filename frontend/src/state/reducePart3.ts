/**
 * Document / change / summary reducer cases.
 */
import type { AppState, DocumentBlock, ID } from "../types/domain";
import type { Action } from "./actions";

function patchDoc(state: AppState, documentId: ID, fn: (doc: AppState["documents"][ID]) => AppState["documents"][ID]): AppState {
  const doc = state.documents[documentId];
  if (!doc) return state;
  return { ...state, documents: { ...state.documents, [documentId]: fn(doc) } };
}

export function reducePart3(state: AppState, action: Action): AppState | null {
  switch (action.type) {
    case "doc.add":
      if (state.documents[action.document.id]) return state;
      return { ...state, documents: { ...state.documents, [action.document.id]: action.document } };

    case "source.add":
      if (state.sources[action.source.id]) return state;
      return { ...state, sources: { ...state.sources, [action.source.id]: action.source } };

    case "doc.block.text":
      return patchDoc(state, action.documentId, (doc) => ({
        ...doc,
        blocks: doc.blocks.map((b) =>
          b.id === action.blockId ? { ...b, segments: [{ text: action.text }] } : b,
        ),
      }));

    case "doc.blocks.replace":
      return patchDoc(state, action.documentId, (doc) => ({
        ...doc,
        blocks: action.blocks,
      }));

    case "doc.title":
      return patchDoc(state, action.documentId, (doc) => ({
        ...doc,
        metadata: { ...doc.metadata, title: action.text },
      }));

    case "doc.highlight.add":
      return patchDoc(state, action.documentId, (doc) => ({
        ...doc,
        highlights: [...doc.highlights, action.highlight],
      }));

    case "change.propose": {
      const withChange: AppState = {
        ...state,
        changes: { ...state.changes, [action.change.id]: action.change },
      };
      if (!action.block) return withChange;
      return patchDoc(withChange, action.change.documentId, (doc) => {
        const block = action.block as DocumentBlock;
        const at = action.afterBlockId ? doc.blocks.findIndex((b) => b.id === action.afterBlockId) : -1;
        const blocks =
          at >= 0
            ? [...doc.blocks.slice(0, at + 1), block, ...doc.blocks.slice(at + 1)]
            : [...doc.blocks, block];
        return { ...doc, blocks };
      });
    }

    case "change.decide": {
      const change = state.changes[action.id];
      if (!change) return state;
      const next = patchDoc(state, change.documentId, (doc) => {
        if (action.status === "accepted" && change.type === "modify") {
          return {
            ...doc,
            blocks: doc.blocks.map((b) =>
              b.id === change.blockId ? { ...b, segments: [{ text: change.newContent }] } : b,
            ),
          };
        }
        if (
          (action.status === "accepted" && change.type === "delete") ||
          (action.status === "rejected" && change.type === "insert")
        ) {
          return { ...doc, blocks: doc.blocks.filter((b) => b.id !== change.blockId) };
        }
        return doc;
      });
      return {
        ...next,
        changes: { ...next.changes, [action.id]: { ...change, status: action.status } },
      };
    }

    case "change.revise": {
      const change = state.changes[action.id];
      if (!change) return state;
      return {
        ...state,
        changes: {
          ...state.changes,
          [action.id]: {
            ...change,
            newContent: action.newContent,
            highlightFragment: action.highlightFragment ?? change.highlightFragment,
          },
        },
      };
    }

    case "summary.addSource":
      return patchDoc(state, action.documentId, (doc) => {
        if (!doc.sourceIds || doc.sourceIds.includes(action.sourceId)) return doc;
        return { ...doc, sourceIds: [...doc.sourceIds, action.sourceId] };
      });

    default: return null;
  }
}
