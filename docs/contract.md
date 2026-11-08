# Backend / front-end contract

Binding for the Go server (`cmd/`, `internal/`) and the Svelte SPA (`web/`). Change it only by editing this file in the same commit as both sides.

## 1. Page shell

Vite builds `web/dist/index.html`. It contains three markers that Go replaces on every page response:

```html
<head>
  <!--ssr-head-->          <!-- Go: <title>, meta description, canonical, OG/Twitter, JSON-LD, robots -->
  ...vite css/js tags...
</head>
<body>
  <div id="app"><!--ssr-body--></div>   <!-- Go: semantic HTML of the page content (no JS needed to read it) -->
  <!--ssr-data-->        <!-- Go: <script type="application/json" id="initial">{...}</script> -->
</body>
```

- The SPA reads `#initial` (`{ "route": "...", "data": {...} }`, where `data` matches the API response for that route), clears `#app`, and mounts. It never refetches the initial route's data.
- Client-side navigation uses the History API and fetches from `/api/v1/*`. Each navigation updates `document.title`, meta description and canonical.
- Go serves `web/dist` (embedded via `//go:embed`) for `/assets/*`, with an immutable cache for hashed files.
- Site name comes from config (`SITE_NAME`, default `Dust Almanac`). Go exposes it to the SPA in `#initial` as `site: { name, base_url }`.

## 2. Routes

| Path | Route id | Initial `data` = |
|---|---|---|
| `/` | `home` | `GET /api/v1/species?featured=true` |
| `/wood/{slug}` | `species` | `GET /api/v1/species/{slug}` |
| `/browse` | `browse` | `GET /api/v1/species` |
| `/risks`, `/protection`, `/protection/{sub}`, `/symptoms`, `/limits`, `/glossary`, `/methodology` | `page` | `GET /api/v1/pages/{path}` |
| `/print/shop-poster` | `page` | `GET /api/v1/pages/print/shop-poster` (SPA shows `images/art/poster-header.png` and uses a one-page print layout) |
| `/sources` | `sources` | `GET /api/v1/sources` |
| `/submit`, `/submit/{slug}` | `submit` | `{ "form_token": "...", "kinds": [...], "target": <species summary or null>, "prefill_name": "..." }`, `noindex`. Query `?kind=` preselects; `?name=` pre-fills new species |
| `/submit/status/{token}` | `submission_status` | `GET /api/v1/submissions/status/{token}`, `noindex`; 404 if the token is unknown |
| `/admin`, `/admin/submissions`, `/admin/submissions/{id}`, `/admin/api-keys` | `admin` | `{ "user": {...} }`. **Requires a session.** Without one, Go 302s to `/auth/login?return=<path>`. `noindex` |
| anything else | `not_found` | `{}` with HTTP 404 |

- Alias URLs (any normalized common name, trade name, botanical name or synonym, e.g. `/wood/wrc`, `/wood/thuja-plicata`) get a **301** to `/wood/{slug}`, from Go.
- `/?q=…` gets `<meta name="robots" content="noindex">`.

## 3. API (`/api/v1`, JSON, `Content-Type: application/json; charset=utf-8`)

Errors: `{"error": {"code": "not_found", "message": "..."}}` with a matching status.

### `GET /species[?featured=true&wood_type=&hazard_type=&evidence_level=]`
```json
{ "species": [ {
  "slug": "oak", "name": "Oak", "common_names": ["oak","red oak","white oak"],
  "botanical_name": "Quercus spp.", "wood_type": "hardwood", "region": "...",
  "featured_rank": 1,
  "hazard_types": ["respiratory_sensitizer","carcinogen"],
  "max_evidence_level": 1,
  "image": { "url": "/images/species/oak.jpg", "alt": "...", "width": 1200, "height": 630 },
  "url": "/wood/oak"
} ] }
```
Ordered by `featured_rank`, then name. `max_evidence_level` is the strongest *specific* level present (1 is strongest; ignore level 0). It is `null` if the species has only level 0 hazards.

