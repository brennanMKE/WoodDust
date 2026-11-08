// Client-side navigation state and the History-API router.
import { ApiError, getJSON } from './api';
import { applyHead, headFor } from './head';
import { isSpaLink, matchRoute, type RouteMatch } from './router';
import type { InitialPayload, SiteInfo } from './types';

interface AppState {
  match: RouteMatch;
  data: unknown;
  search: string;
  loading: boolean;
  error: string | null;
  site: SiteInfo;
  /** Increments on every completed client-side navigation (0 = first render). */
  navCount: number;
}

export const app: AppState = $state({
  match: matchRoute('/'),
  data: null,
  search: '',
  loading: false,
  error: null,
  site: { name: 'Dust Almanac', base_url: '' },
  navCount: 0,
});

let controller: AbortController | null = null;

export function readInitial(): InitialPayload | null {
  const el = document.getElementById('initial');
  if (!el?.textContent) return null;
  try {
    return JSON.parse(el.textContent) as InitialPayload;
  } catch {
    return null;
  }
}

export async function boot(initial: InitialPayload | null): Promise<void> {
  if (initial?.site) {
    app.site = {
      name: initial.site.name || app.site.name,
      base_url: initial.site.base_url || '',
    };
  }
  const match = matchRoute(location.pathname);
  app.search = location.search;

  if (initial && initial.route === 'not_found') {
    app.match = { ...match, id: 'not_found', api: null };
    app.data = {};
    return;
  }
  if (initial && initial.route === match.id) {
    // Never refetch the initial route's data (contract §1).
    app.match = match;
    app.data = initial.data;
    return;
  }
  // No server shell (Vite dev / fixtures): load like a client navigation,
  // but leave the head alone if Go already rendered it.
  await load(match, location.search, { initial: true });
}

async function load(match: RouteMatch, search: string, opts: { initial?: boolean } = {}): Promise<boolean> {
  controller?.abort();
  const ctl = new AbortController();
  controller = ctl;
  app.loading = true;
  app.error = null;
  let data: unknown = {};
  let finalMatch = match;
  try {
    if (match.api) data = await getJSON(match.api, ctl.signal);
  } catch (err) {
    if (ctl.signal.aborted) return false;
    if (err instanceof ApiError && err.status === 404) {
      if (match.id === 'species' && !opts.initial) {
        // Possibly an alias (e.g. /wood/wrc): let the server answer with its 301.
        location.assign(match.path);
        return false;
      }
      finalMatch = { ...match, id: 'not_found', api: null };
      data = {};
    } else {
      app.loading = false;
      app.error = err instanceof Error ? err.message : 'Something went wrong.';
      return false;
    }
  }
  if (ctl.signal.aborted) return false;
  app.match = finalMatch;
  app.data = data;
  app.search = search;
  app.loading = false;
  applyHead(headFor(finalMatch, data, app.site, search));
  return true;
}

export async function navigate(href: string, opts: { replace?: boolean } = {}): Promise<void> {
  const url = new URL(href, location.href);
  if (!isSpaLink(url, location.origin)) {
    location.assign(url.href);
    return;
  }
  const samePage = url.pathname === location.pathname && url.search === location.search;
  if (samePage && url.hash) {
    history.pushState(null, '', url.href);
    scrollToHash(url.hash);
    return;
  }
  // Remember where we were so Back restores it.
  history.replaceState({ ...(history.state ?? {}), scrollY: window.scrollY }, '');
  const match = matchRoute(url.pathname);
  if (opts.replace) history.replaceState({ scrollY: 0 }, '', url.href);
  else history.pushState({ scrollY: 0 }, '', url.href);
  const ok = await load(match, url.search);
  if (!ok) return;
  app.navCount++;
  requestAnimationFrame(() => {
    if (url.hash) scrollToHash(url.hash);
    else window.scrollTo(0, 0);
  });
}

/** Update only the query string of the current route (e.g. browse filters). */
export function replaceSearch(search: string): void {
  const url = new URL(location.href);
  url.search = search;
  history.replaceState(history.state, '', url.href);
  app.search = url.search;
}

function scrollToHash(hash: string) {
  const el = document.getElementById(decodeURIComponent(hash.slice(1)));
  if (!el) return;
  el.scrollIntoView();
  if (!el.hasAttribute('tabindex')) el.setAttribute('tabindex', '-1');
  el.focus({ preventScroll: true });
}

export function installRouter(): void {
  history.scrollRestoration = 'manual';

  document.addEventListener('click', (e) => {
    if (e.defaultPrevented || e.button !== 0 || e.metaKey || e.ctrlKey || e.shiftKey || e.altKey) return;
    const a = (e.target as Element | null)?.closest?.('a');
    if (!a || !a.hasAttribute('href')) return;
    if ((a.target && a.target !== '_self') || a.hasAttribute('download') || a.getAttribute('rel')?.includes('external')) return;
    const url = new URL(a.href, location.href);
    if (!isSpaLink(url, location.origin)) return;
    e.preventDefault();
    void navigate(url.href);
  });

  window.addEventListener('popstate', async (e) => {
    const y = (e.state as { scrollY?: number } | null)?.scrollY ?? 0;
    const match = matchRoute(location.pathname);
    if (match.path === app.match.path && location.search === app.search) {
      if (location.hash) scrollToHash(location.hash);
      return;
    }
    const ok = await load(match, location.search);
    if (ok) {
      app.navCount++;
      requestAnimationFrame(() => window.scrollTo(0, y));
    }
  });
}
