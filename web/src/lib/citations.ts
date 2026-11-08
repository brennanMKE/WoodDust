// Citation checks for the submission form (docs/contract.md §6). The Go server
// applies the same rules; these give instant feedback. Pure, so they're tested.

export type CitationType = 'doi' | 'pmid' | 'url';

const DOI_RE = /^10\.\d{4,9}\/\S+$/;

/** Accepts "10.1136/oem.48.3.201", "doi:10.…" or "https://doi.org/10.…"; returns the bare DOI. */
export function normalizeDOI(raw: string): string {
  return raw
    .trim()
    .replace(/^https?:\/\/(dx\.)?doi\.org\//i, '')
    .replace(/^doi:\s*/i, '');
}

export function isDOI(raw: string): boolean {
  return DOI_RE.test(normalizeDOI(raw));
}

/** Accepts "1234567", "PMID: 1234567" or a pubmed.ncbi.nlm.nih.gov URL; returns the digits. */
export function normalizePMID(raw: string): string {
  return raw
    .trim()
    .replace(/^https?:\/\/(www\.)?pubmed\.ncbi\.nlm\.nih\.gov\/(\d+)\/?$/i, '$2')
    .replace(/^pmid:?\s*/i, '');
}

export function isPMID(raw: string): boolean {
  return /^\d{1,9}$/.test(normalizePMID(raw)) && Number(normalizePMID(raw)) > 0;
}

export function isHttpURL(raw: string): boolean {
  const s = raw.trim();
  if (!/^https?:\/\//i.test(s)) return false;
  try {
    const u = new URL(s);
    return (u.protocol === 'http:' || u.protocol === 'https:') && u.hostname.includes('.');
  } catch {
    return false;
  }
}

/** Secondary compilations and forums: leads only, never a citable source (handoff §1, §2). */
export const LEADS_ONLY_DOMAINS = [
  'wood-database.com',
  'haz-map.com',
  'scrollsawvillage.com',
  'woodbin.com',
  'wooduchoose.com',
  'finewoodworking.com',
  'woodturner.org',
  'lumberjocks.com',
  'sawmillcreek.org',
  'woodworkingtalk.com',
  'reddit.com',
];

/** A warning for a leads-only URL, or null. Mirrors the server's check-url so the form can warn offline. */
export function leadsOnlyWarning(raw: string): string | null {
  if (!isHttpURL(raw)) return null;
  const host = new URL(raw.trim()).hostname.toLowerCase().replace(/^www\./, '');
  const hit =
    LEADS_ONLY_DOMAINS.find((d) => host === d || host.endsWith('.' + d)) ??
    (host.includes('forum') ? host : undefined);
  if (!hit) return null;
  return `${host} is a secondary compilation or forum. We use it only for leads, so please also give the study or agency page it relies on.`;
}

/** Validate one citation value; returns an error message or null. Empty is not an error here. */
export function citationError(type: CitationType, value: string): string | null {
  if (!value.trim()) return null;
  switch (type) {
    case 'doi':
      return isDOI(value) ? null : 'That doesn’t look like a DOI. DOIs start with 10. and contain a slash, e.g. 10.1136/oem.48.3.201.';
    case 'pmid':
      return isPMID(value) ? null : 'A PubMed ID is digits only, e.g. 2021399.';
    case 'url':
      return isHttpURL(value) ? null : 'Enter a full web address starting with https:// or http://.';
  }
}

/** Guess the type of a pasted value so a DOI pasted into the URL box still works. */
export function guessCitationType(value: string): CitationType | null {
  const v = value.trim();
  if (!v) return null;
  if (/^(doi:|10\.)/i.test(v) || /doi\.org\//i.test(v)) return 'doi';
  if (/^(pmid:?\s*)?\d+$/i.test(v) || /pubmed\.ncbi\.nlm\.nih\.gov\/\d+/i.test(v)) return 'pmid';
  if (/^https?:\/\//i.test(v)) return 'url';
  return null;
}

export function doiHref(doi: string): string {
  return `https://doi.org/${normalizeDOI(doi)}`;
}

export function pmidHref(pmid: string): string {
  return `https://pubmed.ncbi.nlm.nih.gov/${normalizePMID(pmid)}/`;
}
