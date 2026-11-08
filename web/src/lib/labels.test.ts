import { describe, expect, it } from 'vitest';
import { EVIDENCE, evidenceLabel, evidenceRank, HAZARD_NAMES, HAZARD_TYPES, hazardName } from './labels';

describe('evidence labels', () => {
  it('match the contract text exactly', () => {
    expect(evidenceLabel(1)).toMatchObject({ label: 'Confirmed in clinical testing', short: 'Confirmed' });
    expect(evidenceLabel(2)).toMatchObject({ label: 'Listed by an agency', short: 'Agency-listed' });
    expect(evidenceLabel(3)).toMatchObject({ label: 'Reported', short: 'Reported' });
    expect(evidenceLabel(0)).toMatchObject({ label: 'Applies to all wood dust', short: 'General' });
  });

  it('treats null or unknown levels as General', () => {
    expect(evidenceLabel(null).short).toBe('General');
    expect(evidenceLabel(7).short).toBe('General');
  });

  it('ranks specific evidence above general', () => {
    const sorted = [0, 3, 1, 2].sort((a, b) => evidenceRank(a) - evidenceRank(b));
    expect(sorted).toEqual([1, 2, 3, 0]);
    expect(evidenceRank(null)).toBe(4);
  });

  it('has an entry for every level', () => {
    expect(Object.keys(EVIDENCE).sort()).toEqual(['0', '1', '2', '3']);
  });
});

describe('hazard names', () => {
  it('match the contract display names', () => {
    expect(hazardName('respiratory_sensitizer')).toBe('Asthma / respiratory allergy');
    expect(hazardName('systemic_toxic')).toBe('Whole-body (systemic) effects');
    expect(hazardName('photosensitization')).toBe('Sun sensitivity (photosensitization)');
  });

  it('cover every hazard type in the display order', () => {
    expect([...HAZARD_TYPES].sort()).toEqual(Object.keys(HAZARD_NAMES).sort());
  });

  it('fall back to a readable name for unknown types', () => {
    expect(hazardName('new_thing')).toBe('new thing');
  });
});
