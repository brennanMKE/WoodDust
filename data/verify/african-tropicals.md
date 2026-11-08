# Verification log: african-tropicals batch

Researched 2026-11-07. Species: ebony, obeche, ramin, makore, limba, okoume, mansonia, opepe.

## Botanical names

All eight names were checked as **accepted** on World Flora Online (WFO GraphQL API). POWO could not be reached (Cloudflare challenge on powo.science.kew.org); a POWO cross-check is still open.

- Mansonia altissima: WFO also accepts var. altissima and var. kamerunia. The trade timber is not tied to one variety.
- Tieghemella heckelii: WFO has two accepted records under different authorities ("(A.Chev.) Pierre ex Dubard" wfo-0000325065, and "Pierre ex A.Chev." wfo-0001236488). This looks like a WFO duplicate; the first one was used.
- Common and trade names not checked against a source: opepe (bilinga, badi, kusia); mansonia (bete); limba (frake, korina); ebony (Gaboon ebony, Cameroon ebony); makore (cherry mahogany). Check these against BS EN 13556 or USDA FPL.

## Makore name conflict (Obata-2000 vs SENSOR-2017)

- SENSOR-2017 maps Obata-2000's "makore" to *Prunus africana*.
- I read the full Obata-2000 text (the open J-STAGE PDF). It **never gives a binomial**. The wood is called "African cherry (Makore)" throughout.
- Its background reference 2 is Hausen B, "Tieghemella heckelli, Makore", in *Woods Injurious to Human Health* (1981), pp 89-91. Reference 6 is Sandermann & Barghoorn 1955, "Makore- und Peroba-Holz". The authors relied on these for the wood's uses and its reported dermatitis.
- **Conclusion:** the tested wood was *Tieghemella heckelii* (makore). The authors tie it to *T. heckelii* through the literature they cite, and "makore" is the trade name for *T. heckelii*. SENSOR's mapping probably came from the common name "African cherry", which *Prunus africana* shares. One caveat: the paper identifies the sample by trade name only and gives no botanical confirmation.

## Primary papers not reachable (cited Tier B instead, or not cited)

- Limba asthma: Oertmann C, Bergmann KC, Allergologie 1993;16:334-340; Kersten W, von Wahl P-G, Allergologie 1994;17:55-60; Wirtz C et al., "Asthmaanfalle durch Abachi- und Limbaholz", Arbeitsmed Sozialmed Umweltmed 1997;32:257-260. None of these is in PubMed or Crossref. They are cited via Schlunssen-2018 at level 3. They need a library copy.
- Okoume (gaboon) asthma: Kersten 1994 (same paper as above). Not read.
- Mansonia: Nava C et al., Med Lav 1975;66:574-576 (PMID 131905, "Mansonia altissima disease", 30 cases); Sigg RH, Med Klin 1968 (PMID 4387523, periarteritis nodosa); Reinl W, Zentralbl Arbeitsmed 1965 (PMID 14305268). None has an abstract and none is cited. The 30-case series may support raising skin and respiratory hazards to level 3 or level 1.
- Mansonia cardiac: Beretta-1970 (an animal study) was checked by title only. Guede-1977 used a bark extract. No human cardiac case report was found, so the human cardiac claim rests on HSE-WIS30 alone.
- Opepe: no primary health report found in PubMed or Europe PMC.
- Makore CNS and blood effects (HSE): no primary source found.

## Cited with title only (no abstract)

- Kanerva-1998 (obeche contact urticaria), Beck-1982 and Bruynzeel-1987 (ramin sensitivity), Noferi-1966 (mansonia rhinopathy), Beretta-1970, MariniBettolo-1966. The citation metadata is verified, but the content details are not. Before raising any of these hazards to a higher level, check whether the ramin papers used patch testing.

## Found but not cited (need full text)

- Ebony skin: Wlodek C, Lovell CR, Contact Dermatitis 2014;71:381-2 (doi:10.1111/cod.12299); Kuner N, Jappe U, Contact Dermatitis 2004;50:258-9 (violinist, ebony patch test?); Rouyer N et al., Contact Dermatitis 2026;95:339-340 (doi:10.1111/cod.70182, airborne contact allergy with photo-aggravation). The *Diospyros* species and the test results could not be confirmed. If confirmed, ebony skin_sensitizer could move to level 1, and a photosensitization hazard could be added.
- Obeche, additional level 1 support: Quirce S et al., J Allergy Clin Immunol 2000;106:400-1 (PMID 10932087); Krawczyk-Szulc P et al., Int J Occup Med Environ Health 2014 (PMID 24840887, positive SIC); Matsumoto Y et al., 2002 (PMID 12166261, ayous, Japanese); Pontier JP et al., 2002 (PMID 12486378, abachi).
- Woods B, Calnan CD, "Toxic woods", Br J Dermatol 1976;94 Suppl 12:1-97 (PMID 132958). This is a handbook-type review that probably covers mansonia, opepe, makore and ebony. It was not read.

## Other open items

- The HSE-WIS30 rev 1 (11/12) text was read from the Wayback Machine snapshot of 2013-02-13 (http://web.archive.org/web/20130213131629id_/http://www.hse.gov.uk/pubns/wis30.pdf). hse.gov.uk/pubns/wis30.pdf now returns 404. The agency agent should record the archive URL.
- HSE's "Ebony" row does not name a species. It is applied to *D. crassiflora* as the main commercial African ebony.
- Wherever HSE says only "dermatitis", I used the type skin_irritant, because HSE does not say whether the reaction is allergic. I used skin_sensitizer only where HSE says "sensitiser" (ebony, mansonia).
