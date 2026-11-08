# Verify: softwoods batch

Species: spruce, fir, douglas-fir, western-hemlock, cedar-of-lebanon, yew, larch. Researched 2026-11-07.

## Citations not reachable (not cited)

- **Kersten W, von Wahl P-G. 1994.** Allergische Atemwegserkrankungen in der holzverarbeitenden Industrie. *Allergologie* 17:55-60. Schlunssen-2018 cites it for fir and spruce occupational asthma. It is not in PubMed, so we have no DOI and have not read it. The fir respiratory hazard therefore rests on `Schlunssen-2018` at level 3.
- **Oertmann C, Bergmann KC. 1993.** Atemwegserkrankungen bei Arbeitern im Holzgewerbe. *Allergologie* 16:334-340. Cited for spruce. Same problem: not in PubMed and not read.
- **Hanslian L, Kadlec K. 1964.** Alergická onemocnění vyvolaná modřínovým dřevem [Allergic diseases caused by larch wood]. *Prac Lek* 16:362-6. PMID 14251055. The PubMed record exists, but it has no abstract and the article is in Czech. We don't know the route or what testing was done, so it is not cited. Worth getting the full text.
- **Carosso et al.** Schlunssen-2018 mentions this study as including a Douglas fir extract. We could not identify it in PubMed by author search.

## Claims with weak or indirect support

- **Spruce skin_sensitizer at level 1.** Estlander-2001 patch-tested patients exposed to "pine or spruce" dust and does not separate the two species. The Majamaa-2004 abstract is missing, so its testing is unconfirmed. A reviewer may prefer level 3.
- **Kespohl-2012 is a single, polysensitized patient.** The paper's sIgE results cover spruce, pine and larch. We kept larch at level 3 because spruce was the dominant allergen. The abstract does not mention fir, although Schlunssen-2018 cites this paper for fir.
- **Yew cardiac toxicity by ingestion is set to level 3.** The evidence is strong (case series with serum taxines, ECG changes and deaths), but our level-1 criteria cover only sensitization tests. Consider adding a label for "confirmed toxicology". Seed toxicity is commonly stated, but we read no primary seed paper.
- **Some hazards rest on HSE-WIS30 alone.** We found no primary report for:
  - yew systemic, cardiac and dermatitis effects from dust
  - Douglas fir dermatitis, septic splinters and rhinitis
  - western hemlock rhinitis and bronchial effects
  - spruce photosensitization
  - larch urticaria
- **Greenberg-1972 (cedar of Lebanon).** The PMID and DOI are verified, but the paper has no abstract and we have not read it.
- **HSE-WIS30 dermatitis entries** don't say whether the reaction is irritant or allergic. We coded them as `skin_irritant`.

## Notes

- **HSE-WIS30.** The hse.gov.uk/pubns/wis30.pdf URL returns 404. We read a third-party mirror instead (oneida-air.com). Its footer says the sheet "contains public sector information ... licensed under the Open Government Licence". The agency agent should confirm the live HSE URL.
- **Botanical names** were checked against the WFO matching API (POWO blocked automated access). All seven are accepted:
  - *Picea*
  - *Abies*
  - *Pseudotsuga menziesii* (Mirb.) Franco
  - *Tsuga heterophylla* (Raf.) Sarg.
  - *Cedrus libani* A.Rich.
  - *Taxus baccata* L.
  - *Larix*
- **Possible duplicate source definitions.** `Kespohl-2012`, `Hessel-1995` and `Estlander-2001` may also be defined by the pine agent. If so, they are the same papers and need deduplicating at merge.
- **ACGIH A4 rows.** We added no ACGIH A4 `carcinogen_class` rows. That is left to the agency agent to apply site-wide.
