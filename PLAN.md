# WoodDust build plan

Companion to `wood-dust-site-handoff.md`. The handoff's data policy (§1–5) still governs. This plan **replaces its §6 "Technical"** (static site, no backend) with the stack below, and **adds** contributor submissions, Portiko sign-in, and a keyed REST API.

## 1. Stack

| Layer | Choice |
|---|---|
| Public endpoint | Apache 2 vhost, TLS, `ProxyPass` to `127.0.0.1:<port>`; serves nothing itself except `/images/` (static, long cache) |
| App server | Go, standard-library `net/http` with 1.22+ pattern routing, bound to localhost only. One binary with `serve`, `seed`, `export`, `apikey` subcommands |
| Database | PostgreSQL, `pgx/v5`, `golang-migrate` append-only migrations |
| Front end | Svelte 5 SPA, built to `web/dist/` and embedded with `//go:embed` |
| Sign-in | Portiko (passkeys), **embedded in the same binary**: `provider` mounted on the site's own mux, `rp` + `rp/session` for the site's sessions. No external identity provider |
| Hosting | Same pattern as other sites: systemd unit, release symlinks, `ssh photon` |

Follows the same conventions as `~/Developer/OpenCircuitSF/Website` (narrow store interfaces in handlers, SQL in the owning store package, `scripts/check.sh` as canonical verification).

## 2. Data flow: git is the source of truth

```
data/species/*.json ──(CI: schema + source-key checks)──► git
        │
        └── wooddust seed ──► Postgres (species, hazards, sources, aliases)
                                   ▲
submissions (Postgres) ── admin approves ── wooddust export <id> ──► JSON patch / PR to data/
```

- `seed` is idempotent and replaces the published-data tables in one transaction. It runs on every deploy.
- Submissions never edit published data directly. An approved submission is exported as a change to `data/species/<slug>.json` (plus `sources`) and goes through a pull request and CI like any other edit. This keeps every hazard claim reviewed, sourced, and in git history.
- Later option: `export` opens the PR itself via `gh`.

## 3. URLs and SEO

The SPA uses history routing; the Go server answers every route with **server-rendered head + content** so crawlers and link unfurlers never depend on JavaScript.

| Path | Page |
|---|---|
| `/` | Top 5 (featured_rank), search box, "all wood dust is hazardous" headline |
| `/wood/{slug}` | Species card |
| `/browse` | Filterable table/cards |
| `/risks`, `/protection/*`, `/symptoms`, `/limits`, `/glossary` | Education pages (§3a) |
| `/sources`, `/methodology` | Sources, evidence levels, data policy, how to contribute |
| `/submit`, `/submit/{slug}` | New species / correction form (sign-in required) |
| `/account` | My submissions, sign out |
| `/admin/*` | Submission queue, API keys (admin only, `noindex`) |
| `/api/v1/*` | REST API |
| `/auth/*` | Portiko `rp` endpoints (callback, logout) |
| `/login`, `/signin`, `/u/*`, `/oauth/*`, `/accounts/*` | Embedded Portiko provider (passkey ceremonies, account page) — `noindex` |
| `/sitemap.xml`, `/robots.txt` | Generated from the DB |

For each request to a page route, Go renders `index.html` from a template with:

- Unique `<title>`, `meta description`, `<link rel="canonical">`.
- Open Graph and Twitter tags: `og:title`, `og:description`, `og:url`, `og:type=article` on cards, `og:image` = `/images/og/{slug}.jpg` (1200×630), `og:image:alt` = `image_alt`, `og:site_name`.
- JSON-LD: `WebSite` + `SearchAction` on `/`; `Article` (with `about`, `citation` list, `dateModified` = `last_reviewed`) and `BreadcrumbList` on cards.
- The card's actual content (names, hazards, evidence labels, sources) as semantic HTML inside the mount element. Svelte hydrates over it or replaces it. Same for `/browse` and content pages.
- Initial data inlined as `<script type="application/json" id="initial">` so the SPA doesn't refetch.

