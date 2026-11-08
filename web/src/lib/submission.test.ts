import { describe, expect, it } from 'vitest';
import {
  friendlyWait,
  initialForm,
  mapServerFields,
  parseRetryAfter,
  reduce,
  timeline,
  toPayload,
  validate,
  warningTexts,
  type FormAction,
  type FormState,
} from './submission';

const run = (s: FormState, ...actions: FormAction[]) => actions.reduce(reduce, s);
const details = 'Reported asthma in two sawmill workers, confirmed by challenge test.';

describe('initialForm', () => {
  it('preselects from ?kind=, a target or ?name=', () => {
    expect(initialForm({ kind: 'new_source' }).kind).toBe('new_source');
    expect(initialForm({ kind: 'bogus' }).kind).toBeNull();
    expect(initialForm({ target: { slug: 'oak', name: 'Oak' } }).kind).toBe('correction');
    const s = initialForm({ name: 'Bubinga' });
    expect(s.kind).toBe('new_species');
    expect(s.speciesName).toBe('Bubinga');
    expect(s.citations).toHaveLength(1);
  });
});

describe('reduce', () => {
  it('adds, edits and removes citation rows, keeping at least one', () => {
    let s = run(initialForm(), { type: 'add_citation' }, { type: 'add_citation' });
    expect(s.citations.map((c) => c.id)).toEqual([1, 2, 3]);
    s = reduce(s, { type: 'update_citation', id: 2, patch: { type: 'pmid', value: '2021399' } });
    expect(s.citations[1]).toMatchObject({ type: 'pmid', value: '2021399' });
    s = run(s, { type: 'remove_citation', id: 1 }, { type: 'remove_citation', id: 2 }, { type: 'remove_citation', id: 3 });
    expect(s.citations).toHaveLength(1);
    expect(s.citations[0].value).toBe('');
  });

  it('caps citation rows', () => {
    let s = initialForm();
    for (let i = 0; i < 20; i++) s = reduce(s, { type: 'add_citation' });
    expect(s.citations.length).toBe(8);
  });

  it('checks a row on blur and clears the error on edit', () => {
    let s = run(initialForm(), { type: 'update_citation', id: 1, patch: { value: '10.1/x' } }, { type: 'check_citation', id: 1 });
    expect(s.errors['citations.0']).toMatch(/DOI/);
    s = reduce(s, { type: 'update_citation', id: 1, patch: { value: '10.1136/x' } });
    expect(s.errors['citations.0']).toBeUndefined();
  });

  it('drops a URL warning when the value changes', () => {
    let s = reduce(initialForm(), { type: 'url_warning', id: 1, warning: 'leads only' });
    expect(s.urlWarnings[1]).toBe('leads only');
    s = reduce(s, { type: 'update_citation', id: 1, patch: { value: 'https://osha.gov' } });
    expect(s.urlWarnings[1]).toBeUndefined();
  });

  it('moves through submit phases', () => {
    let s = reduce(initialForm({ kind: 'reaction_report' }), { type: 'submit_start' });
    expect(s.phase).toBe('submitting');
    s = reduce(s, { type: 'submit_rate_limited', retryAfter: 120 });
    expect(s.phase).toBe('editing');
    expect(s.banner).toEqual({ kind: 'rate_limited', retryAfter: 120 });
    s = run(s, { type: 'submit_start' }, { type: 'submit_invalid', message: 'bad', fields: { 'citations[0].doi': 'nope', target_slug: 'pick one' } });
    expect(s.errors).toEqual({ 'citations.0': 'nope', target: 'pick one' });
    expect(s.banner).toBeNull();
    s = run(s, { type: 'submit_start' }, { type: 'submit_ok', result: { reference: 'DA-0042', statusUrl: '/submit/status/t', warnings: [] } });
    expect(s.phase).toBe('done');
    expect(s.result?.reference).toBe('DA-0042');
    expect(s.errors).toEqual({});
  });

  it('typing in a field clears its error', () => {
    let s = reduce(initialForm({ kind: 'new_species' }), { type: 'validate' });
    expect(s.errors.details).toBeDefined();
    s = reduce(s, { type: 'set', field: 'details', value: 'x' });
    expect(s.errors.details).toBeUndefined();
  });
});

