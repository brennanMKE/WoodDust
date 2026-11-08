# web/: the Svelte SPA

Vite + Svelte 5 (runes) + TypeScript, without SvelteKit. The build goes to `dist/`, and the Go server embeds it (`embed.go`). `docs/contract.md` is the binding interface between this app and Go.

## Commands

```sh
npm install
npm run dev            # Vite on :5173; proxies /api, /images, /sitemap.xml to Go on 127.0.0.1:$PORT (default 8080)
npm run dev:fixtures   # same, but no Go needed (see below)
npm run check          # tsc for the config files + svelte-check (expect 0 errors, 0 warnings)
npm test               # vitest: router, combobox, labels, DOI/PMID validators, submission reducer, API-key display-once
npm run build          # writes dist/index.html + dist/assets/*
```

## Fixture mode

`VITE_FIXTURES=1` (or `npm run dev:fixtures`) makes the dev server answer `/api/v1/*` and `/images/*` itself, using `fixtures/server.ts`:

- species, sources and the regulatory summary come from `../data/`
- pages come from `../content/pages/*.md`, rendered with a very small Markdown subset that supports headings, lists, links, emphasis and `[^KEY]` footnotes. A missing page gets a placeholder
- search is approximate: exact match, then prefix, then trigram similarity
- `/images/placeholder.svg` is generated, and any other image is served from `../images/` if it exists
- submissions, admin and API keys (contract §6-§7) use an in-memory store seeded with a few examples. `/api/v1/me` returns a fake owner. `/auth/logout` signs out, so `/admin` then goes to `/auth/login`, which signs straight back in. POSTs are rate-limited to 5 per 10 minutes; details containing `ratelimit` force a 429

The fixtures are only loaded from `vite.config.ts` during `vite serve`. Nothing under `src/` imports them, so they are not in the production bundle. Go remains the reference for API behavior.

## How it boots

`index.html` contains the three markers that Go fills on every page response: `<!--ssr-head-->`, `<!--ssr-body-->` inside `#app`, and `<!--ssr-data-->`. When the app boots, `src/main.ts`:

1. reads `#initial` (`{route, data, site}`)
2. uses `data` as is if `route` matches the current URL. It never refetches the initial route
3. clears `#app` and mounts the app

Without `#initial` (Vite dev), it fetches the route's API data instead.

Navigation is handled by `src/lib/nav.svelte.ts` with the History API. It intercepts same-origin links to known routes (`src/lib/router.ts`), fetches `/api/v1/*`, and updates the title, meta description, canonical link and robots (`noindex` for `/?q=` and `/submit`). It then moves focus to the new page's `<h1>` and announces the page change.

## Layout

```
src/
  main.ts, App.svelte, app.css     entry, shell, design tokens (light/dark, print)
  lib/      router, nav state, api, head/meta, labels (contract §5), combobox logic, footnotes,
            citations (DOI/PMID/URL checks), submission (form reducer, status timeline)
  components/  Header, Footer, SearchBox (ARIA combobox), EvidenceBadge, SpeciesTile,
               SpeciesImage, SourceList, Art (illustration slot with fallback)
  views/    Home, Species, Browse, Page, Sources, Submit (anonymous form), SubmissionStatus, NotFound
  admin/    owner-only screens (dashboard, submissions, detail/export, API keys); a lazily loaded chunk
fixtures/   dev-only fixture API
```

## Submissions and admin

- `/submit` and `/submit/{slug}` are an anonymous form (contract §6). Its rules live in `src/lib/submission.ts`, a pure reducer. `form_token` comes from `#initial`; on client navigation `nav` fetches `/api/v1/submissions/form-token` and the target species. `SearchBox` with `mode="pick"` is the species picker.
- `/admin/*` loads `src/admin/Admin.svelte` with a dynamic `import()`, so it builds as its own `Admin-*.js` chunk. Every admin route's data is `GET /api/v1/me`, and a 401 goes to `/auth/login?return=…`. Requests use `credentials: 'same-origin'`; the browser adds `Origin` for the CSRF check.

## Illustrations

The files, sizes and alt text are listed in `images/prompts/site-art.md`. Every slot copes with a missing file:

- **Optional art** hides itself: the home hero, the no-results art, the spot illustrations and the page headers.
- **The 404 art** falls back to a faint tree-ring drawing of the same size.
- **The header emblem** falls back to an inline Babe-blue ring mark.
- **The forest border** is a CSS background, so nothing shows when the file is missing.
- **The favicon** switches to `images/art/favicon.png` only after that file loads.

## Printing

The species card prints as a one-page shop card with a print stylesheet. It drops the photo, the description, and the general hazards that apply to all wood dust, and it sets the sources in small type across three columns. 49 of the 51 species fit on one Letter or A4 page; pine (8 hazards) runs onto a second page.
