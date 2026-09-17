/**
 * Chat thread reducer cases (Phase 4).
 */
import type { AppState } from "../types/domain";
import type { Action } from "./actions";

export function reduceChat(state: AppState, action: Action): AppState | null {
  switch (action.type) {
    case "chat.ensure": {
      const th = action.thread;
      const existing = state.chat.byId[th.id];
      if (existing) {
        return {
          ...state,
          chat: {
            activeId: state.chat.activeId ?? th.id,
            byId: state.chat.byId,
          },
        };
      }
      return {
        ...state,
        chat: {
          activeId: state.chat.activeId ?? th.id,
          byId: { ...state.chat.byId, [th.id]: th },
        },
      };
    }

    case "chat.setActive": {
      if (!state.chat.byId[action.id]) return state;
      return { ...state, chat: { ...state.chat, activeId: action.id } };
    }

    case "chat.append": {
      const th = state.chat.byId[action.threadId];
      if (!th) return state;
      return {
        ...state,
        chat: {
          ...state.chat,
          activeId: state.chat.activeId ?? action.threadId,
          byId: {
            ...state.chat.byId,
            [action.threadId]: {
              ...th,
              messages: [...th.messages, action.message],
              updatedAt: action.message.createdAt,
            },
          },
        },
      };
    }

    case "chat.clear": {
      const th = state.chat.byId[action.threadId];
      if (!th) return state;
      return {
        ...state,
        chat: {
          ...state.chat,
          byId: {
            ...state.chat.byId,
            [action.threadId]: { ...th, messages: [], updatedAt: Date.now() },
          },
        },
      };
    }

    default:
      return null;
  }
}
