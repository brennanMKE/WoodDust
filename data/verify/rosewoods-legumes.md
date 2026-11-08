# Verify: rosewoods-legumes batch

Species: cocobolo, brazilian-rosewood, rosewood, zebrawood, padauk, kejaat, wenge. Reviewed 2026-11-07.

## Access limits this session

- POWO API was blocked by a Cloudflare challenge, so all botanical names were checked against the World Flora Online matching API instead. All 7 resolved to accepted names (WFO IDs are in each file). Re-check on POWO if POWO is the preferred authority.
- The USDA FPL site returned "Service unavailable". The cocobolo tech sheet was read from the Wayback snapshot dated 2026-01-23 of the official URL.
- HSE WIS30 is no longer at hse.gov.uk/pubns/wis30.pdf (404). The rev1 (11/12) PDF was read from the Wayback snapshot dated 2013-02-13. The shared `HSE-WIS30` key owner should update the URL.

## Citations whose method could not be confirmed (no abstract, full text not read)

| Key | Issue | Effect on data |
|---|---|---|
| Eaton-1973 | No abstract. The species (cocobolo) and the outcomes (rhinitis/asthma) come from the Schlunssen-2018 table. | Cocobolo respiratory hazard set to level 3. Could rise to level 1 if the full text shows challenge, skin or sIgE testing. |
| Ordman-1949 | PMID 15397784 confirmed. It has no DOI and no abstract. | Kejaat asthma set to level 3. |
| Doyen-2023 | No abstract on PubMed. SIC confirmation comes from the UCLouvain repository record and search summaries. The paper's botanical name for padauk was not seen. | Padauk asthma set to level 1. Confirm SIC and species in the full text. |
| Fisher-1981, Dias-1992, OrtizBrugues-2016 | These are D. nigra dermatitis letters, probably with patch tests, but the method is unconfirmed. | Brazilian rosewood skin hazard set to level 3. |
| KiecSwierczynska-2004 | The title names P. soyauxii. There is no abstract. | Used together with Rackett-1997 for the level 1 padauk skin hazard. |
| Piletta-1996, Gallo-1996, Athavale-2003 | D. latifolia case reports. Only the titles were read. | These are supporting sources in the rosewood skin hazard. The level 1 rests on Pfohler-2018 and Gamboa-2024. |
| Schulz-1962 | German, no abstract. | Only the title-level claim is used: a quinone sensitizer in D. nigra and D. retusa. |
| OrtizBrugues-2016 | PubMed spells a co-author "Rustenmeyer T". This is probably Rustemeyer. | The citation uses Rustemeyer. Check it. |

## Key naming

- The handoff placeholder `Doyen-padauk` is defined as `Doyen-2023` (J Allergy Clin Immunol Pract 2023;11(10):3240-3241.e1, PMID 37352932).
- `Rackett-1997` (cocobolo, padauk, pao ferro, macassar ebony, ziricote) may also be wanted by other batches, so check for duplicate definitions when merging.

## Botanical name / scope questions

- Zebrawood: Bush-1978 identifies the wood only to the genus Microberlinia. M. bisulcata is also sold as zebrawood. The "zebrano" name and the M. bisulcata note are not sourced.
- Padauk: HSE-WIS30 says effects are "species-dependant" and gives no botanical name. Mehta 2007 (PMID 17343633) concerns P. dalbergioides (Andaman padauk), so it is only mentioned in related_species_note.
- Cocobolo: HSE-WIS30 lists "Rosewood (many species)" without naming cocobolo, so it is not cited on the cocobolo page. Decide whether it should be.
- These common names have no Tier D source: kejaat aliases (kiaat, mukwa, bloodwood), padauk aliases (barwood, camwood), Brazilian rosewood aliases (Rio rosewood, jacaranda), and the wenge, zebrawood and kejaat regions and descriptions. The cocobolo names are from the FPL sheet.

## Leads not added (need full review)

- Yang 2015, Int J Dermatol (PMID 25944249): occupational toxic epidermal necrolysis linked to Dalbergia cochinchinensis.
- Li 2018, JAAD Case Rep (PMID 29693067): carpenter's pigmentation from African padauk dust.
- Jensen 2016 (PMID 26251338) and Leider 1950 (PMID 15426324): cocobolo dermatitis.
- Hausen 1982 (PMID 7160307), Haustein 1982 (PMID 6461497) and Vihervaara 2019 (PMID 30957233, African blackwood): further Dalbergia contact allergy reports.
- Wenge: no peer-reviewed health reports were found on PubMed. All wenge hazards rest on HSE-WIS30 alone.
