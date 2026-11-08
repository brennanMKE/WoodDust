// Submission form state (docs/contract.md §6, PLAN.md §6) as a pure reducer,
// so the rules are unit tested and the Svelte view stays thin.
import {
  citationError,
  normalizeDOI,
  normalizePMID,
  type CitationType,
} from './citations';

export type SubmissionKind = 'new_species' | 'correction' | 'new_source' | 'reaction_report';

export const KINDS: { id: SubmissionKind; label: string; hint: string }[] = [
  { id: 'new_species', label: 'Suggest a new species', hint: 'A wood we haven’t listed, with a source for its hazards.' },
  { id: 'correction', label: 'Correct a listed species', hint: 'Something on a species card is wrong or out of date.' },
  { id: 'new_source', label: 'Add a better source', hint: 'A primary study or agency page for a claim we already make.' },
  {
    id: 'reaction_report',
    label: 'Report a reaction',
    hint: 'You or someone you know reacted to a wood. Never published.',
  },
];

export function isKind(v: unknown): v is SubmissionKind {
  return typeof v === 'string' && KINDS.some((k) => k.id === v);
}

export function kindLabel(kind: string): string {
  return KINDS.find((k) => k.id === kind)?.label ?? kind.replace(/_/g, ' ');
}

export const DETAILS_MIN = 20;
export const DETAILS_MAX = 4000;
export const MAX_CITATIONS = 8;

/** Character count the way the server counts it (Unicode code points). */
export function charCount(s: string): number {
  return [...s].length;
}

export interface TargetSpecies {
  slug: string;
  name: string;
  botanical_name?: string;
}

export interface CitationRow {
  id: number;
  type: CitationType;
  value: string;
  text: string;
}

export type Banner =
  | { kind: 'rate_limited'; retryAfter: number | null }
  | { kind: 'error'; message: string }
  | null;

export interface SubmitResult {
  id?: number;
  reference: string;
  statusUrl: string;
  warnings: string[];
}

export interface FormState {
  kind: SubmissionKind | null;
  target: TargetSpecies | null;
  speciesName: string;
  botanicalName: string;
  hazardType: string;
  details: string;
  email: string;
  website: string; // honeypot
  citations: CitationRow[];
  nextId: number;
  phase: 'editing' | 'submitting' | 'done';
  /** Field key -> message. Keys: kind, target, species_name, details, email, citations, citations.<index>. */
  errors: Record<string, string>;
  /** Leads-only warnings by citation row id. */
  urlWarnings: Record<number, string>;
  banner: Banner;
  result: SubmitResult | null;
}

export interface InitOptions {
  kind?: string | null;
  target?: TargetSpecies | null;
  name?: string;
}

export function initialForm(opts: InitOptions = {}): FormState {
  let kind: SubmissionKind | null = isKind(opts.kind) ? opts.kind : null;
  if (!kind && opts.target) kind = 'correction';
  if (!kind && opts.name) kind = 'new_species';
  return {
    kind,
    target: opts.target ?? null,
    speciesName: opts.name ?? '',
    botanicalName: '',
    hazardType: '',
    details: '',
    email: '',
    website: '',
    citations: [{ id: 1, type: 'doi', value: '', text: '' }],
    nextId: 2,
    phase: 'editing',
    errors: {},
    urlWarnings: {},
    banner: null,
    result: null,
  };
}

export type TextField = 'speciesName' | 'botanicalName' | 'hazardType' | 'details' | 'email' | 'website';

export type FormAction =
  | { type: 'set_kind'; kind: SubmissionKind }
  | { type: 'set_target'; target: TargetSpecies | null }
  | { type: 'set'; field: TextField; value: string }
  | { type: 'add_citation' }
  | { type: 'remove_citation'; id: number }
  | { type: 'update_citation'; id: number; patch: Partial<Omit<CitationRow, 'id'>> }
  | { type: 'check_citation'; id: number }
  | { type: 'url_warning'; id: number; warning: string | null }
  | { type: 'validate' }
  | { type: 'submit_start' }
  | { type: 'submit_ok'; result: SubmitResult }
  | { type: 'submit_invalid'; message: string; fields: Record<string, string> }
  | { type: 'submit_rate_limited'; retryAfter: number | null }
  | { type: 'submit_failed'; message: string }
  | { type: 'reset'; opts?: InitOptions };