describe('validate', () => {
  it('requires a kind, details and, for hazard claims, a source', () => {
    expect(Object.keys(validate(initialForm())).sort()).toEqual(['details', 'kind']);
    const e = validate(run(initialForm({ kind: 'new_species' }), { type: 'set', field: 'details', value: details }));
    expect(e).toHaveProperty('species_name');
    expect(e).toHaveProperty('citations');
  });

  it('requires a target for corrections and new sources', () => {
    const s = run(initialForm({ kind: 'correction' }), { type: 'set', field: 'details', value: details });
    expect(validate(s).target).toBeDefined();
    expect(validate(reduce(s, { type: 'set_target', target: { slug: 'oak', name: 'Oak' } })).target).toBeUndefined();
  });

  it('lets reaction reports skip sources and name an unlisted wood', () => {
    const s = run(
      initialForm({ kind: 'reaction_report' }),
      { type: 'set', field: 'details', value: details },
      { type: 'set', field: 'speciesName', value: 'Bubinga' },
    );
    expect(validate(s)).toEqual({});
  });

  it('checks details length, email and citation formats', () => {
    const s = run(
      initialForm({ kind: 'new_source', target: { slug: 'oak', name: 'Oak' } }),
      { type: 'set', field: 'details', value: 'x'.repeat(4001) },
      { type: 'set', field: 'email', value: 'nope' },
      { type: 'update_citation', id: 1, patch: { type: 'pmid', value: 'PMC1' } },
    );
    const e = validate(s);
    expect(e.details).toMatch(/4000/);
    expect(e.email).toBeDefined();
    expect(e['citations.0']).toMatch(/digits/);
  });
});

describe('toPayload', () => {
  it('builds the contract body with normalized citations', () => {
    const s = run(
      initialForm({ kind: 'new_species', name: ' Bubinga ' }),
      { type: 'set', field: 'botanicalName', value: 'Guibourtia spp.' },
      { type: 'set', field: 'hazardType', value: 'skin_sensitizer' },
      { type: 'set', field: 'details', value: `  ${details}  ` },
      { type: 'update_citation', id: 1, patch: { value: 'https://doi.org/10.1136/x', text: ' Smith 1990 ' } },
      { type: 'add_citation' },
      { type: 'update_citation', id: 2, patch: { type: 'pmid', value: 'PMID: 42' } },
      { type: 'add_citation' }, // empty rows are dropped
    );
    expect(toPayload(s, 'tok')).toEqual({
      kind: 'new_species',
      species_name: 'Bubinga',
      botanical_name: 'Guibourtia spp.',
      hazard_type: 'skin_sensitizer',
      details,
      citations: [{ doi: '10.1136/x', text: 'Smith 1990' }, { pmid: '42' }],
      form_token: 'tok',
      website: '',
    });
  });

  it('sends target_slug for corrections and passes the honeypot through', () => {
    const s = run(
      initialForm({ kind: 'correction', target: { slug: 'oak', name: 'Oak' } }),
      { type: 'set', field: 'website', value: 'http://spam' },
      { type: 'set', field: 'email', value: ' a@b.co ' },
    );
    const p = toPayload(s, 't');
    expect(p.target_slug).toBe('oak');
    expect(p.species_name).toBeUndefined();
    expect(p.website).toBe('http://spam');
    expect(p.email).toBe('a@b.co');
  });
});

describe('server responses', () => {
  it('maps 422 field names and falls back to a form-level message', () => {
    expect(mapServerFields({ 'citations.2': 'x', details: 'y' }, '')).toEqual({ 'citations.2': 'x', details: 'y' });
    expect(mapServerFields({}, 'Try again')).toEqual({ form: 'Try again' });
  });
  it('parses Retry-After as seconds or a date', () => {
    expect(parseRetryAfter('120')).toBe(120);
    expect(parseRetryAfter(new Date(10_000 + 300_000).toUTCString(), 10_000)).toBe(300);
    expect(parseRetryAfter(null)).toBeNull();
    expect(parseRetryAfter('soon')).toBeNull();
  });
  it('words the wait plainly', () => {
    expect(friendlyWait(30)).toBe('in about a minute');
    expect(friendlyWait(600)).toBe('in about 10 minutes');
    expect(friendlyWait(7200)).toBe('in about 2 hours');
    expect(friendlyWait(null)).toBe('in a little while');
  });
  it('reads warnings as strings or objects', () => {
    expect(warningTexts(['a', { message: 'b' }, 3])).toEqual(['a', 'b']);
    expect(warningTexts(undefined)).toEqual([]);
  });
});

describe('timeline', () => {
  it('marks done, current and upcoming steps', () => {
    const t = timeline('triaging', 'correction');
    expect(t.map((s) => [s.status, s.state])).toEqual([
      ['new', 'done'],
      ['triaging', 'current'],
      ['accepted', 'upcoming'],
      ['exported', 'upcoming'],
      ['merged', 'upcoming'],
    ]);
    expect(t[2].label).toBe('Decision');
  });
  it('inserts needs_info, ends at rejected, and hides spam', () => {
    expect(timeline('needs_info').map((s) => s.status)).toContain('needs_info');
    expect(timeline('rejected').map((s) => [s.status, s.state]).at(-1)).toEqual(['rejected', 'done']);
    expect(timeline('spam')[0]).toMatchObject({ status: 'new', state: 'current', label: 'Received' });
  });
  it('never shows publishing steps for reaction reports', () => {
    expect(timeline('accepted', 'reaction_report').map((s) => s.status)).toEqual(['new', 'triaging', 'accepted']);
  });
});
