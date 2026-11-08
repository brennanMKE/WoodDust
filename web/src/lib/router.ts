// Route table from docs/contract.md §2, plus client-only routes (`submit`).
// Pure functions only, so they can be unit tested without a DOM.

export type RouteId = 'home' | 'species' | 'browse' | 'page' | 'sources' | 'submit' | 'not_found';

export interface RouteMatch {
  id: RouteId;
  path: string;
  params: Record<string, string>;
  /** API path whose response is this route's `data`, or null if the route needs none. */
  api: string | null;
}

export const PAGE_PATHS = ['risks', 'protection', 'symptoms', 'limits', 'glossary', 'methodology'] as const;

const SEGMENT = /^[A-Za-z0-9][A-Za-z0-9._~%-]*$/;

/** Strip query/hash and a trailing slash ("/browse/" -> "/browse"). */
export function normalizePath(path: string): string {
  let p = path.split(/[?#]/)[0] || '/';
  if (!p.startsWith('/')) p = '/' + p;
  p = p.replace(/\/{2,}/g, '/');
  if (p.length > 1 && p.endsWith('/')) p = p.slice(0, -1);
  return p;
}

export function matchRoute(rawPath: string): RouteMatch {
  const path = normalizePath(rawPath);
  const parts = path === '/' ? [] : path.slice(1).split('/');
  const notFound: RouteMatch = { id: 'not_found', path, params: {}, api: null };

  if (parts.length === 0) {
    return { id: 'home', path, params: {}, api: '/api/v1/species?featured=true' };
  }
  if (parts.some((s) => !SEGMENT.test(s))) return notFound;

  const [first, second, ...rest] = parts;
  if (rest.length > 0) return notFound;

  switch (first) {
    case 'wood':
      if (!second) return notFound;
      return {
        id: 'species',
        path,
        params: { slug: second },
        api: `/api/v1/species/${encodeURIComponent(second)}`,
      };
    case 'browse':
      return second ? notFound : { id: 'browse', path, params: {}, api: '/api/v1/species' };
    case 'sources':
      return second ? notFound : { id: 'sources', path, params: {}, api: '/api/v1/sources' };
    case 'submit':
      return { id: 'submit', path, params: second ? { slug: second } : {}, api: null };
    case 'print':
      // Printable pages (contract §2: /print/shop-poster), served by the page API.
      if (!second) return notFound;
      return {
        id: 'page',
        path,
        params: { page: `print/${second}` },
        api: `/api/v1/pages/print/${encodeURIComponent(second)}`,
      };
    case 'protection':
      return {
        id: 'page',
        path,
        params: { page: second ? `protection/${second}` : 'protection' },
        api: `/api/v1/pages/${second ? `protection/${encodeURIComponent(second)}` : 'protection'}`,
      };
    default:
      if (!second && (PAGE_PATHS as readonly string[]).includes(first)) {
        return { id: 'page', path, params: { page: first }, api: `/api/v1/pages/${first}` };
      }
      return notFound;
  }
}

/**
 * Should a click on this URL be handled by the SPA? Only same-origin links to
 * routes the SPA knows; everything else (API, images, sitemap, server-only
 * pages such as sign-in) is a normal browser navigation.
 */
export function isSpaLink(url: URL, origin: string): boolean {
  if (url.origin !== origin) return false;
  if (/^\/(api|images|assets|auth|login|signin|admin|account|oauth|accounts|u)(\/|$)/.test(url.pathname)) return false;
  if (/\.[a-z0-9]+$/i.test(url.pathname)) return false; // files like /sitemap.xml
  return matchRoute(url.pathname).id !== 'not_found';
}
