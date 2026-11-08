# Verify list: batch `mahogany-teak`

Species: mahogany, african-mahogany, teak, iroko, sapele, meranti, afrormosia. Reviewed 2026-11-07.

All botanical names were checked against the WFO Plant List (list.worldfloraonline.org). POWO could not be used because its API returned a Cloudflare challenge. All PMIDs and DOIs in `data/sources/mahogany-teak.json` were confirmed on PubMed.

## ACGIH "mahogany" (A2): which genus?

- The OSHA page (osha.gov/chemicaldata/798, checked 2026-11-07) lists "TLV-A2 (birch, mahogany, teak, walnut)" with no botanical name.
- Following the handoff seed list, the A2 carcinogen hazard and `carcinogen_class` were applied to **mahogany (Swietenia)** only.
- On purpose, **african-mahogany (Khaya) has no ACGIH A2 entry.** Another agent is resolving this. If ACGIH turns out to include Khaya, add a level 2 `carcinogen` hazard and a `carcinogen_class` row to `african-mahogany.json`. If it means Khaya only, remove them from `mahogany.json`.
- HSE-WIS30 also lists "Mahogany" without a botanical name. It is cited on the Swietenia entry only.

## Primary papers not reachable (claims kept at level 3 via Schlunssen-2018)

- Kersten W, von Wahl P-G. Allergologie 1994;17:55-60. Cited by Schlunssen for occupational asthma (OA) from *Swietenia mahagoni*, sapele, iroko and meranti. Not in PubMed.
- Oertmann C, Bergmann KC. Allergologie 1993;16:334-340. Cited by Schlunssen for OA from *Khaya anthotheca*, teak and sapele. Not in PubMed.
- Vandenplas O et al. Thorax 1996;51:472-478 (PMID 8711673). Cited by Schlunssen for meranti (*Shorea pauciflora*) OA. The abstract says "wood dust" only, and the PMC scanned PDF was blocked, so the species is unconfirmed. Not cited.

## Papers with no abstract (content beyond the title unconfirmed)

- **Sosman-1969** (NEJM, PMID 4980984): the seed lead for mahogany asthma. The full text is paywalled (403). We could not confirm that it involves mahogany or what tests it used. It is defined but **not cited** on any hazard. Need: read the full text, then add it to the mahogany asthma hazard (level 1 if challenge-confirmed).
- **AlvarezCuesta-2004** (sapele): the title says asthma plus IgE-mediated contact dermatitis. If the paper used a specific inhalation challenge, prick test or sIgE, upgrade the sapele asthma and skin hazards to level 1. PubMed lists pages 88-98; check them.
- **Fowler-1985** (Honduran mahogany dermatitis): patch testing not confirmed.
- **Schmidt-1978** (teak contact urticaria with systemic effects): details unconfirmed.
- **Schmalle-1984** (deoxylapachol structure): cited for the compound identity based on its title. Schulz-1977 (abstract) supports that deoxylapachol from teak is a sensitizer.
- **Pickering-1972** (iroko and western red cedar asthma): cited as supporting only.
- **Hinnen-1995** and **Stingeni-1998** (iroko allergic contact dermatitis): patch testing not confirmed, so iroko dermatitis stays at level 2. These could be upgraded to level 1 after a full-text check.
- **Koh-2016** (meranti contact urticaria) and **Goransson-1980** (lauan urticaria and rhinoconjunctivitis): testing not confirmed, so both stay at level 3.

## Names and taxonomy

- **Meranti:** WFO now places *Shorea pauciflora* in *Rubroshorea* (Ashton & Heckenhauer, 2022). Many red meranti species moved there, and some lauan species moved to *Pentacme*. The entry keeps "Shorea spp." and lists Rubroshorea under synonyms. A reviewer should decide on the display botanical name.
- **Afrormosia:** *Pericopsis elata* (Harms) Meeuwen is accepted, and *Afrormosia elata* Harms is a synonym (WFO). The handoff's dagger mark is resolved.
- **Iroko:** *Milicia excelsa* is accepted, and *Chlorophora excelsa* is a synonym (WFO). *Milicia regia* is also traded as iroko but was not checked separately.
- **Khaya:** *K. ivorensis*, *K. anthotheca* and *K. senegalensis* are accepted. *K. grandifoliola* gave two candidate authorships, so the match was ambiguous (C.DC. is accepted).

## Other open items

- **Afrormosia:** the only source is HSE-WIS30. Its "nervous system effects" entry has no detail. The route (inhalation) was assumed. No primary reports were found in PubMed.
- **Descriptions:** the non-health facts in the descriptions (uses, regions, CITES listing for afrormosia, silica in teak) were written from general knowledge, with no Tier D source attached. A reviewer could add FPL tech-sheet sources.
- **HSE-WIS30:** checked against a third-party mirror of the PDF (oneida-air.com), because hse.gov.uk/pubns/wis30.pdf returns 404. The mirror's footer says it is licensed under the Open Government Licence.
- **Unused leads:**
  - Casas-2003 (iroko OA, PMID 12605819)
  - de las Marinas-2013 (iroko basophil activation, PMID 24654319)
  - Rao-2010 (teak ACD, PMID 20445306)
  - Estlander-1999 (teak and obeche ACD, PMID 10475519)
  - Harris-1989 (nail discoloration from mahogany, PMID 2521598)
