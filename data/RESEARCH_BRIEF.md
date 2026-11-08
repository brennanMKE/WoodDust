# Research brief for species data agents

Read `wood-dust-site-handoff.md` sections 1-5 first. That file is the authority on data policy. This brief adds the mechanics.

## Output files (only write your own files; other agents run in parallel)

- `data/species/<slug>.json` — one per species, must validate against `data/schema/species.schema.json`.
- `data/sources.json` — the single source registry (batch files were merged into it). New Tier C and D sources you introduce go here, keyed by source key, validating against `data/schema/sources.schema.json`.
- `data/verify/<batch-name>.md` — every item you could not verify (citation, DOI, botanical name, claim). Never fill gaps from memory.

## Shared source keys (defined by the `agency` agent — reference, do not define)

`IARC-62`, `IARC-100C`, `OSHA-CSI`, `HSE-WIS30`, `EU-CMRD`, `HSE-EH40`, `NIOSH-REL`, `Schlunssen-2018`, `SENSOR-2017`, `Baatjies-2023`, `Goldsmith-1988`, `POWO`, `WFO`.

You may cite those keys freely. Everything else you cite, you define in your batch sources file. Key format: `FirstAuthor-Year` (ASCII, no diacritics, e.g. `GodnicCvar-1990`); add a suffix letter if a collision is possible (`Kespohl-2010a`). For USDA FPL tech sheets use `FPL-<genus>` (e.g. `FPL-Dalbergia-retusa`).

## Research method

1. Verify the botanical name on POWO (powo.science.kew.org) or WFO. Set `botanical_name_verified` and `botanical_name_source`. Put old names in `synonyms`.
2. For each hazard lead in the handoff seed list, find the primary paper. Confirm its PMID/DOI on PubMed (pubmed.ncbi.nlm.nih.gov) or Crossref. Read the abstract to confirm the paper actually supports the claim and what testing was done (specific inhalation challenge, patch test, sIgE → evidence level 1; case report without confirmation → 3).
3. Search PubMed for additional primary reports on the species ("<common name> wood asthma", "<genus> contact dermatitis", "<genus> occupational") — add well-supported ones.
4. HSE-WIS30 mentions → evidence level 2 (agency listing). ACGIH carcinogen classes via OSHA-CSI → level 2 `carcinogen` hazard. Never mark level 2 for a mention in a secondary review (Schlunssen-2018 etc. are Tier B: use them to find the primary paper; cite the primary paper; cite Tier B only if no primary is reachable and mark evidence level 3).
5. Do NOT use The Wood Database, Haz-Map, woodworking forums, or magazine charts as sources. Leads only.
6. Do NOT paste abstract text. Summaries are your own plain-language words, 1-2 sentences, calm tone.

## Every species file must include

- A `carcinogen` hazard at evidence level 0, sources `["IARC-62"]`, noting all wood dust is classified as carcinogenic to humans (nasal/sinus cancer). Plus a level 2 carcinogen entry if ACGIH names the species/genus specifically (oak, beech = A1; birch, mahogany, teak, walnut = A2) with source `OSHA-CSI` and a matching `carcinogen_class` row.
- `exposure_limits` with the ACGIH TLV (1 mg/m³ inhalable; western red cedar 0.5) sourced to `OSHA-CSI`. Do not add EU/UK values — the agency agent owns those and they will be applied site-wide.
- `description`: 1-3 sentences on what the wood is and how woodworkers use it (own words).
- `image_prompt` and `image_alt` (see below).
- `last_reviewed`: `2026-11-07`.
- `notes_internal`: anything a reviewer should know (conflicts, weak evidence, unresolved names).

## Image prompts

Style (user-chosen): photoreal close-up of a planed board, consistent across all species. Template — adapt the bracketed parts with accurate, species-specific visual detail (heartwood/sapwood color, grain pattern, figure, pore structure, luster, color change with age):

> Photorealistic studio photograph of a freshly planed [species common name] ([botanical name]) board, shot at a low three-quarter angle so the face grain fills most of the frame. [Specific color description of heartwood and any sapwood]. [Grain pattern, figure, ray fleck, pore texture]. Soft diffused daylight from upper left, gentle falloff, neutral warm-grey seamless background, shallow depth of field with the near edge tack-sharp. No text, no logos, no hands, no tools, no finish or varnish, natural unfinished wood. Wide 1200x630 composition (1.91:1) with the board centered and margin on both sides for cropping.

`image_alt`: one plain sentence describing what the image shows (e.g. "Planed board of black walnut with dark chocolate-brown heartwood and a stripe of pale sapwood.").

## Featured species

Homepage top 5, set `featured_rank`: 1 oak, 2 maple, 3 walnut, 4 cherry, 5 pine.

## Genus-level entries

Where the seed list says `spp.` make one entry for the genus (e.g. slug `oak`, id `quercus-spp`, botanical_name `Quercus spp.`), list the main woodworking species in `common_names`/`description` (e.g. red oak, white oak, English oak), and say in each hazard summary which species the evidence came from when it is narrower than the genus.

## Validate before finishing

Run `npx --yes ajv-cli@5 validate --spec=draft2020 -c ajv-formats -s data/schema/species.schema.json -d "data/species/<your-slugs>.json"` (and the same for your sources file with sources.schema.json). Fix all errors. If ajv-cli does not work, validate with a short Python script using `jsonschema` if available, or at least `python3 -m json.tool`.

Also check: every source key in your species files exists either in your batch sources file or in the shared list above.

## Final report (keep it short)

Species written, count of hazards by evidence level, sources defined (verified vs not), and the top unresolved items.