Rules:
- Unknown slug → real **404** status with a noindex page. Aliases (`/wood/wrc`, `/wood/thuja-plicata`) → **301** to the canonical slug.
- Search result pages (`/?q=`) are `noindex`; `/submit`, `/account`, `/admin` are `noindex` and disallowed in robots.txt.
- Sitemap lists every species with `lastmod` = `last_reviewed`.
- No tracking scripts, no third-party fonts; target Lighthouse 95+ and WCAG 2.2 AA (handoff §6).

## 3a. Education pages: wood dust risks and protection

These pages are a core part of the site, not an appendix. Many people searching for one wood are really asking "is this dust bad for me, and what should I wear?". Every species card links to them, and the homepage has a short "All wood dust is hazardous" section pointing to `/risks` and `/protection`.

| Path | Covers | Main sources |
|---|---|---|
| `/risks` | Why all wood dust is hazardous: nasal and sinus cancer (IARC Group 1); sensitization and occupational asthma, which can be permanent and worsen with every exposure; rhinitis; dermatitis; eye irritation; species with toxic compounds; fire and dust-explosion risk. Exposure adds up over years. Fine dust you can't see matters most | IARC-62/100C, OSHA-CSI, HSE, Baatjies-2023 |
| `/protection` | Overview using the hierarchy of controls: remove dust at the source first, then clean the air, then wear protection. A one-screen checklist | NIOSH, HSE, OSHA |
| `/protection/dust-collection` | Extraction at the tool, dust collectors vs shop vacs, filter ratings, ambient air cleaners, why sanding makes the worst dust | NIOSH, HSE |
| `/protection/respirators` | Dust masks vs N95 vs P100 vs PAPR. Fit testing and seal checks. Why beards break the seal. When to replace filters. Cartridges needed for finishing | NIOSH, OSHA respiratory protection |
| `/protection/skin-and-eyes` | Gloves, long sleeves, washing, barrier creams (limits), safety glasses vs goggles, irritating species | HSE, NIOSH |
| `/protection/cleanup` | HEPA vacuum and damp wipe. Never compressed air or dry sweeping. Clothing and taking dust home | OSHA, HSE |
| `/protection/fire-and-explosion` | Combustible dust, collector fires, spark sources, oily rags from finishing | OSHA combustible dust, NFPA references (cite, don't copy) |
| `/symptoms` | Signs to watch for (cough, wheeze or stuffy nose that improves on days away from the shop, rashes, nosebleeds) and when to see a doctor. Not medical advice; plain disclaimer | NIOSH, HSE |
| `/limits` | OSHA vs NIOSH vs ACGIH vs UK vs EU limits, and what 1 mg/m³ looks like in a hobby shop | `data/regulatory.json` |
| `/glossary` | Sensitizer, irritant, inhalable vs respirable, etc. Terms are linked from cards with hover definitions | |

Rules:
- Our own words. Every factual claim links to a source from `data/sources.json`. Same data policy as the species data.
- Calm and practical. Explain why, then what to do. No fear-mongering, no product endorsements, no affiliate links.
- Content lives as Markdown with front matter in `content/pages/`. Go renders it server-side for SEO, with the same head/OG/JSON-LD treatment (`Article`, `BreadcrumbList`).
- Each page gets its own OG image prompt (stored in front matter), e.g. a woodworker wearing a respirator at a sanding station.
- A printable shop poster (`/print/shop-poster`): a one-page summary of the protection checklist.
- Drafting begins once the agency agent's NIOSH/OSHA/HSE guidance sources are in `data/sources/agency.json`.

## 4. Search

- `GET /api/v1/search?q=` over common names, aliases, botanical names and synonyms. Postgres `pg_trgm` similarity plus prefix match on a normalized (lowercased, diacritics stripped) name table.
- The front end debounces input, shows suggestions as the user types (Google-style), and Enter goes to the top match's `/wood/{slug}`. No match → "Not listed yet. Suggest it" linking to `/submit?name=…`.

## 5. Sign-in with Portiko (embedded, self-contained)

WoodDust runs its own Portiko provider in-process. No `accounts.*` host and no dependency on another site. Pin a Portiko commit in `go.mod` (the README says `main` is a preview and nothing is tagged).

**Provider, in the same binary:**
- `provider.New(ctx, cfg, provider.Options{Mailer: …})` at startup. Build `provider.Config` in code from WoodDust's config so its `DATABASE_URL` and `PORT` don't clash with the site's.
  - `Issuer` = the site origin.
  - `WebAuthnRPID` = the apex domain.
  - `WebAuthnRPOrigins` = the site origin.
  - `TrustedProxies` = `127.0.0.1`, because Apache fronts it (see Portiko `docs/rate-limits.md`).
  - Also `CRYPTO_KEY`, SMTP settings for one-time codes, and `ServiceName` = the site name.
- **Separate Postgres database** (e.g. `wooddust_identity`), because Portiko runs its own embedded migrations and both projects use golang-migrate's `schema_migrations` table. Same Postgres server, separate database.
- **Route mounting:** the provider's router claims `GET /{$}`, `/health`, `/login`, `/signin`, `/profile`, `/u/{n}/`, `/oauth/`, `/accounts/`, `/.well-known/openid-configuration`, `/.well-known/jwks.json` and the Portiko discovery paths. WoodDust owns `/`, so mount `p.Handler()` only on the identity paths (`/oauth/`, `/accounts/`, `/.well-known/…`, `/login`, `/login/select`, `/signin`, `/u/`). WoodDust keeps `/`, `/health` and `/profile` for itself.
  - Verify during the build that Portiko's templates and redirects don't assume it owns `/`. If they do, open a Portiko issue asking for an option to serve under a path prefix, rather than working around it.
- `p.Sweep(ctx)` runs in a goroutine, and `p.Close()` runs on shutdown.
- Operator commands come from `server.Main`, wrapped as `wooddust identity <cmd>` (e.g. `register-client`, `rotate-key`, `keys`, `seal-keys`).

**Site sessions:**
- Register the site as the provider's single first-party client: `wooddust identity register-client -first-party wooddust <audience> https://<domain>/auth/callback`.
- The site then uses `rp` for sign-in, callback, refresh, logout and `RequireUser`, with sessions stored by `rp/session`. Needs `COOKIE_KEY`.
- The redirect to the provider and back stays on the same origin, so a person sees a single sign-in.
- The REST API uses `verify` to accept Portiko access tokens, alongside API keys (§7).
- `BOOTSTRAP_CODE` enrolls the first account (you), who is then marked `is_admin` locally.

**Authorization stays in WoodDust:** a local `users` row keyed by Portiko subject holds `display_name`, `is_admin`, `is_moderator`, `active`. Portiko provides identity only, per its own design.

Browsing, search and the data download stay anonymous. Only submitting, the account page and the admin console need sign-in.

## 6. Submissions (additions and corrections)

Kinds:

| Kind | Purpose | Required |
|---|---|---|
| `new_species` | Species not listed | Common name, botanical name if known, at least one hazard claim with a source |
| `correction` | Change to a listed species | Target slug, field or hazard being corrected, proposed change, source |
| `new_source` | Better/primary source for an existing claim | Target slug + hazard, citation with DOI/PMID/URL |
| `reaction_report` | "I reacted to this wood" | Species, symptoms, free text. Shows the "not a medical service, see a doctor" notice; never published; leads only |

Form rules (enforce the handoff's data policy at input time):
- Hazard claims require a citation with DOI, PMID or an official URL. The form validates DOI/PMID format and warns when the URL is a leads-only domain (Wood Database, Haz-Map, forums), as listed in `CONTRIBUTING.md`.
- A plain-language notice: write in your own words; do not paste abstracts or charts.
- Rate limit per user and per IP; size limits; no file uploads in v1.

Workflow: `new → triaging → needs_info → accepted | rejected → exported → merged`. Each change is recorded in an audit log, and the submitter can see the status on `/account`. Moderators can comment, and `needs_info` comments are shown to the submitter.

## 7. REST API

Base `/api/v1`, JSON only, versioned, documented in `docs/api.md` (and an OpenAPI file).

**Public (no key, rate-limited per IP):**
- `GET /species` (summary list; filters `hazard_type`, `evidence_level`, `wood_type`, `featured`)
- `GET /species/{slug}` (full record with sources)
- `GET /search?q=`
- `GET /sources`, `GET /sources/{key}`
- `GET /download/species.json`, `/download/species.csv` (CC BY 4.0, pending confirmation)

**Authenticated (API key or session):**
- `GET /submissions?status=&kind=&since=`
- `GET /submissions/{id}`
- `PATCH /submissions/{id}` (status, moderator note); requires `submissions:write`
- `GET /stats` → counts by status and kind, `new_since_last_check`, newest submission time. Built for Home Assistant
- `GET /health` (unauthenticated, for uptime checks)

**API keys:**
- Created and revoked in `/admin/api-keys` (or `wooddust apikey create --name ha --scopes submissions:read`).
- Format `wd_<8-char prefix>_<32 random bytes base62>`. The full key is shown **once**. Only the prefix and a SHA-256 hash are stored. Compare in constant time.
- Columns: `name`, `prefix`, `hash`, `scopes[]`, `owner_user_id`, `created_at`, `last_used_at`, `expires_at`, `revoked_at`.
- Sent as `Authorization: Bearer wd_…`. Every use updates `last_used_at` and writes to the audit log on writes.
- Scopes: `submissions:read`, `submissions:write`, `stats:read`.

**Home Assistant example** (`configuration.yaml`):

```yaml
rest:
  - resource: https://<domain>/api/v1/stats
    headers:
      Authorization: !secret wooddust_api_key
    scan_interval: 900
    sensor:
      - name: WoodDust open submissions
        value_template: "{{ value_json.open }}"
        json_attributes: [by_kind, newest_at, new_since_last_check]
```

## 8. Database (initial migrations)

- Published data (rebuilt by `seed`): `species`, `species_names` (name, kind: common/trade/botanical/synonym, normalized, trigram index), `hazards`, `hazard_sources`, `sources`, `exposure_limits`, `carcinogen_classes`, `seed_runs`.
- Users/auth: `users` (subject, roles), `rp/session` tables. Portiko's own tables live in the separate `wooddust_identity` database.
- Contributions: `submissions` (kind, status, target_slug, payload jsonb, submitter, timestamps), `submission_events` (status changes, comments), `audit_log`.
- API: `api_keys`.
- Rate limiting in memory (single host), like OpenCircuitSF.

## 9. Phases

1. **Data** (in progress): eight research agents fill `data/species`, `data/sources`, `data/verify`. Then merge into `data/sources.json`, run cross-file checks, and do a human review pass.
2. **Skeleton**: repo layout, migrations, `seed`, public read API, CI (schema validation, unknown source keys, hazard without source, link checker, `go test`, `svelte-check`).
3. **Public site**: SPA routes, server-rendered head and content, top 5 home, search, cards, browse, sitemap/robots, 404/301 handling. Generate images from the prompts in each species file into `images/og/{slug}.jpg`.
4. **Education pages** (§3a): risks, protection (5 subpages), symptoms, limits, glossary, sources/methodology, shop poster. Drafted by a research agent from NIOSH/OSHA/HSE, in our own words, then reviewed by you.
5. **Portiko sign-in + submissions**: embed the provider, mount its routes, `identity` subcommands, register the client, `rp` integration, bootstrap the admin, then forms, account page, moderation queue, `export`.
6. **API keys + Home Assistant**: key management, `/stats`, the HA config above, `docs/api.md`.
7. **Deploy**: Apache vhost, systemd unit, deploy script, backups, then a Lighthouse/a11y pass.

## 10. Open questions

- Domain name and site name (handoff: TBD). Needed for the Portiko issuer, WebAuthn RP ID, canonical URLs and OG tags.
- Outbound email for Portiko's one-time codes: SMTP or SES?
- License confirmation: MIT (code), CC BY 4.0 (data).
- Who moderates besides you, and should accepted contributors be credited on species pages?
