import { describe, expect, it } from 'vitest';
import { comboInput, comboKey, initialCombo } from './combobox';

describe('comboKey', () => {
  it('ArrowDown opens a closed list on the first option, then moves and wraps', () => {
    let r = comboKey(initialCombo, 'ArrowDown', 3);
    expect(r.state).toEqual({ open: true, active: 0 });
    expect(r.preventDefault).toBe(true);
    r = comboKey(r.state, 'ArrowDown', 3);
    expect(r.state.active).toBe(1);
    r = comboKey({ open: true, active: 2 }, 'ArrowDown', 3);
    expect(r.state.active).toBe(0);
  });

  it('ArrowUp opens on the last option and wraps from the first', () => {
    expect(comboKey(initialCombo, 'ArrowUp', 3).state).toEqual({ open: true, active: 2 });
    expect(comboKey({ open: true, active: 0 }, 'ArrowUp', 3).state.active).toBe(2);
    expect(comboKey({ open: true, active: -1 }, 'ArrowUp', 3).state.active).toBe(2);
  });

  it('arrows do nothing with no results', () => {
    expect(comboKey(initialCombo, 'ArrowDown', 0).state).toEqual(initialCombo);
  });

  it('Enter selects the highlighted option, or the top match when none is highlighted', () => {
    expect(comboKey({ open: true, active: 2 }, 'Enter', 3).action).toEqual({ kind: 'select', index: 2 });
    expect(comboKey({ open: true, active: -1 }, 'Enter', 3).action).toEqual({ kind: 'select', index: 0 });
    expect(comboKey(initialCombo, 'Enter', 3).action).toEqual({ kind: 'select', index: 0 });
    // A stale highlight beyond the current results falls back to the top match.
    expect(comboKey({ open: true, active: 5 }, 'Enter', 2).action).toEqual({ kind: 'select', index: 0 });
  });

  it('Enter with no results reports no match', () => {
    expect(comboKey(initialCombo, 'Enter', 0).action).toEqual({ kind: 'no_match' });
  });

  it('Escape closes an open list first, then clears', () => {
    const r = comboKey({ open: true, active: 1 }, 'Escape', 3);
    expect(r.state).toEqual({ open: false, active: -1 });
    expect(r.action.kind).toBe('none');
    expect(comboKey(r.state, 'Escape', 3).action.kind).toBe('clear');
  });

  it('Tab closes without preventing default; other keys pass through', () => {
    const t = comboKey({ open: true, active: 1 }, 'Tab', 3);
    expect(t.state.open).toBe(false);
    expect(t.preventDefault).toBe(false);
    const k = comboKey({ open: true, active: 1 }, 'a', 3);
    expect(k.preventDefault).toBe(false);
    expect(k.state).toEqual({ open: true, active: 1 });
  });
});

describe('comboInput', () => {
  it('opens only when there are results and clears the highlight', () => {
    expect(comboInput(4)).toEqual({ open: true, active: -1 });
    expect(comboInput(0)).toEqual({ open: false, active: -1 });
  });
});
