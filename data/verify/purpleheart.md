# Verify: purpleheart

Species: purpleheart (`peltogyne-spp`). Reviewed 2026-11-08.

Purpleheart has very little specific health evidence. The card carries the general level-0 cancer entry, the ACGIH TLV and A4 class shared by unlisted hardwoods, and one level-3 skin report. Absence of other reports is not evidence of safety.

## Access limits this session

- POWO was blocked by a Cloudflare challenge (HTTP 403). All names were checked on World Flora Online (matching API plus GraphQL for accepted status and family). Re-check on POWO if POWO is the preferred authority.
- The USDA FPL site returned HTTP 502. The Peltogyne tech sheet (`peltog1new.html`) was read from the Wayback capture of the official URL dated 2023-06-01.
- The Wiley full text of Mehta-2007 returned HTTP 403 (closed access).

## Citations whose method could not be confirmed

| Key | Issue | Effect on data |
|---|---|---|
| Mehta-2007 | PMID 17343633 and DOI 10.1111/j.1600-0536.2007.00981.x were confirmed on PubMed and Crossref. It is a one-page letter with no abstract, and the full text was not read. Two pointers suggest positive patch tests to purpleheart: NLM MeSH lists "Patch Tests" and "Music", and a Semantic Scholar machine-generated summary says so. Neither is a human reading. The paper's botanical name for purpleheart is unknown, and so is the exposure route (dust, or contact with an instrument). | Skin hazard set to **level 3**, with no `source_part`. Upgrade to level 1 if the letter shows a positive patch test to purpleheart. Add `dust` or `heartwood` to `source_part` once the exposure is known. |

## Searches with no usable result

- PubMed: "Peltogyne" (11 hits, none on health), "purpleheart" (5; only Mehta-2007 is relevant, the rest are about purple-heart radish), "purple heart wood", "amaranth wood dermatitis", "violetwood", "Peltogyne asthma" (0).
- Europe PMC full text: "purpleheart"; Peltogyne AND dermatitis/asthma/allergy/patch; "purple heart" AND wood AND dermatitis/asthma. None of the hits is a health report on the wood.
- HSE-WIS30: Table 2 (`data/wis30-facts.json`) has no purpleheart, Peltogyne or amaranth row, so there is no level-2 entry.
- Schlunssen-2018 and Baatjies-2023 (PMC full text) and the SENSOR-2017 PDF: no mention of purpleheart or Peltogyne.
- Leads seen but not cited: a Wikipedia mirror claims purpleheart dust causes irritation and nausea, perhaps from dalbergione-type compounds. No primary source was found, so nothing was added.
- Papers that cite Mehta-2007 and might describe it: Podjasek 2011 (PMID 21504691), Swinnen 2013 (PMID 23343440), Crepy 2015 (PMID 25905552) and Chomiczewska-Skora 2013 (PMID 23650772, Polish). Their full texts were not read. Podjasek's abstract does not say which woods its three patients reacted to.

## Botanical names

- WFO: genus *Peltogyne* Vogel (wfo-4000028510) is accepted, family Fabaceae.
- Accepted species: *P. paniculata* Benth. (wfo-0000174172), *P. venosa* (Vahl) Benth. (wfo-0000186735), *P. purpurea* Pittier (wfo-0000168366) and *P. mexicana* Martinez (wfo-0000168364).
- *P. pubescens* Benth. (wfo-0000171495) is a **synonym** of *P. paniculata* on WFO. It is listed in `synonyms`, not as a separate species.
- Also seen on WFO but not used: *P. densiflora*, *P. porphyrocardia*, *P. recifensis* and *P. confertiflora*.
- Not verified: which species make up most of the commercial purpleheart trade. The FPL sheet is genus-level only. The description says "species such as", not "the main traded species".

## Common names

- Source: FPL-Peltogyne, which gives purpleheart, amaranth, palo morado, morado, tananeo, koroboreli, purperhart, amarante, pau roxo, guarabu and violetwood. "Purple heart" (two words) is kept as a search spelling.
- "Morado" was left out because it is also used for Santos rosewood or pau ferro (*Machaerium scleroxylon*). That is a lead and was not checked here. "Tananeo" was left out as obscure.
- "Amaranth" also names the unrelated food plant and grain *Amaranthus*. No species in this dataset uses it, so no alias collision.
