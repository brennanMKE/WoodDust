import { describe, expect, it } from 'vitest';
import {
  citationError,
  doiHref,
  guessCitationType,
  isDOI,
  isHttpURL,
  isPMID,
  leadsOnlyWarning,
  normalizeDOI,
  normalizePMID,
  pmidHref,
} from './citations';

describe('DOI', () => {
  it('accepts the contract pattern ^10.\\d{4,9}/\\S+$', () => {
    expect(isDOI('10.1136/oem.48.3.201')).toBe(true);
    expect(isDOI('10.1111/j.1600-0536.1990.tb01568.x')).toBe(true);
    expect(isDOI('10.123456789/x')).toBe(true);
  });
  it('accepts doi: and doi.org prefixes, normalizing them away', () => {
    expect(isDOI('doi:10.1136/oem.48.3.201')).toBe(true);
    expect(isDOI(' https://doi.org/10.1136/oem.48.3.201 ')).toBe(true);
    expect(normalizeDOI('https://dx.doi.org/10.1136/x')).toBe('10.1136/x');
  });
  it('rejects malformed DOIs', () => {
    expect(isDOI('10.123/x')).toBe(false); // registrant too short
    expect(isDOI('10.1136')).toBe(false); // no suffix
    expect(isDOI('10.1136/has space')).toBe(false);
    expect(isDOI('11.1136/oem')).toBe(false);
    expect(isDOI('')).toBe(false);
  });
});

describe('PMID', () => {
  it('accepts digits, with an optional PMID: prefix or PubMed URL', () => {
    expect(isPMID('2021399')).toBe(true);
    expect(isPMID('PMID: 2021399')).toBe(true);
    expect(isPMID('https://pubmed.ncbi.nlm.nih.gov/2021399/')).toBe(true);
    expect(normalizePMID('pmid 2021399')).toBe('2021399');
  });
  it('rejects anything else', () => {
    expect(isPMID('PMC123456')).toBe(false);
    expect(isPMID('12a4')).toBe(false);
    expect(isPMID('0')).toBe(false);
    expect(isPMID('1234567890')).toBe(false);
    expect(isPMID('')).toBe(false);
  });
});

describe('URLs and leads-only domains', () => {
  it('needs http(s)', () => {
    expect(isHttpURL('https://www.osha.gov/wood-dust')).toBe(true);
    expect(isHttpURL('ftp://example.com/x')).toBe(false);
    expect(isHttpURL('www.osha.gov')).toBe(false);
    expect(isHttpURL('javascript:alert(1)')).toBe(false);
  });
  it('warns for compilations and forums, not agencies', () => {
    expect(leadsOnlyWarning('https://www.wood-database.com/wood-articles/wood-allergies-and-toxicity/')).toMatch(/leads/);
    expect(leadsOnlyWarning('https://haz-map.com/Agents/1')).not.toBeNull();
    expect(leadsOnlyWarning('https://forums.example.org/t/1')).not.toBeNull();
    expect(leadsOnlyWarning('https://www.osha.gov/wood-dust')).toBeNull();
    expect(leadsOnlyWarning('not a url')).toBeNull();
  });
});

describe('citationError and helpers', () => {
  it('gives one message per type, and none for empty values', () => {
    expect(citationError('doi', '')).toBeNull();
    expect(citationError('doi', '10.1/x')).toMatch(/DOI/);
    expect(citationError('pmid', 'abc')).toMatch(/digits/);
    expect(citationError('url', 'osha.gov')).toMatch(/https/);
    expect(citationError('url', 'https://osha.gov')).toBeNull();
  });
  it('guesses the type of pasted values', () => {
    expect(guessCitationType('10.1136/x')).toBe('doi');
    expect(guessCitationType('https://doi.org/10.1136/x')).toBe('doi');
    expect(guessCitationType('2021399')).toBe('pmid');
    expect(guessCitationType('https://osha.gov')).toBe('url');
    expect(guessCitationType('Smith 1990')).toBeNull();
  });
  it('builds doi.org and PubMed links', () => {
    expect(doiHref('doi:10.1136/x')).toBe('https://doi.org/10.1136/x');
    expect(pmidHref('PMID: 42')).toBe('https://pubmed.ncbi.nlm.nih.gov/42/');
  });
});
