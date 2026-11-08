import type { SourceMap, Source, SpeciesDetail, Hazard } from './types';
import { evidenceRank, HAZARD_TYPES } from './labels';

/** Number keys in order of first appearance: 1, 2, 3… */
export function numberKeys(keys: Iterable<string>): Map<string, number> {
  const out = new Map<string, number>();
  for (const k of keys) if (k && !out.has(k)) out.set(k, out.size + 1);
  return out;
}

/** Sources in footnote order, followed by any cited-but-unnumbered extras. */
export function orderedSources(numbers: Map<string, number>, sources: SourceMap | undefined): Source[] {
  if (!sources) return [];
  const list: Source[] = [];
  for (const k of numbers.keys()) {
    const s = sources[k];
    if (s) list.push({ ...s, key: k });
  }
  for (const [k, s] of Object.entries(sources)) {
    if (!numbers.has(k)) {
      numbers.set(k, numbers.size + 1);
      list.push({ ...s, key: k });
    }
  }
  return list;
}

/** Footnote keys in the order they appear in rendered HTML (href="#src-KEY"). */
export function keysFromHtml(html: string): string[] {
  return [...html.matchAll(/href="#src-([^"]+)"/g)].map((m) => decodeURIComponent(m[1]));
}

export interface HazardGroup {
  type: string;
  hazards: Hazard[];
  strongest: number;
}

/** Group hazards by type; strongest evidence first, both across and within groups. */
export function groupHazards(hazards: Hazard[]): HazardGroup[] {
  const map = new Map<string, Hazard[]>();
  for (const h of hazards) {
    const arr = map.get(h.type) ?? [];
    arr.push(h);
    map.set(h.type, arr);
  }
  const order = (t: string) => {
    const i = (HAZARD_TYPES as string[]).indexOf(t);
    return i < 0 ? 99 : i;
  };
  return [...map.entries()]
    .map(([type, hs]) => {
      const sorted = [...hs].sort((a, b) => evidenceRank(a.evidence_level) - evidenceRank(b.evidence_level));
      return { type, hazards: sorted, strongest: evidenceRank(sorted[0].evidence_level) };
    })
    .sort((a, b) => a.strongest - b.strongest || order(a.type) - order(b.type));
}

/** All source keys a species card cites, in on-page order. */
export function speciesCitationOrder(s: SpeciesDetail, groups: HazardGroup[]): string[] {
  const keys: string[] = [];
  for (const g of groups) for (const h of g.hazards) keys.push(...h.sources);
  for (const c of s.carcinogen_class ?? []) keys.push(c.source);
  for (const l of s.exposure_limits ?? []) keys.push(l.source);
  for (const c of s.known_compounds ?? []) keys.push(c.source);
  return keys;
}

export function sourceHref(s: Source): string | null {
  if (s.url) return s.url;
  if (s.doi) return `https://doi.org/${s.doi}`;
  if (s.pmid) return `https://pubmed.ncbi.nlm.nih.gov/${s.pmid}/`;
  return null;
}
