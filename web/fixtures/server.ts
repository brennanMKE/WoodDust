// Dev-only fixture API (VITE_FIXTURES=1 npm run dev). Answers /api/v1/* and
// /images/* from the repo's real data files so the SPA can be developed
// without the Go server. Loaded only by vite.config.ts during `vite serve`;
// never imported by src/, so it is not in the production bundle.
//
// It approximates docs/contract.md; the Go server is the reference. Content
// pages are rendered with a deliberately tiny Markdown subset.
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import type { IncomingMessage, ServerResponse } from 'node:http';
import type { Plugin } from 'vite';

const here = path.dirname(fileURLToPath(import.meta.url));
const repo = path.resolve(here, '..', '..');
const dataDir = path.join(repo, 'data');
const pagesDir = path.join(repo, 'content', 'pages');
const imagesDir = path.join(repo, 'images');

type Json = Record<string, any>;

function readJSON(file: string): any {
  return JSON.parse(fs.readFileSync(file, 'utf8'));
}

function loadSpecies(): Json[] {
  const dir = path.join(dataDir, 'species');
  return fs
    .readdirSync(dir)
    .filter((f) => f.endsWith('.json'))
    .map((f) => readJSON(path.join(dir, f)));
}

function loadSources(): Record<string, Json> {
  const all = readJSON(path.join(dataDir, 'sources.json')) as Record<string, Json>;
  const out: Record<string, Json> = {};
  for (const [k, v] of Object.entries(all)) if (!k.startsWith('_')) out[k] = { ...v, key: k };
  return out;
}

function regulatory(): Json {
  try {
    const r = readJSON(path.join(dataDir, 'regulatory.json'));
    return {
      eu_hardwood_mg_m3: r.eu_binding_limit?.value_mg_m3,
      uk_hardwood_mg_m3: r.uk_eh40?.hardwood_dust?.value_mg_m3,
      uk_softwood_mg_m3: r.uk_eh40?.softwood_dust?.value_mg_m3,
      niosh_rel_mg_m3: r.niosh_rel?.value_mg_m3,
    };
  } catch {
    return {};
  }
}