### `GET /species/{slug}`
The species JSON record, **minus** `notes_internal`, `image_prompt` and `images`, **plus**:
- `name` (the first common name, title-cased)
- `url`
- `image` (as above)
- `sources`: an object mapping every key the record cites to its full `data/sources.json` entry, with `key` added.

Also includes the site-wide `regulatory` summary for limits display: `{ "eu_hardwood_mg_m3": 2, "uk_hardwood_mg_m3": 3, "uk_softwood_mg_m3": 5, "niosh_rel_mg_m3": 1 }`, sourced from `data/regulatory.json`.

### `GET /search?q=&limit=8`
```json
{ "query": "wrc", "results": [ {
  "slug": "western-red-cedar", "name": "Western red cedar", "botanical_name": "Thuja plicata",
  "matched_name": "WRC", "match_kind": "common|trade|botanical|synonym", "score": 0.93, "url": "/wood/western-red-cedar"
} ] }
```
Case- and diacritic-insensitive. Prefix matches rank above trigram similarity. An empty `q` returns `[]`.

### `GET /sources`, `GET /sources/{key}`
`{ "sources": { "<key>": {...entry, "key": "<key>"} } }`, and a single entry for `/sources/{key}`.

### `GET /pages/{path}`
Pages are rendered from `content/pages/*.md` (YAML front matter + Markdown).
```json
{ "slug": "protection/respirators", "title": "...", "description": "...", "last_reviewed": "2026-11-07",
  "html": "<h2>...</h2>...", "toc": [{ "id": "...", "text": "...", "level": 2 }],
  "sources": { "<key>": {...} }, "image": { "url": "/images/pages/protection-respirators.jpg", "alt": "..." } }
```
Footnote markers `[^KEY]` in the Markdown render as `<sup><a href="#src-KEY">n</a></sup>`, numbered in order of first use. Go renders the HTML (goldmark); the SPA shows `html` as-is and renders the sources list from `sources`.

### `GET /health`
`{"status":"ok","species":51,"sources":155}`

## 4. Images

