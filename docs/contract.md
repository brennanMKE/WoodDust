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
- Site name comes from config (`SITE_NAME`, default `Wood Dust`). Go exposes it to the SPA in `#initial` as `site: { name, base_url }`.

## 2. Routes

| Path | Route id | Initial `data` = |
|---|---|---|
| `/` | `home` | `GET /api/v1/species?featured=true` |
| `/wood/{slug}` | `species` | `GET /api/v1/species/{slug}` |
| `/browse` | `browse` | `GET /api/v1/species` |
| `/risks`, `/protection`, `/protection/{sub}`, `/symptoms`, `/limits`, `/glossary`, `/methodology` | `page` | `GET /api/v1/pages/{path}` |
| `/sources` | `sources` | `GET /api/v1/sources` |
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
