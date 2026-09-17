import type { AppState } from "../types/domain";
import type { Action } from "./actions";

export function reducePart1b(state: AppState, action: Action): AppState | null {
  switch (action.type) {
    case "activity.start":
      return { ...state, aiActivities: [action.activity, ...state.aiActivities] };

    case "activity.update":
      return {
        ...state,
        aiActivities: state.aiActivities.map((a) => (a.id === action.id ? { ...a, ...action.patch } : a)),
      };

    /* ---------------- sources ---------------- */

    default: return null;
  }
}