// Display names: capitalize words except small joining words (unless first),
// e.g. "cedar of lebanon" -> "Cedar of Lebanon". Matches the Go server.
const SMALL_WORDS = new Set(['of', 'the', 'and', 'de', 'du', 'la']);
const titleCase = (s: string) =>
  s.replace(/(^|[\s-])(\p{L}[\p{L}']*)/gu, (m, sep: string, word: string) =>
    m.length !== word.length + sep.length || (sep !== '' && SMALL_WORDS.has(word.toLowerCase()))
      ? m
      : sep + word[0].toUpperCase() + word.slice(1),
  );
const norm = (s: string) =>
  s
    .normalize('NFD')
    .replace(/[̀-ͯ]/g, '')
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, ' ')
    .trim();

function image(s: Json): Json {
  const file = path.join(imagesDir, 'species', `${s.slug}.jpg`);
  const url = fs.existsSync(file) ? `/images/species/${s.slug}.jpg` : '/images/placeholder.svg';
  return { url, alt: s.image_alt ?? '', width: 1200, height: 630 };
}

function summary(s: Json): Json {
  const specific = s.hazards.map((h: Json) => h.evidence_level).filter((l: number) => l > 0);
  return {
    slug: s.slug,
    name: titleCase(s.common_names[0]),
    common_names: s.common_names,
    botanical_name: s.botanical_name,
    wood_type: s.wood_type,
    region: s.region,
    featured_rank: s.featured_rank ?? null,
    hazard_types: [...new Set(s.hazards.map((h: Json) => h.type))],
    max_evidence_level: specific.length ? Math.min(...specific) : null,
    image: image(s),
    url: `/wood/${s.slug}`,
  };
}

function detail(s: Json): Json {
  const { notes_internal: _n, image_prompt: _p, images: _i, ...rest } = s;
  const all = loadSources();
  const keys = new Set<string>();
  for (const h of s.hazards) for (const k of h.sources) keys.add(k);
  for (const x of [...(s.exposure_limits ?? []), ...(s.carcinogen_class ?? []), ...(s.known_compounds ?? [])])
    keys.add(x.source);
  const sources: Record<string, Json> = {};
  for (const k of keys) if (all[k]) sources[k] = all[k];
  return { ...rest, name: titleCase(s.common_names[0]), url: `/wood/${s.slug}`, image: image(s), sources, regulatory: regulatory() };
}

function trigrams(s: string): Set<string> {
  const t = `  ${s} `;
  const out = new Set<string>();
  for (let i = 0; i < t.length - 2; i++) out.add(t.slice(i, i + 3));
  return out;
}

function similarity(a: string, b: string): number {
  const A = trigrams(a);
  const B = trigrams(b);
  let n = 0;
  for (const x of A) if (B.has(x)) n++;
  return n / (A.size + B.size - n || 1);
}

function search(q: string, limit: number): Json[] {
  const nq = norm(q);
  if (!nq) return [];
  const best = new Map<string, Json>();
  for (const s of loadSpecies()) {
    const names: [string, string][] = [
      ...s.common_names.map((n: string, i: number) => [n, i === 0 ? 'common' : 'trade'] as [string, string]),
      [s.botanical_name, 'botanical'],
      ...(s.synonyms ?? []).map((n: string) => [n, 'synonym'] as [string, string]),
    ];
    for (const [name, kind] of names) {
      const nn = norm(name);
      let score = 0;
      if (nn === nq) score = 1;
      else if (nn.startsWith(nq)) score = 0.9 + 0.05 * (nq.length / nn.length);
      else if (nn.split(' ').some((w) => w.startsWith(nq))) score = 0.8;
      else score = similarity(nq, nn) * 0.75;
      if (score < 0.3) continue;
      const prev = best.get(s.slug);
      if (!prev || score > prev.score)
        best.set(s.slug, {
          slug: s.slug,
          name: titleCase(s.common_names[0]),
          botanical_name: s.botanical_name,
          matched_name: name,
          match_kind: kind,
          score: Math.round(score * 100) / 100,
          url: `/wood/${s.slug}`,
        });
    }
  }
  return [...best.values()].sort((a, b) => b.score - a.score || a.name.localeCompare(b.name)).slice(0, limit);
}

// --- Content pages: front matter + a tiny Markdown subset -------------------

function slugify(s: string): string {
  return norm(s).replace(/ /g, '-');
}

function inline(md: string, foot: Map<string, number>): string {
  let s = md.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
  s = s.replace(/\[\^([A-Za-z0-9-]+)\]/g, (_m, k) => {
    if (!foot.has(k)) foot.set(k, foot.size + 1);
    return `<sup><a href="#src-${k}">${foot.get(k)}</a></sup>`;
  });
  s = s.replace(/\[([^\]]+)\]\(([^)\s]+)\)/g, '<a href="$2">$1</a>');
  s = s.replace(/\*\*([^*]+)\*\*/g, '<strong>$1</strong>');
  s = s.replace(/(^|[^*])\*([^*]+)\*/g, '$1<em>$2</em>');
  s = s.replace(/`([^`]+)`/g, '<code>$1</code>');
  return s;
}

function renderPage(pagePath: string): Json | null {
  const file = path.join(pagesDir, `${pagePath.replace(/\//g, '-')}.md`);
  const alt = path.join(pagesDir, `${pagePath}.md`);
  // Files may name their URL in front matter (e.g. shop-poster.md has slug: print/shop-poster).
  const bySlug = fs.existsSync(pagesDir)
    ? fs
        .readdirSync(pagesDir)
        .filter((f) => f.endsWith('.md'))
        .map((f) => path.join(pagesDir, f))
        .find((f) => new RegExp(`^slug:\\s*["']?${pagePath.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')}["']?\\s*$`, 'm').test(fs.readFileSync(f, 'utf8').split('\n---')[0]))
    : undefined;
  const src = bySlug ?? (fs.existsSync(file) ? file : fs.existsSync(alt) ? alt : null);
  const known = ['risks', 'protection', 'symptoms', 'limits', 'glossary', 'methodology'];
  if (!src) {
    if (!known.includes(pagePath.split('/')[0])) return null;
    const title = titleCase(pagePath.split('/').pop()!.replace(/-/g, ' '));
    return {
      slug: pagePath,
      title,
      description: `Fixture placeholder for /${pagePath}.`,
      last_reviewed: '',
      html: `<h2 id="draft">Draft not written yet</h2><p>No <code>content/pages</code> file exists for <code>${pagePath}</code>. This placeholder comes from the dev fixtures.</p><h2 id="example">Example section</h2><p>Footnotes render like this.<sup><a href="#src-IARC-62">1</a></sup></p>`,
      toc: [
        { id: 'draft', text: 'Draft not written yet', level: 2 },
        { id: 'example', text: 'Example section', level: 2 },
      ],
      sources: Object.fromEntries(Object.entries(loadSources()).filter(([k]) => k === 'IARC-62')),
    };
  }
  const raw = fs.readFileSync(src, 'utf8');
  const fm: Json = {};
  let body = raw;
  const m = raw.match(/^---\n([\s\S]*?)\n---\n?/);
  if (m) {
    body = raw.slice(m[0].length);
    for (const line of m[1].split('\n')) {
      const kv = line.match(/^([A-Za-z_]+):\s*(.*)$/);
      if (kv) fm[kv[1]] = kv[2].replace(/^["']|["']$/g, '');
    }
  }
  const foot = new Map<string, number>();
  const toc: Json[] = [];
  const html: string[] = [];
  let list: string | null = null;
  let para: string[] = [];
  const flush = () => {
    if (para.length) html.push(`<p>${inline(para.join(' '), foot)}</p>`);
    para = [];
  };
  const closeList = () => {
    if (list) html.push(`</${list}>`);
    list = null;
  };
  for (const line of body.split('\n')) {
    const h = line.match(/^(#{1,4})\s+(.*)$/);
    const li = line.match(/^\s*([-*]|\d+\.)\s+(.*)$/);
    if (h) {
      flush();
      closeList();
      const level = h[1].length;
      const id = slugify(h[2]);
      if (level > 1) toc.push({ id, text: h[2], level });
      html.push(`<h${level} id="${id}">${inline(h[2], foot)}</h${level}>`);
    } else if (li) {
      flush();
      const kind = /\d/.test(li[1]) ? 'ol' : 'ul';
      if (list !== kind) {
        closeList();
        html.push(`<${kind}>`);
        list = kind;
      }
      const task = li[2].match(/^\[( |x)\]\s+(.*)$/);
      html.push(
        task
          ? `<li><input${task[1] === 'x' ? ' checked=""' : ''} disabled="" type="checkbox"> ${inline(task[2], foot)}</li>`
          : `<li>${inline(li[2], foot)}</li>`,
      );
    } else if (!line.trim()) {
      flush();
      closeList();
    } else {
      closeList();
      para.push(line.trim());
    }
  }
  flush();
  closeList();
  const all = loadSources();
  const sources: Record<string, Json> = {};
  for (const k of foot.keys()) if (all[k]) sources[k] = all[k];
  const imgFile = `${pagePath.replace(/\//g, '-')}.jpg`;
  return {
    slug: pagePath,
    title: fm.title ?? titleCase(pagePath.replace(/[/-]/g, ' ')),
    description: fm.description ?? '',
    last_reviewed: fm.last_reviewed ?? '',
    html: html.join('\n'),
    toc,
    sources,
    image: fs.existsSync(path.join(imagesDir, 'pages', imgFile))
      ? { url: `/images/pages/${imgFile}`, alt: fm.image_alt ?? '', width: 1200, height: 630 }
      : undefined,
  };
}

// --- HTTP --------------------------------------------------------------------

const PLACEHOLDER_SVG = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1200 630"><defs><linearGradient id="g" x1="0" y1="0" x2="1" y2="1"><stop offset="0" stop-color="#d2ae84"/><stop offset="1" stop-color="#a87447"/></linearGradient></defs><rect width="1200" height="630" fill="url(#g)"/><g fill="none" stroke="#7a4e24" stroke-opacity=".28" stroke-width="5">${Array.from(
  { length: 12 },
  (_, i) => `<ellipse cx="600" cy="760" rx="${120 + i * 85}" ry="${60 + i * 52}"/>`,
).join('')}</g></svg>`;

function send(res: ServerResponse, status: number, body: unknown) {
  res.statusCode = status;
  res.setHeader('Content-Type', 'application/json; charset=utf-8');
  res.end(JSON.stringify(body));
}

const notFound = (res: ServerResponse, what = 'Not found') =>
  send(res, 404, { error: { code: 'not_found', message: what } });

function handleApi(url: URL, res: ServerResponse) {
  const p = url.pathname.replace(/^\/api\/v1/, '');
  let m: RegExpMatchArray | null;

  if (p === '/health') {
    return send(res, 200, { status: 'ok', species: loadSpecies().length, sources: Object.keys(loadSources()).length });
  }
  if (p === '/species') {
    const q = url.searchParams;
    let list = loadSpecies().map(summary);
    if (q.get('featured') === 'true') list = list.filter((s) => s.featured_rank != null);
    if (q.get('wood_type')) list = list.filter((s) => s.wood_type === q.get('wood_type'));
    if (q.get('hazard_type')) list = list.filter((s) => s.hazard_types.includes(q.get('hazard_type')));
    if (q.get('evidence_level'))
      list = list.filter((s) => String(s.max_evidence_level) === q.get('evidence_level'));
    list.sort((a, b) => (a.featured_rank ?? 1e9) - (b.featured_rank ?? 1e9) || a.name.localeCompare(b.name));
    return send(res, 200, { species: list });
  }
  if ((m = p.match(/^\/species\/([^/]+)$/))) {
    const s = loadSpecies().find((x) => x.slug === decodeURIComponent(m![1]));
    return s ? send(res, 200, detail(s)) : notFound(res, 'No such species');
  }
  if (p === '/search') {
    const q = url.searchParams.get('q') ?? '';
    const limit = Math.min(Number(url.searchParams.get('limit')) || 8, 25);
    return send(res, 200, { query: q, results: search(q, limit) });
  }
  if (p === '/sources') return send(res, 200, { sources: loadSources() });
  if ((m = p.match(/^\/sources\/([^/]+)$/))) {
    const s = loadSources()[decodeURIComponent(m[1])];
    return s ? send(res, 200, s) : notFound(res, 'No such source');
  }
  if ((m = p.match(/^\/pages\/(.+)$/))) {
    const page = renderPage(decodeURIComponent(m[1]));
    return page ? send(res, 200, page) : notFound(res, 'No such page');
  }
  return notFound(res);
}

const MIME: Record<string, string> = {
  '.jpg': 'image/jpeg',
  '.jpeg': 'image/jpeg',
  '.png': 'image/png',
  '.webp': 'image/webp',
  '.svg': 'image/svg+xml',
};

function handleImage(url: URL, res: ServerResponse, next: () => void) {
  const rel = decodeURIComponent(url.pathname.replace(/^\/images\//, ''));
  const file = path.resolve(imagesDir, rel);
  if (file.startsWith(imagesDir + path.sep) && fs.existsSync(file) && fs.statSync(file).isFile()) {
    res.setHeader('Content-Type', MIME[path.extname(file).toLowerCase()] ?? 'application/octet-stream');
    fs.createReadStream(file).pipe(res);
    return;
  }
  if (rel === 'placeholder.svg') {
    res.setHeader('Content-Type', 'image/svg+xml');
    res.end(PLACEHOLDER_SVG);
    return;
  }
  res.statusCode = 404;
  res.end();
  void next;
}

export function fixturesPlugin(): Plugin {
  return {
    name: 'wooddust-fixtures',
    apply: 'serve',
    configureServer(server) {
      server.config.logger.info('  ➜  Fixtures: serving /api/v1 and /images from ../data and ../content/pages');
      server.middlewares.use((req: IncomingMessage, res: ServerResponse, next: () => void) => {
        const url = new URL(req.url ?? '/', 'http://localhost');
        try {
          if (url.pathname.startsWith('/api/v1/')) return handleApi(url, res);
          if (url.pathname.startsWith('/images/')) return handleImage(url, res, next);
        } catch (err) {
          return send(res, 500, { error: { code: 'fixture_error', message: String(err) } });
        }
        next();
      });
    },
  };
}