const FIELD_KEY: Partial<Record<TextField, string>> = {
  speciesName: 'species_name',
  botanicalName: 'botanical_name',
  hazardType: 'hazard_type',
  details: 'details',
  email: 'email',
};

function without(errors: Record<string, string>, ...keys: string[]): Record<string, string> {
  const out = { ...errors };
  for (const k of keys) delete out[k];
  return out;
}

/** Does this kind need a listed target species? (reaction reports may name an unlisted one instead) */
export function needsTarget(kind: SubmissionKind | null): boolean {
  return kind === 'correction' || kind === 'new_source';
}

/** Hazard claims need at least one citation (PLAN §6); reaction reports don't. */
export function needsCitation(kind: SubmissionKind | null): boolean {
  return kind === 'new_species' || kind === 'correction' || kind === 'new_source';
}

export function reduce(state: FormState, action: FormAction): FormState {
  switch (action.type) {
    case 'set_kind':
      return { ...state, kind: action.kind, errors: without(state.errors, 'kind', 'target', 'species_name', 'citations') };
    case 'set_target':
      return { ...state, target: action.target, errors: without(state.errors, 'target', 'species_name') };
    case 'set': {
      const key = FIELD_KEY[action.field];
      return { ...state, [action.field]: action.value, errors: key ? without(state.errors, key) : state.errors };
    }
    case 'add_citation':
      if (state.citations.length >= MAX_CITATIONS) return state;
      return {
        ...state,
        citations: [...state.citations, { id: state.nextId, type: 'doi', value: '', text: '' }],
        nextId: state.nextId + 1,
        errors: without(state.errors, 'citations'),
      };
    case 'remove_citation': {
      const citations = state.citations.filter((c) => c.id !== action.id);
      const urlWarnings = { ...state.urlWarnings };
      delete urlWarnings[action.id];
      // Row indexes shift, so drop per-row errors and re-check on the next validate.
      const errors = Object.fromEntries(Object.entries(state.errors).filter(([k]) => !k.startsWith('citations.')));
      return {
        ...state,
        citations: citations.length ? citations : [{ id: state.nextId, type: 'doi', value: '', text: '' }],
        nextId: citations.length ? state.nextId : state.nextId + 1,
        urlWarnings,
        errors,
      };
    }
    case 'update_citation': {
      const i = state.citations.findIndex((c) => c.id === action.id);
      if (i < 0) return state;
      const citations = state.citations.map((c) => (c.id === action.id ? { ...c, ...action.patch } : c));
      const urlWarnings = { ...state.urlWarnings };
      if ('value' in action.patch || 'type' in action.patch) delete urlWarnings[action.id];
      return { ...state, citations, urlWarnings, errors: without(state.errors, `citations.${i}`, 'citations') };
    }
    case 'check_citation': {
      const i = state.citations.findIndex((c) => c.id === action.id);
      if (i < 0) return state;
      const c = state.citations[i];
      const err = citationError(c.type, c.value);
      const errors = without(state.errors, `citations.${i}`);
      if (err) errors[`citations.${i}`] = err;
      return { ...state, errors };
    }
    case 'url_warning': {
      const urlWarnings = { ...state.urlWarnings };
      if (action.warning) urlWarnings[action.id] = action.warning;
      else delete urlWarnings[action.id];
      return { ...state, urlWarnings };
    }
    case 'validate':
      return { ...state, errors: validate(state) };
    case 'submit_start':
      return { ...state, phase: 'submitting', banner: null };
    case 'submit_ok':
      return { ...state, phase: 'done', result: action.result, errors: {}, banner: null };
    case 'submit_invalid':
      return {
        ...state,
        phase: 'editing',
        errors: mapServerFields(action.fields, action.message),
        banner: null,
      };
    case 'submit_rate_limited':
      return { ...state, phase: 'editing', banner: { kind: 'rate_limited', retryAfter: action.retryAfter } };
    case 'submit_failed':
      return { ...state, phase: 'editing', banner: { kind: 'error', message: action.message } };
    case 'reset':
      return initialForm(action.opts);
  }
}

