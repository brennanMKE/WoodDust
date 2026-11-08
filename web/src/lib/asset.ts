import { app } from './nav.svelte';

/**
 * Versioned URL for an image the SPA references by fixed path, e.g.
 * asset('/images/art/emblem.png') -> '/images/art/emblem.png?v=1a2b3c4d5e'.
 * The server sends content hashes in #initial (site.assets), so a changed or
 * newly added file gets a new URL and no stale cached copy (or cached 404) is
 * reused. Unknown files are returned unchanged.
 */
export function asset(path: string): string {
  const rel = path.replace(/^\/images\//, '');
  const v = app.site.assets?.[rel];
  return v ? `${path}?v=${v}` : path;
}
