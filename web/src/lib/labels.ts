// Display text from docs/contract.md §5. The Go server renders the same text.
import type { EvidenceLevel, HazardType } from './types';

export interface EvidenceLabel {
  level: EvidenceLevel;
  label: string;
  short: string;
  /** Plain-language explanation shown in tooltips and the methodology key. */
  explain: string;
}

export const EVIDENCE: Record<EvidenceLevel, EvidenceLabel> = {
  1: {
    level: 1,
    label: 'Confirmed in clinical testing',
    short: 'Confirmed',
    explain: 'A peer-reviewed report confirmed this wood with a challenge test, patch test or specific antibody test.',
  },
  2: {
    level: 2,
    label: 'Listed by an agency',
    short: 'Agency-listed',
    explain: 'An agency such as IARC, ACGIH, NIOSH, HSE or the EU names this wood or its group.',
  },
  3: {
    level: 3,
    label: 'Reported',
    short: 'Reported',
    explain: 'Case reports or handbook mentions, without confirmation testing.',
  },
  0: {
    level: 0,
    label: 'Applies to all wood dust',
    short: 'General',
    explain: 'True of every kind of wood dust, whatever the species.',
  },
};

export function evidenceLabel(level: number | null | undefined): EvidenceLabel {
  if (level === 1 || level === 2 || level === 3 || level === 0) return EVIDENCE[level];
  return EVIDENCE[0];
}

/** Strongest first: 1, 2, 3, then 0 (general). */
export function evidenceRank(level: number | null | undefined): number {
  if (level === 1 || level === 2 || level === 3) return level;
  return 4;
}

export const HAZARD_TYPES: HazardType[] = [
  'respiratory_sensitizer',
  'carcinogen',
  'skin_sensitizer',
  'hypersensitivity_pneumonitis',
  'respiratory_irritant',
  'skin_irritant',
  'eye_irritant',
  'systemic_toxic',
  'cardiac',
  'photosensitization',
  'splinter_infection',
];

export const HAZARD_NAMES: Record<HazardType, string> = {
  respiratory_sensitizer: 'Asthma / respiratory allergy',
  respiratory_irritant: 'Respiratory irritation',
  skin_sensitizer: 'Skin allergy',
  skin_irritant: 'Skin irritation',
  eye_irritant: 'Eye irritation',
  hypersensitivity_pneumonitis: 'Hypersensitivity pneumonitis',
  systemic_toxic: 'Whole-body (systemic) effects',
  cardiac: 'Heart effects',
  carcinogen: 'Cancer',
  splinter_infection: 'Splinter infection',
  photosensitization: 'Sun sensitivity (photosensitization)',
};

export function hazardName(type: string): string {
  return (HAZARD_NAMES as Record<string, string>)[type] ?? type.replace(/_/g, ' ');
}

/** ACGIH carcinogen categories (names of the categories themselves). */
export const ACGIH_CLASSES: Record<string, string> = {
  A1: 'Confirmed human carcinogen',
  A2: 'Suspected human carcinogen',
  A3: 'Confirmed animal carcinogen with unknown relevance to humans',
  A4: 'Not classifiable as a human carcinogen',
  A5: 'Not suspected as a human carcinogen',
};

export const TIER_NAMES: Record<string, string> = {
  A: 'Agency and regulatory listings',
  B: 'Peer-reviewed reviews',
  C: 'Primary studies',
  D: 'Technical references (names and non-health facts)',
};
