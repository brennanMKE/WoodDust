# Contributing to Dust Almanac

Thank you for helping. The site exists because the charts copied from forum to forum have no sources. Every claim here has to be traceable, and every word has to be our own.

There are two ways to contribute:

- **No account needed:** use the form at `/submit` on the site, or the "Suggest a correction" link on any species page. You get a reference number and a private status link.
- **Pull request:** edit `data/species/<slug>.json` (and `data/sources.json`) directly. CI checks the schema and that every source key exists.

## Rules for hazard claims

1. **Every new hazard needs a Tier A–C source** (see `wood-dust-site-handoff.md` §2):
   - **A:** agency or regulatory listings (IARC, ACGIH via OSHA, NIOSH, HSE, EU).
   - **B:** peer-reviewed reviews.
   - **C:** primary studies and case reports.

   Give a **DOI**, a **PubMed ID**, or an **official URL**.
2. **Leads-only sources are not accepted as citations.** They are useful for spotting a species we are missing. Then find the study or agency page they rely on and cite that. Leads-only sources include:
   - The Wood Database (`wood-database.com`)
   - Haz-Map (`haz-map.com`)
   - `scrollsawvillage.com`, `woodbin.com`, `wooduchoose.com`
   - Fine Woodworking (`finewoodworking.com`), AAW / American Woodturner (`woodturner.org`)
   - Forums and wikis: `reddit.com`, `lumberjocks.com`, `sawmillcreek.org`, `woodworkingtalk.com`, `woodnet.net`, `woodweb.com`, `ukworkshop.co.uk`, `wikipedia.org`

   The submission form warns when a URL is on one of these domains. The list lives in code (`LeadsOnlyDomains` in `internal/contrib/validate.go`). **Keep this list and that one in sync.**
3. **Write in your own words.** Do not paste abstracts, paper text, or anyone's chart, and do not copy potency or incidence ratings. Facts and citations are fine; someone else's wording, selection and arrangement are not (handoff §1).
4. **Evidence level** comes from the type of source, not from opinion:
   - 1: confirmed in clinical testing
   - 2: listed by an agency
   - 3: reported
   - 0: applies to all wood dust

   Say why the level fits.
5. **Botanical names** are checked against Plants of the World Online or World Flora Online.

## Pull request checklist

- [ ] Every hazard cites at least one Tier A–C source key that exists in `data/sources.json`.
- [ ] New sources have a DOI, PMID or official URL, and `verified` is true only if you read the source.
- [ ] Summaries and descriptions are in my own words.
- [ ] The evidence level is justified by the source type.
- [ ] The botanical name is checked against POWO or WFO.
- [ ] `scripts/check.sh` passes (`go run ./cmd/wooddust seed -check` at minimum).

## Reaction reports

"I reacted to this wood" reports are welcome through the form. They are leads for research only: they are never published and never become a hazard claim on their own. **This is not a medical service.** If you are unwell, see a doctor.

## What the submission form keeps

- Your suggestion.
- An optional email, used only for follow-up questions. It is deleted when the suggestion is closed.
- A keyed hash of your IP address under a salt that changes daily, used only to stop abuse. The raw IP is never stored.

## Code

- Go server: `cmd/`, `internal/`.
- Svelte front end: `web/`.

See `docs/architecture.md` and `docs/contract.md`. Run `scripts/check.sh` before opening a pull request.
