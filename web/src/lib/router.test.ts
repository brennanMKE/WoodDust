import { describe, expect, it } from 'vitest';
import { isSpaLink, matchRoute, normalizePath } from './router';

describe('normalizePath', () => {
  it('strips trailing slashes, queries and hashes', () => {
    expect(normalizePath('/browse/')).toBe('/browse');
    expect(normalizePath('/wood/oak?x=1#src-A')).toBe('/wood/oak');
    expect(normalizePath('')).toBe('/');
    expect(normalizePath('/')).toBe('/');
  });
});

describe('matchRoute', () => {
  it('matches home with the featured species API', () => {
    expect(matchRoute('/')).toMatchObject({ id: 'home', api: '/api/v1/species?featured=true' });
    expect(matchRoute('/?q=oak').id).toBe('home');
  });

  it('matches species cards', () => {
    const m = matchRoute('/wood/western-red-cedar');
    expect(m.id).toBe('species');
    expect(m.params.slug).toBe('western-red-cedar');
    expect(m.api).toBe('/api/v1/species/western-red-cedar');
    expect(matchRoute('/wood').id).toBe('not_found');
    expect(matchRoute('/wood/oak/extra').id).toBe('not_found');
  });

  it('matches browse and sources', () => {
    expect(matchRoute('/browse')).toMatchObject({ id: 'browse', api: '/api/v1/species' });
    expect(matchRoute('/sources/')).toMatchObject({ id: 'sources', api: '/api/v1/sources' });
    expect(matchRoute('/browse/x').id).toBe('not_found');
  });

  it('matches content pages, including protection subpages', () => {
    expect(matchRoute('/risks')).toMatchObject({ id: 'page', api: '/api/v1/pages/risks' });
    expect(matchRoute('/methodology').api).toBe('/api/v1/pages/methodology');
    expect(matchRoute('/protection')).toMatchObject({ id: 'page', api: '/api/v1/pages/protection' });
    expect(matchRoute('/protection/respirators')).toMatchObject({
      id: 'page',
      params: { page: 'protection/respirators' },
      api: '/api/v1/pages/protection/respirators',
    });
    expect(matchRoute('/risks/more').id).toBe('not_found');
  });

  it('matches the printable shop poster as a page', () => {
    expect(matchRoute('/print/shop-poster')).toMatchObject({
      id: 'page',
      params: { page: 'print/shop-poster' },
      api: '/api/v1/pages/print/shop-poster',
    });
    expect(matchRoute('/print').id).toBe('not_found');
  });

  it('matches submit routes (route id "submit") without an API call', () => {
    expect(matchRoute('/submit')).toMatchObject({ id: 'submit', api: null });
    expect(matchRoute('/submit/oak').params.slug).toBe('oak');
  });

  it('falls through to not_found', () => {
    expect(matchRoute('/nope').id).toBe('not_found');
    expect(matchRoute('/wood/..').id).toBe('not_found');
  });
});

describe('isSpaLink', () => {
  const origin = 'https://example.test';
  const u = (p: string) => new URL(p, origin);
  it('handles known routes on the same origin', () => {
    expect(isSpaLink(u('/wood/oak'), origin)).toBe(true);
    expect(isSpaLink(u('/protection/respirators'), origin)).toBe(true);
    expect(isSpaLink(u('/print/shop-poster'), origin)).toBe(true);
    expect(isSpaLink(u('/submit/oak'), origin)).toBe(true);
  });
  it('leaves other links to the browser', () => {
    expect(isSpaLink(u('/sitemap.xml'), origin)).toBe(false);
    expect(isSpaLink(u('/api/v1/species'), origin)).toBe(false);
    expect(isSpaLink(u('/login'), origin)).toBe(false);
    expect(isSpaLink(u('/unknown'), origin)).toBe(false);
    expect(isSpaLink(new URL('https://other.test/wood/oak'), origin)).toBe(false);
  });
});
