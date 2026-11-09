import type { RouteMatch } from './router';
import type { ContentPage, PosterData, SiteInfo, SpeciesDetail, SubmitData } from './types';
import { hazardName } from './labels';

export interface HeadInfo {
  title: string;
  description: string;
  canonical: string;
  noindex: boolean;
}

const DEFAULT_DESC =
  'Wood dust hazards by species: asthma, allergies, irritation and cancer, with evidence levels, exposure limits and sources. All wood dust is hazardous.';

export function headFor(match: RouteMatch, data: unknown, site: SiteInfo, search: string): HeadInfo {
  const base = (site.base_url || (typeof location !== 'undefined' ? location.origin : '')).replace(/\/$/, '');
  const canonical = base + match.path;
  const params = new URLSearchParams(search);
  const t = (s: string) => `${s} | ${site.name}`;

  switch (match.id) {
    case 'home':
      return {
        title: `${site.name}: wood dust hazards by species`,
        description: DEFAULT_DESC,
        canonical: base + '/',
        noindex: params.has('q'),
      };
    case 'species': {
      const s = data as SpeciesDetail | null;
      if (!s?.name) break;
      const types = [...new Set(s.hazards.filter((h) => h.evidence_level !== 0).map((h) => hazardName(h.type)))];
      const list = types.length ? types.slice(0, 4).join(', ').toLowerCase() : 'the hazards shared by all wood dust';
      return {
        title: t(`${s.name} wood dust hazards`),
        description: `${s.name} (${s.botanical_name}) dust: ${list}. Evidence levels, exposure limits and sources.`,
        canonical: base + `/wood/${s.slug}`,
        noindex: false,
      };
    }
    case 'browse':
      return {
        title: t('Browse wood species'),
        description: 'Filter wood species by hazard type, evidence level and hardwood or softwood.',
        canonical,
        noindex: false,
      };
    case 'page': {
      const p = data as ContentPage | null;
      if (!p?.title) break;
      return { title: t(p.title), description: p.description || DEFAULT_DESC, canonical, noindex: false };
    }
    case 'sources':
      return {
        title: t('Sources'),
        description: 'Every source cited on this site: agency listings, reviews and primary studies.',
        canonical,
        noindex: false,
      };
    case 'submit': {
      const target = (data as SubmitData | null)?.target;
      const kind = params.get('kind');
      const title = target
        ? kind === 'correction'
          ? `Report a correction: ${target.name}`
          : `Send a note about ${target.name}`
        : kind === 'new_species' || params.has('name')
          ? 'Suggest a species'
          : 'Suggest or correct';
      return {
        title: t(title),
        description: 'Suggest a species, correct a card, add a better source or report a reaction. Anonymous; every submission is reviewed.',
        canonical,
        noindex: true,
      };
    }
    case 'submission_status':
      return { title: t('Submission status'), description: DEFAULT_DESC, canonical, noindex: true };
    case 'admin':
      return { title: t('Admin'), description: DEFAULT_DESC, canonical, noindex: true };
    case 'posters':
      return {
        title: t('Shop posters'),
        description:
          'Free printable posters for wood shops: Paul Bunyan and Babe on dust and respirators, with a QR code to the facts. Letter and A4 PDFs.',
        canonical: base + '/posters',
        noindex: false,
      };
    case 'poster_print': {
      const p = (data as PosterData | null)?.poster;
      return {
        title: t(p ? `Print: ${p.title} poster` : 'Print a poster'),
        description: DEFAULT_DESC,
        canonical: base + '/posters',
        noindex: true,
      };
    }
  }
  return { title: t('Page not found'), description: DEFAULT_DESC, canonical, noindex: true };
}

function setMeta(selector: string, create: () => HTMLElement, attr: string, value: string | null) {
  let el = document.head.querySelector<HTMLElement>(selector);
  if (value === null) {
    el?.remove();
    return;
  }
  if (!el) {
    el = create();
    document.head.appendChild(el);
  }
  el.setAttribute(attr, value);
}

export function applyHead(h: HeadInfo): void {
  document.title = h.title;
  setMeta(
    'meta[name="description"]',
    () => Object.assign(document.createElement('meta'), { name: 'description' }),
    'content',
    h.description,
  );
  setMeta(
    'link[rel="canonical"]',
    () => Object.assign(document.createElement('link'), { rel: 'canonical' }),
    'href',
    h.noindex ? null : h.canonical,
  );
  setMeta(
    'meta[name="robots"]',
    () => Object.assign(document.createElement('meta'), { name: 'robots' }),
    'content',
    h.noindex ? 'noindex' : null,
  );
  setMeta('meta[property="og:title"]', () => og('og:title'), 'content', h.title);
  setMeta('meta[property="og:description"]', () => og('og:description'), 'content', h.description);
  setMeta('meta[property="og:url"]', () => og('og:url'), 'content', h.canonical);
}

function og(property: string): HTMLElement {
  const m = document.createElement('meta');
  m.setAttribute('property', property);
  return m;
}
