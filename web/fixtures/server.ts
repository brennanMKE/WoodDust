// Dev-only fixture API (VITE_FIXTURES=1 npm run dev). Answers /api/v1/* and
// /images/* from the repo's real data files so the SPA can be developed
// without the Go server. Loaded only by vite.config.ts during `vite serve`;
// never imported by src/, so it is not in the production bundle.
//
// It approximates docs/contract.md; the Go server is the reference. Content
// pages are rendered with a deliberately tiny Markdown subset.
import crypto from 'node:crypto';
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

// --- Submissions, admin and API keys (contract §6-§7), in memory ----------
//
// Approximates the Go behaviour so every screen can be exercised:
// - POST /submissions validates like Go (422), fakes a 201 for honeypot hits,
//   and rate-limits to 5 per 10 minutes (429 + Retry-After). Put "ratelimit"
//   in the details to force a 429.
// - The admin session is a module flag: /auth/logout clears it (then /admin
//   redirects to /auth/login, which signs straight back in).

type Sub = Json & { id: number; token: string; events: Json[] };
const subs: Sub[] = [];
const keys: Json[] = [];
let signedIn = true;
let posts: number[] = [];
const FORM_SECRET = crypto.randomBytes(16).toString('hex');
const LEADS = ['wood-database.com', 'haz-map.com', 'scrollsawvillage.com', 'woodbin.com', 'wooduchoose.com', 'finewoodworking.com'];
const HAZARDS = new Set([
  'respiratory_sensitizer', 'respiratory_irritant', 'skin_sensitizer', 'skin_irritant', 'eye_irritant',
  'hypersensitivity_pneumonitis', 'systemic_toxic', 'cardiac', 'carcinogen', 'splinter_infection', 'photosensitization',
]);
const SUB_KINDS = ['new_species', 'correction', 'new_source', 'reaction_report'];

const now = () => new Date().toISOString();
const ref = (id: number) => `DA-${String(id).padStart(4, '0')}`;
const sign = (t: string) => crypto.createHmac('sha256', FORM_SECRET).update(t).digest('hex').slice(0, 16);
const formToken = () => {
  const t = String(Date.now());
  return `${t}.${sign(t)}`;
};
const base62 = (n: number) => {
  const a = '0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz';
  return Array.from(crypto.randomBytes(n), (b) => a[b % 62]).join('');
};

function leadsWarning(url: string): string | null {
  try {
    const host = new URL(url).hostname.replace(/^www\./, '');
    if (LEADS.some((d) => host === d || host.endsWith('.' + d)) || host.includes('forum'))
      return `${host} is a leads-only source: we can't cite it. Please give the study or agency page it relies on.`;
  } catch {
    /* not a URL */
  }
  return null;
}

function readBody(req: IncomingMessage): Promise<Json | null> {
  if (req.method === 'GET' || req.method === 'HEAD') return Promise.resolve(null);
  return new Promise((resolve) => {
    let raw = '';
    req.on('data', (c) => (raw += c));
    req.on('end', () => {
      try {
        resolve(raw ? JSON.parse(raw) : null);
      } catch {
        resolve(null);
      }
    });
  });
}

