// Keyboard behaviour for the search combobox, following the WAI-ARIA APG
// "editable combobox with list autocomplete" pattern. Pure so it can be tested.

export interface ComboState {
  open: boolean;
  /** Index of the highlighted option, or -1 when focus is only in the input. */
  active: number;
}

export type ComboAction =
  | { kind: 'none' }
  | { kind: 'select'; index: number } // go to this option
  | { kind: 'no_match' } // Enter with nothing to select
  | { kind: 'clear' }; // Escape on a closed list clears the input

export interface ComboResult {
  state: ComboState;
  action: ComboAction;
  /** Whether the key event should have its default prevented. */
  preventDefault: boolean;
}

export const initialCombo: ComboState = { open: false, active: -1 };

export function comboKey(state: ComboState, key: string, count: number): ComboResult {
  const none: ComboAction = { kind: 'none' };
  switch (key) {
    case 'ArrowDown': {
      if (count === 0) return { state, action: none, preventDefault: true };
      if (!state.open) return { state: { open: true, active: 0 }, action: none, preventDefault: true };
      const active = state.active >= count - 1 ? 0 : state.active + 1;
      return { state: { open: true, active }, action: none, preventDefault: true };
    }
    case 'ArrowUp': {
      if (count === 0) return { state, action: none, preventDefault: true };
      if (!state.open) return { state: { open: true, active: count - 1 }, action: none, preventDefault: true };
      const active = state.active <= 0 ? count - 1 : state.active - 1;
      return { state: { open: true, active }, action: none, preventDefault: true };
    }
    case 'Enter': {
      if (count === 0) return { state: { open: false, active: -1 }, action: { kind: 'no_match' }, preventDefault: true };
      const index = state.open && state.active >= 0 && state.active < count ? state.active : 0;
      return { state: { open: false, active: -1 }, action: { kind: 'select', index }, preventDefault: true };
    }
    case 'Escape': {
      if (state.open) return { state: { open: false, active: -1 }, action: none, preventDefault: true };
      return { state, action: { kind: 'clear' }, preventDefault: true };
    }
    case 'Tab':
      return { state: { open: false, active: -1 }, action: none, preventDefault: false };
    default:
      return { state, action: none, preventDefault: false };
  }
}

/** Typing changes the query: open the list if there are results, and drop the highlight. */
export function comboInput(count: number): ComboState {
  return { open: count > 0, active: -1 };
}
