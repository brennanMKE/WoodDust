# Architecture

One Go binary (`cmd/wooddust`) serves the site and API on `127.0.0.1:$PORT` behind Apache. Published data lives in git (`data/`, `content/pages/`) and is copied into Postgres by `wooddust seed`. The Svelte SPA (`web/`) is built to `web/dist` and embedded.

```
data/*.json ──seed──► Postgres (published tables) ──► internal/store ─┐
content/pages/*.md ──► internal/content (rendered at startup) ─────────┼─► internal/site ─► middleware ─► net/http
web/dist (embedded) ──► page shell with <!--ssr-*--> markers ──────────┘
```

## Packages

| Package | Role |
|---|---|
| `cmd/wooddust` | Subcommands `serve`, `seed [-check] [-strict]`, `migrate up\|down [N]\|version`, `apikey create\|list\|revoke`, `users list\|admin\|unadmin`, `identity <portiko cmd>`; wires config, stores, identity, middleware |
| `internal/config` | Environment config (`.env.example`). Bind host is fixed to `127.0.0.1` |
| `internal/model` | Types mirroring `data/schema`, display labels (contract §5), `Normalize` and the search score formula |
| `internal/data` | Loads `data/` strictly (unknown fields fail) and runs cross-file integrity checks |
| `internal/content` | Markdown pages: YAML front matter, goldmark, `[^KEY]` citation markers, heading ids, TOC, anchor checks |
| `internal/db` | pgx pool and golang-migrate with migrations embedded from `migrations/` |
| `internal/store` | Postgres catalog: queries, search SQL, and the transactional `Seed` |
| `internal/memstore` | In-memory catalog from a `data.Dataset`: the handler-test fake, and `serve` without `DATABASE_URL` |
| `internal/site` | Handlers for the API (contract §3) and server-rendered pages (contract §1–2), sitemap, robots, images, assets |
| `internal/contrib` | Anonymous submissions (validation, form tokens, honeypot, IP hashing, submission limiter), admin API, API keys, stats, export drafts, the auth layer (session or key, scopes, Origin CSRF check), and the Postgres store for migration `000002` incl. the `rp/session` store |
| `internal/identity` | Embedded Portiko provider and the site's `rp` client; mounts identity paths, `/auth/*`, the bootstrap enrollment gate; `wooddust identity` passthrough. See `docs/portiko-integration.md` |
| `internal/middleware` | Security headers/CSP, gzip, request logging, client IP via `TRUSTED_PROXIES`, per-IP API rate limit |
| `web` | `//go:embed all:dist` (see `web/embed.go` for the `.gitkeep` arrangement) |

Handlers depend on the narrow `site.Catalog` interface, never on a concrete store, so tests run with `memstore` and no database.

## Data model

Migration `000001` creates `species` (full record also kept as `doc jsonb`), `species_names`, `hazards`, `hazard_sources`, `sources` (`doc jsonb`), `exposure_limits`, `carcinogen_classes`, `regulatory` (one row: the raw `regulatory.json` plus the derived summary) and `seed_runs`. The API reads the `doc` columns; the normalized tables exist for SQL queries and future features.

`species_names.normalized` is a generated column, `wd_normalize(name)` = lowercase, `unaccent`, non-alphanumeric runs to one space. It has a `gin_trgm_ops` index (fuzzy) and a `text_pattern_ops` index (prefix). Go's `model.Normalize` must agree; a DB test checks every name.

`seed` truncates and refills every published table in one transaction under an advisory lock, so it is idempotent and safe to run on every start. Future tables (submissions, users, API keys) must refer to species by slug text, not by foreign key, so reseeding can truncate.

## Search

Each species is scored by its best-matching name: exact 1.0; whole-name prefix 0.90–0.99; word prefix 0.80–0.89; otherwise `pg_trgm` similarity × 0.79 when ≥ 0.3. Prefix matches always outrank fuzzy ones. Ties go to featured rank, then name. The same formula is in SQL (`internal/store`) and Go (`model.MatchScore`); a DB test checks they agree.

## Aliases and integrity

`/wood/<anything>` that is not a slug is normalized and looked up in `species_names`. One match redirects 301; several (e.g. "palisander") give a 404 that links to each candidate. `seed -check` reports such shared names as warnings and fails on errors: unknown source keys (species, regulatory, content pages), duplicate ids/slugs/featured ranks, a slug that is another species' alias, and `/page#id` links in content that match no heading.

## Page rendering

Every page route renders `web/dist/index.html` with three markers replaced: head (title, description, canonical, robots, OG/Twitter with image size and alt, JSON-LD), body (semantic HTML of the content, from `internal/site/templates`), and `<script type="application/json" id="initial">` holding `{route, data, site}` where `data` equals the matching API response. Without a built SPA a minimal built-in shell is used. Unknown routes return a real 404 with `noindex`; `/?q=` is `noindex`.

Content pages: the leading `# Title` in a Markdown file is dropped (the title comes from front matter and is rendered once as the `<h1>`). Front-matter `sources` not cited inline are listed after the numbered ones.

## Security

CSP is same-origin only, with no inline script allowance: the only inline scripts are JSON blocks (`application/json`, `application/ld+json`), which browsers never execute; `json.Marshal` escapes `<`, `>` and `&` so data cannot close the tag. Also `frame-ancestors 'none'`, `nosniff`, `Referrer-Policy`, `X-Frame-Options`, `Permissions-Policy`. `/api/` is rate-limited per client IP (in memory); the client IP comes from `X-Forwarded-For` only when the peer is in `TRUSTED_PROXIES`. The one CSP exception is Portiko's own pages (`/signin`, `/u/`, …), whose templates need inline script and style; they get a separate same-origin policy with `'unsafe-inline'` (docs/portiko-integration.md §2).

## Contributions, sign-in and API keys

Migration `000002` adds `users`, `rp_sessions`, `submissions`, `submission_events`, `api_keys` and `audit_log` (species referenced by slug text). `internal/contrib` registers the submission and admin endpoints on the site mux (contract §6–7); `internal/site` renders `/submit`, `/submit/status/{token}` and `/admin*` with their `#initial` data. The Portiko provider runs in-process against its own database (`IDENTITY_DATABASE_URL`) and is mounted only on its identity paths; see `docs/portiko-integration.md`. Without `DATABASE_URL`, submission endpoints answer 503; without `IDENTITY_DATABASE_URL`, sign-in is disabled and `/admin` answers 503 while API keys keep working. API docs: `docs/api.md`.