function seedSubmissions() {
  if (subs.length) return;
  const mk = (kind: string, status: string, extra: Json, hoursAgo: number) => {
    const id = subs.length + 1;
    const at = new Date(Date.now() - hoursAgo * 3600e3).toISOString();
    subs.push({
      id, reference: ref(id), token: crypto.randomBytes(16).toString('hex'), kind, status,
      details: 'Fixture submission written in the submitter’s own words, long enough to pass validation.',
      citations: [], email: null, ip_hash: crypto.randomBytes(16).toString('hex'), created_at: at, updated_at: at,
      events: [{ at, actor: 'submitter', type: 'created', to_status: 'new' }], ...extra,
    });
  };
  mk('new_species', 'new', { species_name: 'Bubinga', botanical_name: 'Guibourtia spp.', hazard_type: 'skin_sensitizer',
    citations: [{ doi: '10.1111/j.1600-0536.1990.tb01568.x', text: 'Hausen? check' }, { url: 'https://www.wood-database.com/bubinga/' }],
    warnings: ['wood-database.com is a leads-only source.'], email: 'maker@example.com' }, 30);
  mk('correction', 'triaging', { target_slug: 'oak', hazard_type: 'carcinogen', citations: [{ pmid: '2021399' }] }, 50);
  mk('new_source', 'accepted', { target_slug: 'western-red-cedar', hazard_type: 'respiratory_sensitizer',
    citations: [{ doi: '10.1136/oem.48.3.201', text: 'Chan-Yeung 1991' }] }, 80);
  mk('reaction_report', 'needs_info', { target_slug: 'cocobolo', hazard_type: 'skin_sensitizer' }, 5);
  subs[3].events.push({ at: now(), actor: 'owner', type: 'note', note: { text: 'Did the rash start after sanding or after handling the finished piece?', visible_to_submitter: true } });
  mk('new_species', 'spam', { species_name: 'Cheap pills' }, 2);
  keys.push({ name: 'triage script', prefix: base62(8), scopes: ['submissions:read', 'stats:read'], created_at: new Date(Date.now() - 864e5 * 9).toISOString(), last_used_at: now(), expires_at: null, revoked_at: null });
}

function statusView(s: Sub): Json {
  return {
    reference: s.reference, kind: s.kind, status: s.status === 'spam' ? 'new' : s.status,
    created_at: s.created_at, updated_at: s.updated_at,
    notes: s.events.filter((e) => e.note?.visible_to_submitter).map((e) => ({ at: e.at, text: e.note.text })),
  };
}

const adminView = ({ token: _t, events: _e, ...rest }: Sub) => rest;