/** Client-side checks; the server repeats them. Returns field key -> message. */
export function validate(s: FormState): Record<string, string> {
  const e: Record<string, string> = {};
  if (!s.kind) e.kind = 'Choose what you’d like to send.';
  if (s.kind === 'new_species' && !s.speciesName.trim()) e.species_name = 'Enter the wood’s common name.';
  if (needsTarget(s.kind) && !s.target) e.target = 'Choose the species this is about.';
  if (s.kind === 'reaction_report' && !s.target && !s.speciesName.trim())
    e.target = 'Choose the species, or type its name if it isn’t listed.';
  const n = charCount(s.details.trim());
  if (n < DETAILS_MIN) e.details = `Please write at least ${DETAILS_MIN} characters (you have ${n}).`;
  else if (charCount(s.details) > DETAILS_MAX) e.details = `Please keep it under ${DETAILS_MAX} characters.`;
  if (s.email.trim() && !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(s.email.trim()))
    e.email = 'That email address doesn’t look right. It’s optional, so you can leave it empty.';
  s.citations.forEach((c, i) => {
    const err = citationError(c.type, c.value);
    if (err) e[`citations.${i}`] = err;
    else if (!c.value.trim() && c.text.trim()) e[`citations.${i}`] = 'Add a DOI, PubMed ID or web address for this source.';
  });
  if (needsCitation(s.kind) && !s.citations.some((c) => c.value.trim()))
    e.citations = 'Add at least one source: a DOI, PubMed ID or official web address.';
  return e;
}

