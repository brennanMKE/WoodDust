# Shop posters

Printable posters for wood shops. Each is a full-page Paul and Babe illustration with one slogan and a QR code that is part of the scene. The art carries the sheet; the QR code takes people to the site. There is no printed URL, no checklist and no form-like panel.

Decided 2026-11-08 (proposal v2, reviewed by Brennan). The WoodDust side is built (see "What's built"); the art awaits approval and ShortLinks setup is still to do.

## Decisions

| Question | Decision |
|---|---|
| First two posters | **Shave** and **Sunbeam** |
| "Posted by" plaque | In v1, so the builder and the ShortLinks key ship with it |
| Resolution | Upscale the 1024 × 1536 art 2× and test-print. Tabloid/A3 may be softer |
| Opt-in shop name | Offered, off by default |
| Click IPs in ShortLinks | Hashed before launch ([ShortLinks #0121](../../ShortLinks/issues/0121.md)) |
| Emblem seal | On the Almanac poster only |
| Sixth poster (fire) | Not now |

## The set

| id | Slogan | Mood | QR code is on | Scan opens | Words from |
|---|---|---|---|---|---|
| `shave` | Paul shaves for a reason. | funny | a card tucked into the shaving mirror | `/protection/respirators` | `protection-respirators.md` (HSE-FitTesting, HSE-WIS23) |
| `sunbeam` | You can't see what you're breathing. | quiet, the one dark sheet | the end grain of a log in the beam of light | `/risks` | `risks.md` (HSE-WIS23) |
| `tall-tale` | Catch it at the tool. | heroic | a crate label on Babe's wagon | `/protection/dust-collection` | `protection.md`, `shop-poster.md` (HSE-WIS23) |
| `sweep` | No brooms. | funny | a tag on Babe's halter | `/protection/cleanup` | `protection-cleanup.md` (HSE-WoodDust) |
| `almanac` | No wood is "safe." Know yours. | warm | a bookplate in the open almanac, plus the emblem seal | `/` | `risks.md` (HSE-WIS30) |

Each may carry one small second line, near-verbatim from the same page. Every word on a poster must already be on the site with a source.

Art prompts: `images/prompts/codex-art.md` §D. The files go in `images/art/posters/<id>.png` and stay unpublished until approved on a printed proof.

## Print files

- **PDF (default):** full-bleed portrait. Letter and A4 come from one layout with a 6 mm safe inset. Tabloid/A3 versions have a 3 mm bleed and crop marks.
- **Also:** a 300 dpi PNG, a greyscale version for low-toner shop printers, and the SVG source.
- **Rendering:** hand-made SVG templates with the art inlined, rendered by `rsvg-convert` at build time. Custom posters are filled in and exported in the browser.
- **Type:** Alfa Slab One for the slogan, Source Sans 3 for the small line and plaque.
- **QR code:** 35–42 mm, pure black on white or pale cream, square to the sheet, with its quiet zone.

## Tracking

Each QR code opens a go.dustalmanac.com link, a ShortLinks instance of its own on photon. The links sit in one campaign, `posters`:

```
utm_source=flyer&utm_medium=print&utm_campaign=poster-<id>&utm_content=stock-v1   (stock)
                                                       …&utm_content=custom     (plaque)
```

A poster with a plaque gets its own random-code link, minted on download. The shop name stays in the browser unless the off-by-default opt-in is ticked; then it's stored only as the ShortLinks link's `placement`, never in a URL.

Abuse limits match the submissions form. Without the key, the download falls back to the stock link.

## What's built (2026-11-08, uncommitted until reviewed)

| Piece | Where | State |
|---|---|---|
| Catalog of the five posters, slogans, destinations, stock aliases | `internal/posters/catalog.go` | Shave and Sunbeam published; the other three hidden |
| Stock QR codes (`go.dustalmanac.com/u/<alias>`, aliases `shave`, `sunbeam`, `talltale`, `sweep`, `almanac`: ShortLinks aliases allow no hyphens) | `posters/qr/<id>.svg`, from `go run ./cmd/posterbuild qr`; embedded in the binary | Built. The aliases still have to be created in ShortLinks (campaign `posters`, `utm_content=stock-v1`) |
| SVG templates, presentation attributes only, slots marked `data-slot` | `posters/src/shave.svg`, `posters/src/sunbeam.svg` | QR and slogan placed on the round-4 art. The QR is ~32 mm on Shave (the card won't hold more) and ~34.5 mm on Sunbeam: check on the printed proof |
| Print files: Letter, A4, greyscale PDFs, 300 dpi PNG, SVG source | `scripts/posters.sh` (`make posters`) → `images/posters/`; `--proof` writes `tmp/posters/` and allows missing art | Works with `rsvg-convert`. Not generated into `images/posters/` until the art is approved. The 2:3 art fills the sheet's height; the sides take the art's paper colour |
| Fonts | `web/public/fonts/` (Alfa Slab One, Source Sans 3 variable, OFL texts); the script points fontconfig at them | Self-hosted |
| `/posters` page, plaque builder, PNG export (canvas, 2048 × 3072), print route `/posters/{id}/print` for PDF | `web/src/views/Posters.svelte`, `components/PosterCard.svelte`, `views/PosterPrint.svelte`, `lib/poster.ts` | Built. Missing art shows the template's placeholder block. Linked from the home page band and the footer |
| Link endpoint `POST /api/v1/posters/links`, with the opt-in and the stock fallback | `internal/posters/links.go` | Built; falls back to stock until the key is set |
| ShortLinks client (v0.4.0 tokens, 5 s, no retries, token redacted) | `internal/posters/client.go` | Built against the documented API; tested with a fake |
| Settings: `settings` + `settings_audit` (migration 000003), `SETTINGS_KEY`, `wooddust settings`, `scripts/set-setting.sh`, `/admin/settings`, dashboard banner | `internal/settings`, `internal/posters/admin.go`, `web/src/admin/Settings.svelte` | Built. Not run against photon |

Still to do: approve the art (then `make posters` and commit the print files), create the go.dustalmanac.com instance and register its port (then set `shortlinks_api_url` if it isn't `127.0.0.1:8084`), create the `posters` campaign and the five stock aliases, mint the tokens and store them with `scripts/set-setting.sh`. The optional second line under each slogan is left out until its wording is picked from the cited page.

## ShortLinks prerequisites

ShortLinks **v0.4.0** (on GitHub `main`) already has both of these:
- **Instance host:** short URLs and QR codes are built on the instance's own `BASE_URL` (upstream #0117).
- **API tokens:** named API tokens, managed under Account › API tokens. A token acts as its owner on `/api/*`, or GET-only if it's read-only. There are no per-route scopes, so the go.dustalmanac.com instance holds only poster links and only Brennan's account.

Still to build before launch: click-IP hashing ([ShortLinks #0121](../../ShortLinks/issues/0121.md)).

## The key in admin Settings

1. Brennan signs in on go.dustalmanac.com through "Recover account". Registration stays closed.
2. He creates a token there (Account › API tokens), not read-only, with a 12-month expiry. Prod and pre-prod get separate tokens.
3. `scripts/set-setting.sh <env> shortlinks_api_key` reads the token hidden and sends it over ssh stdin to `wooddust settings set --stdin`.
4. It's stored AES-256-GCM in a new `settings` table, under a `SETTINGS_KEY` that `provision.sh` generates on the host.
5. `/admin/settings` shows the prefix and status, with Test connection, Replace and Clear.

## Phases

0. **ShortLinks:** deploy v0.4.0 as the new go.dustalmanac.com instance. Click-IP hashing (#0121) lands before the posters launch.
1. **Art, then Shave and Sunbeam:** Codex round, proofs, 2× upscale, templates, print files, the `/posters` page, the go.dustalmanac.com instance, and settings with the admin screen.
2. **Plaque:** the link endpoint, export in the browser, the opt-in and the fallback.
3. **The rest:** Tall Tale, Sweep and Almanac, plus the Tabloid/A3 print-shop files.