async function createSubmission(req: IncomingMessage, res: ServerResponse) {
  seedSubmissions();
  const b = (await readBody(req)) ?? {};
  const t = Date.now();
  posts = posts.filter((x) => t - x < 600e3);
  if (posts.length >= 5 || String(b.details ?? '').includes('ratelimit')) {
    res.setHeader('Retry-After', String(Math.ceil((600e3 - (t - (posts[0] ?? t))) / 1000) || 600));
    return send(res, 429, { error: { code: 'rate_limited', message: 'Too many submissions; try again later.' } });
  }
  posts.push(t);
  const fields: Record<string, string> = {};
  const [issued, sig] = String(b.form_token ?? '').split('.');
  if (!issued || sign(issued) !== sig) fields.form_token = 'The form token is missing or invalid.';
  if (!SUB_KINDS.includes(b.kind)) fields.kind = 'Unknown kind.';
  const details = String(b.details ?? '').trim();
  if ([...details].length < 20 || [...details].length > 4000) fields.details = 'Details must be 20 to 4000 characters.';
  if (b.kind === 'new_species' && !String(b.species_name ?? '').trim()) fields.species_name = 'Required.';
  if ((b.kind === 'correction' || b.kind === 'new_source') && !b.target_slug) fields.target_slug = 'Choose a species.';
  if (b.target_slug && !loadSpecies().some((x) => x.slug === b.target_slug)) fields.target_slug = 'No such species.';
  if (b.hazard_type && !HAZARDS.has(b.hazard_type)) fields.hazard_type = 'Unknown hazard type.';
  const cites: Json[] = Array.isArray(b.citations) ? b.citations : [];
  const warnings: string[] = [];
  cites.forEach((c, i) => {
    if (!c.doi && !c.pmid && !c.url) fields[`citations[${i}]`] = 'Needs a DOI, PMID or URL.';
    else if (c.doi && !/^10\.\d{4,9}\/\S+$/.test(c.doi)) fields[`citations[${i}].doi`] = 'Not a valid DOI.';
    else if (c.pmid && !/^\d+$/.test(c.pmid)) fields[`citations[${i}].pmid`] = 'A PMID is digits only.';
    else if (c.url && !/^https?:\/\//.test(c.url)) fields[`citations[${i}].url`] = 'URLs must be http(s).';
    const w = c.url ? leadsWarning(c.url) : null;
    if (w) warnings.push(w);
  });
  if (b.kind !== 'reaction_report' && !cites.length) fields.citations = 'At least one citation is required.';
  if (b.email && !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(b.email)) fields.email = 'Not a valid email.';
  if (Object.keys(fields).length)
    return send(res, 422, { error: { code: 'invalid', message: 'Some fields need fixing.', fields } });
  const spam = Boolean(b.website) || t - Number(issued) < 3000;
  const id = subs.length + 1;
  const at = now();
  const sub: Sub = {
    id, reference: ref(id), token: crypto.randomBytes(16).toString('hex'), kind: b.kind, status: spam ? 'spam' : 'new',
    target_slug: b.target_slug ?? null, species_name: b.species_name ?? null, botanical_name: b.botanical_name ?? null,
    hazard_type: b.hazard_type ?? null, details, citations: cites, email: b.email || null,
    ip_hash: crypto.createHash('sha256').update('127.0.0.1' + at.slice(0, 10)).digest('hex'),
    warnings, created_at: at, updated_at: at, events: [{ at, actor: 'submitter', type: 'created', to_status: spam ? 'spam' : 'new' }],
  };
  subs.push(sub);
  return send(res, 201, { id, reference: sub.reference, status_url: `/submit/status/${sub.token}`, ...(warnings.length ? { warnings } : {}) });
}

function handleSubmissionApi(req: IncomingMessage, url: URL, p: string, res: ServerResponse, body: Json | null): boolean {
  seedSubmissions();
  let m: RegExpMatchArray | null;
  const done = (status: number, b: unknown) => (send(res, status, b), true);
  if (p === '/submissions/form-token') return done(200, { form_token: formToken() });
  if (p === '/submissions/check-url') {
    const w = leadsWarning(url.searchParams.get('url') ?? '');
    return done(200, { url: url.searchParams.get('url'), leads_only: Boolean(w), warnings: w ? [w] : [] });
  }
  if ((m = p.match(/^\/submissions\/status\/([^/]+)$/))) {
    const s = subs.find((x) => x.token === m![1]);
    return s ? done(200, statusView(s)) : done(404, { error: { code: 'not_found', message: 'Unknown status link' } });
  }
  // Everything below needs the (fake) owner session.
  if (!signedIn) return done(401, { error: { code: 'unauthenticated', message: 'Sign in required' } });
  if (p === '/me') return done(200, { user: { display_name: 'Fixture Owner', is_admin: true } });
  if (p === '/stats') {
    const count = (f: (s: Sub) => string) => subs.reduce<Record<string, number>>((a, s) => ((a[f(s)] = (a[f(s)] ?? 0) + 1), a), {});
    const day = Date.now() - 864e5;
    return done(200, {
      open: subs.filter((s) => ['new', 'triaging', 'needs_info'].includes(s.status)).length,
      by_status: count((s) => s.status), by_kind: count((s) => s.kind),
      newest_at: subs.length ? subs[subs.length - 1].created_at : null,
      new_last_24h: subs.filter((s) => Date.parse(s.created_at) > day && s.status !== 'spam').length,
      spam_last_24h: subs.filter((s) => Date.parse(s.created_at) > day && s.status === 'spam').length,
    });
  }
  if (p === '/submissions' && req.method === 'GET') {
    const q = url.searchParams;
    let list = [...subs].reverse();
    if (q.get('status')) list = list.filter((s) => s.status === q.get('status'));
    if (q.get('kind')) list = list.filter((s) => s.kind === q.get('kind'));
    const limit = Math.min(Number(q.get('limit')) || 25, 100);
    const start = Number(q.get('cursor') || 0);
    const page = list.slice(start, start + limit);
    return done(200, { submissions: page.map(adminView), next_cursor: start + limit < list.length ? String(start + limit) : null });
  }
  if ((m = p.match(/^\/submissions\/(\d+)(\/export)?$/))) {
    const s = subs.find((x) => x.id === Number(m![1]));
    if (!s) return done(404, { error: { code: 'not_found', message: 'No such submission' } });
    if (m[2]) {
      if (!['accepted', 'exported', 'merged'].includes(s.status))
        return done(409, { error: { code: 'not_accepted', message: 'Only accepted submissions can be exported.' } });
      const slug = s.target_slug ?? String(s.species_name ?? 'new').toLowerCase().replace(/[^a-z0-9]+/g, '-');
      return done(200, {
        path: `data/species/${slug}.json`,
        submission: s.reference,
        patch: { hazards: [{ type: s.hazard_type, summary: '(write in our own words)', evidence_level: 3, sources: s.citations.map((c: Json) => c.doi ?? c.pmid ?? c.url) }] },
      });
    }
    if (req.method === 'PATCH') {
      const at = now();
      if (body?.status && body.status !== s.status) {
        s.events.push({ at, actor: 'owner', type: 'status', from_status: s.status, to_status: body.status });
        s.status = body.status;
      }
      if (body?.note?.text) s.events.push({ at, actor: 'owner', type: 'note', note: { text: body.note.text, visible_to_submitter: Boolean(body.note.visible_to_submitter) } });
      s.updated_at = at;
      return done(200, { ...adminView(s), events: s.events });
    }
    return done(200, { ...adminView(s), events: s.events });
  }
  if (p === '/admin/api-keys' && req.method === 'GET') return done(200, { keys });
  if (p === '/admin/api-keys' && req.method === 'POST') {
    if (!body?.name || !Array.isArray(body.scopes) || !body.scopes.length)
      return done(422, { error: { code: 'invalid', message: 'Name and at least one scope are required.' } });
    const prefix = base62(8);
    const k = { name: body.name, prefix, scopes: body.scopes, created_at: now(), last_used_at: null, expires_at: body.expires_at ?? null, revoked_at: null };
    keys.unshift(k);
    return done(201, { ...k, key: `da_${prefix}_${base62(43)}` });
  }
  if ((m = p.match(/^\/admin\/api-keys\/([^/]+)$/)) && req.method === 'DELETE') {
    const k = keys.find((x) => x.prefix === m![1]);
    if (!k) return done(404, { error: { code: 'not_found', message: 'No such key' } });
    k.revoked_at = now();
    res.statusCode = 204;
    res.end();
    return true;
  }
  return false;
}

function handleAuth(url: URL, res: ServerResponse) {
  // Fake Portiko: logout clears the session, login sets it and returns.
  signedIn = url.pathname === '/auth/login';
  const back = url.searchParams.get('return');
  res.statusCode = 302;
  res.setHeader('Location', signedIn && back?.startsWith('/') ? back : '/');
  res.end();
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

async function handleApi(req: IncomingMessage, url: URL, res: ServerResponse) {
  const p = url.pathname.replace(/^\/api\/v1/, '');
  let m: RegExpMatchArray | null;

  if (p === '/submissions' && req.method === 'POST') return createSubmission(req, res);
  if (p.startsWith('/submissions') || p === '/stats' || p === '/me' || p.startsWith('/admin/')) {
    if (handleSubmissionApi(req, url, p, res, await readBody(req))) return;
  }

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
          if (url.pathname.startsWith('/api/v1/')) {
            handleApi(req, url, res).catch((err) =>
              send(res, 500, { error: { code: 'fixture_error', message: String(err) } }),
            );
            return;
          }
          if (url.pathname === '/auth/login' || url.pathname === '/auth/logout') return handleAuth(url, res);
          if (url.pathname.startsWith('/images/')) return handleImage(url, res, next);
        } catch (err) {
          return send(res, 500, { error: { code: 'fixture_error', message: String(err) } });
        }
        next();
      });
    },
  };
}
