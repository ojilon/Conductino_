import type { AppState } from "../types/domain";
import type { Action } from "./actions";
import { uid } from "../utils/helpers";

export function reducePart4(state: AppState, action: Action): AppState | null {
  switch (action.type) {
    case "selection.set":
      return { ...state, selection: action.selection };

    case "toast":
      return {
        ...state,
        toast: action.message ? { id: uid("toast"), message: action.message } : null,
      };

    case "settings.set":
      return { ...state, settingsOpen: action.open };

    default:
      return null;
  }
}