/** Server field names ("citations[0].doi", "citations.0", "target_slug") -> form keys. */
export function mapServerFields(fields: Record<string, string> | undefined, message: string): Record<string, string> {
  const out: Record<string, string> = {};
  for (const [k, v] of Object.entries(fields ?? {})) {
    const m = k.match(/^citations?[[.](\d+)/);
    if (m) out[`citations.${m[1]}`] = v;
    else if (k === 'target_slug') out.target = v;
    else out[k] = v;
  }
  if (!Object.keys(out).length) out.form = message || 'Something in the form needs fixing.';
  return out;
}

export interface SubmissionPayload {
  kind: SubmissionKind;
  target_slug?: string;
  species_name?: string;
  botanical_name?: string;
  hazard_type?: string;
  details: string;
  citations: { doi?: string; pmid?: string; url?: string; text?: string }[];
  email?: string;
  form_token: string;
  website: string;
}

export function toPayload(s: FormState, formToken: string): SubmissionPayload {
  const p: SubmissionPayload = {
    kind: s.kind!,
    details: s.details.trim(),
    citations: s.citations
      .filter((c) => c.value.trim())
      .map((c) => {
        const out: SubmissionPayload['citations'][number] = {};
        if (c.type === 'doi') out.doi = normalizeDOI(c.value);
        else if (c.type === 'pmid') out.pmid = normalizePMID(c.value);
        else out.url = c.value.trim();
        if (c.text.trim()) out.text = c.text.trim();
        return out;
      }),
    form_token: formToken,
    website: s.website,
  };
  if (s.kind !== 'new_species' && s.target) p.target_slug = s.target.slug;
  if (s.kind === 'new_species' || (s.kind === 'reaction_report' && !s.target)) {
    if (s.speciesName.trim()) p.species_name = s.speciesName.trim();
  }
  if (s.kind === 'new_species' && s.botanicalName.trim()) p.botanical_name = s.botanicalName.trim();
  if (s.hazardType) p.hazard_type = s.hazardType;
  if (s.email.trim()) p.email = s.email.trim();
  return p;
}

/** Retry-After is either delta-seconds or an HTTP date. Returns seconds, or null if absent/unparseable. */
export function parseRetryAfter(value: string | null | undefined, now = Date.now()): number | null {
  if (!value) return null;
  const v = value.trim();
  if (/^\d+$/.test(v)) return Number(v);
  const t = Date.parse(v);
  if (Number.isNaN(t)) return null;
  return Math.max(0, Math.round((t - now) / 1000));
}

export function friendlyWait(seconds: number | null): string {
  if (seconds === null) return 'in a little while';
  if (seconds < 90) return 'in about a minute';
  const min = Math.round(seconds / 60);
  if (min < 90) return `in about ${min} minutes`;
  const h = Math.round(min / 60);
  if (h < 36) return `in about ${h} hour${h === 1 ? '' : 's'}`;
  return 'tomorrow';
}

/** Warnings may come as strings or { message } objects. */
export function warningTexts(w: unknown): string[] {
  if (!Array.isArray(w)) return [];
  return w
    .map((x) => (typeof x === 'string' ? x : x && typeof x === 'object' ? String((x as { message?: unknown }).message ?? '') : ''))
    .filter(Boolean);
}

// --- Status timeline (status page) ----------------------------------------

export type SubmissionStatus =
  | 'new'
  | 'triaging'
  | 'needs_info'
  | 'accepted'
  | 'rejected'
  | 'exported'
  | 'merged'
  | 'spam';

export const STATUS_TEXT: Record<SubmissionStatus, { label: string; explain: string }> = {
  new: { label: 'Received', explain: 'We have your submission. Nobody has looked at it yet.' },
  triaging: { label: 'Being reviewed', explain: 'We’re reading it and checking the sources.' },
  needs_info: {
    label: 'Waiting on a question',
    explain: 'We need a bit more information. See the notes below; if you left an email we may write to you.',
  },
  accepted: { label: 'Accepted', explain: 'It checks out and will be added to the dataset.' },
  rejected: { label: 'Not accepted', explain: 'We couldn’t use it. Any note below says why.' },
  exported: { label: 'Being added', explain: 'It’s been written up as a change to the dataset and is waiting for review.' },
  merged: { label: 'Published', explain: 'The change is live on the site. Thank you.' },
  spam: { label: 'Received', explain: 'We have your submission. Nobody has looked at it yet.' },
};

export function statusLabel(status: string): string {
  return (STATUS_TEXT as Record<string, { label: string }>)[status]?.label ?? status.replace(/_/g, ' ');
}

export interface TimelineStep {
  status: SubmissionStatus;
  label: string;
  explain: string;
  state: 'done' | 'current' | 'upcoming';
}

/**
 * The plain-language path from "Received" to "Published" for a status:
 * new → triaging → (needs_info) → accepted | rejected → exported → merged.
 * Reaction reports are never published, so they stop at the decision.
 */
export function timeline(status: string, kind?: string): TimelineStep[] {
  const st: SubmissionStatus = status === 'spam' || !(status in STATUS_TEXT) ? 'new' : (status as SubmissionStatus);
  let path: SubmissionStatus[] = ['new', 'triaging', 'accepted', 'exported', 'merged'];
  if (st === 'needs_info') path = ['new', 'triaging', 'needs_info', 'accepted', 'exported', 'merged'];
  if (st === 'rejected') path = ['new', 'triaging', 'rejected'];
  if (kind === 'reaction_report') path = path.filter((s) => s !== 'exported' && s !== 'merged');
  const at = path.indexOf(st);
  return path.map((s, i) => ({
    status: s,
    label: s === 'accepted' && at < i ? 'Decision' : STATUS_TEXT[s].label,
    explain: s === 'accepted' && at < i ? 'We’ll accept it or explain why not.' : STATUS_TEXT[s].explain,
    state: i < at ? 'done' : i === at ? (i === path.length - 1 ? 'done' : 'current') : 'upcoming',
  }));
}