- `/images/species/{slug}.jpg` and `/images/pages/{slug-with-dashes}.jpg`, 1200×630. They are served from `IMAGES_DIR` (the repo's `images/` in dev; served directly by Apache in production).
- If a file is missing, `image.url` is `/images/placeholder.svg` (a neutral wood-grain SVG with no text, shipped in the repo) and `og:image` falls back to `/images/og-default.jpg` (or the placeholder SVG until that exists).

## 5. Evidence labels (both sides render the same text)

| Level | Label | Short |
|---|---|---|
| 1 | Confirmed in clinical testing | Confirmed |
| 2 | Listed by an agency | Agency-listed |
| 3 | Reported | Reported |
| 0 | Applies to all wood dust | General |

Badges always show the text, never color alone.

Hazard type display names: `respiratory_sensitizer` "Asthma / respiratory allergy", `respiratory_irritant` "Respiratory irritation", `skin_sensitizer` "Skin allergy", `skin_irritant` "Skin irritation", `eye_irritant` "Eye irritation", `hypersensitivity_pneumonitis` "Hypersensitivity pneumonitis", `systemic_toxic` "Whole-body (systemic) effects", `cardiac` "Heart effects", `carcinogen` "Cancer", `splinter_infection` "Splinter infection", `photosensitization` "Sun sensitivity (photosensitization)".

## 6. Submissions (anonymous)

### `POST /api/v1/submissions` (public, rate-limited)
Request:
```json
{ "kind": "new_species|correction|new_source|reaction_report",
  "target_slug": "oak",                    // required for correction/new_source/reaction_report when the species is listed
  "species_name": "Bubinga",               // new_species (or an unlisted species for reaction_report)
  "botanical_name": "Guibourtia spp.",     // optional
  "hazard_type": "skin_sensitizer",        // optional, enum from the species schema
  "details": "Plain-language description, own words",   // required, 20..4000 chars
  "citations": [ { "doi": "", "pmid": "", "url": "", "text": "" } ],  // ≥1 required for new_species/correction/new_source; each needs doi, pmid or url
  "email": "",                              // optional
  "form_token": "...",                      // from #initial data; signed, carries the issue time
  "website": ""                             // honeypot: must be empty
}
```
- Responses:
  - `201 { "id": 42, "reference": "DA-0042", "status_url": "/submit/status/<token>" }`.
  - `422 { "error": { "code": "invalid", "message": "...", "fields": { "details": "..." } } }`.
  - `429` when rate-limited, with a `Retry-After` header.
- A honeypot hit or a form submitted too fast returns a normal-looking `201` with a fake reference, and the submission is stored as `spam`. This avoids teaching bots.
- Validation: DOI pattern `^10\.\d{4,9}/\S+$`; PMID is digits only; URLs must be http(s). URLs on leads-only domains (wood-database.com, haz-map.com, forums) are accepted with a `warnings` array in the 201 response. The form shows those warnings beforehand too, via `GET /api/v1/submissions/check-url?url=`.
- `GET /api/v1/submissions/form-token` returns a fresh `{ "form_token": "..." }` for the SPA on client-side navigation.

### `GET /api/v1/submissions/status/{token}` (public, by unguessable token)
`{ "reference": "DA-0042", "kind": "...", "status": "new|triaging|needs_info|accepted|rejected|exported|merged", "created_at": "...", "updated_at": "...", "notes": [ { "at": "...", "text": "..." } ] }`. Only notes marked visible to the submitter are included. Spam shows as `new`.

## 7. Admin (session or API key)

Session auth comes from Portiko `rp` (owner only). Unsafe methods with a session require a matching `Origin` header (CSRF). API-key requests use `Authorization: Bearer da_…` and don't need `Origin`.

| Method and path | Scope | Notes |
|---|---|---|
| `GET /api/v1/submissions?status=&kind=&since=&limit=&cursor=` | `submissions:read` | Newest first. Includes `email` and the IP hash, never a raw IP |
| `GET /api/v1/submissions/{id}` | `submissions:read` | Full record, plus events |
| `PATCH /api/v1/submissions/{id}` | `submissions:write` | `{ "status"?, "note"?: { "text", "visible_to_submitter": bool } }` |
| `GET /api/v1/submissions/{id}/export` | `submissions:read` | Proposed `data/species/<slug>.json` patch (JSON) for accepted submissions |
| `GET /api/v1/stats` | `stats:read` | `{ "open": n, "by_status": {}, "by_kind": {}, "newest_at": "...", "new_last_24h": n, "spam_last_24h": n }` |
| `GET /api/v1/admin/api-keys` | session only | List: `name`, `prefix`, `scopes`, `created_at`, `last_used_at`, `expires_at`, `revoked_at` |
| `POST /api/v1/admin/api-keys` | session only | `{ name, scopes[], expires_at? }` returns `{ ..., "key": "da_<prefix>_<secret>" }`. The full key is shown **once** |
| `DELETE /api/v1/admin/api-keys/{prefix}` | session only | Revokes the key |
| `GET /api/v1/me` | session | `{ "user": { "display_name", "is_admin" } }`, or 401 |

Key format: `da_` + an 8-character prefix + `_` + 32 random bytes in base62. Only the SHA-256 of the full key is stored, compared in constant time.

`/auth/login`, `/auth/callback` and `/auth/logout` belong to Portiko `rp`. The embedded provider owns `/login`, `/signin`, `/u/`, `/oauth/`, `/accounts/` and `/.well-known/openid-configuration` (see PLAN §5). All of them are `noindex` and disallowed in robots.txt.
