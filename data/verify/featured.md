# Verify list: batch `featured`

Species: oak, maple, walnut, cherry, pine, western-red-cedar, eastern-white-cedar, redwood. Researched 2026-11-07.

## Shared sources (for the `agency` agent)

- **HSE-WIS30 live URL is dead.** `https://www.hse.gov.uk/pubns/wis30.pdf` returns a 404 page. I read WIS30 rev 1 (11/12) from the Wayback Machine: `https://web.archive.org/web/20130213131629/http://www.hse.gov.uk/pubns/wis30.pdf`. All HSE-WIS30 claims in this batch come from that copy. Its OGL status is still unconfirmed.
- **POWO blocks scripted access** (Cloudflare 403). I verified botanical names on WFO instead (list.worldfloraonline.org matching API, then the WFO record URLs stored in `botanical_name_source`).
- **OSHA-CSI** confirmed via https://www.osha.gov/chemicaldata/798. It lists A1 oak and beech, A2 birch, mahogany, teak and walnut, and A4 for all others including western red cedar. Western red cedar has a 0.5 mg/m³ TLV with DSEN and RSEN notations.

## Citations without a PubMed abstract (PMID/DOI confirmed, content checked from title only)

For these papers I could not confirm the testing method from an abstract.

- `ChanYeung-1973`: the level 1 rating also rests on ChanYeung-1982 and ChanYeung-1987.
- `Cohen-1967`: sequoiosis. The claim that it is caused by mold (Graphium or Aureobasidium) is not verified, so the summary hedges.
- `Emanuel-1966` and `Wenzel-1967`: maple bark disease, kept at level 3.
- `Malmstrom-1999`: alveolitis from pine sawdust, level 3. Whether mold was involved is unknown.
- `OlveraCortes-2019`: no abstract and no DOI on PubMed. I confirmed the content from the publisher's open case page on mdedge.com.

## Claims needing full-text check

- **Malo-1995:** the abstract does not name the oak species. The Schlunssen-2018 table says *Quercus robur*.
- **Estlander-2001:** the walnut species behind the 2 patch-test-positive patients is not named. The walnut skin-sensitizer entry is level 1 on that basis.
- **Skovsted-2003:** the pine species is not named. **Douwes-2001/2006:** the abstracts say "pine". It is presumably *Pinus radiata* (NZ plantations), but this is not stated.
- **Booken-2006:** a German-language paper. The patch test details are not in the English abstract.
- **WRC "central nervous system effects":** this is an HSE-WIS30 listing only, and I found no primary source. Consider dropping or softening it.
- **Maple "decrease in lung function":** this is an HSE-WIS30 listing only, with no maple species named and no primary source found.
- **Oak and pine skin/eye listings from HSE:** these are agency-only (level 2) and have no primary paper.

## Botanical names

- *Juglans olanchana* (Bush-1983) is treated by WFO as a **synonym of *Juglans pyriformis* Liebm.** The walnut page keeps the paper's name and notes the synonym.
- All genus and species names below were verified on WFO: *Quercus* L., *Acer* L., *Juglans* L., *J. nigra* L., *J. regia* L., *Prunus serotina* Ehrh., *Pinus* L., *Thuja plicata* Donn ex D.Don, *T. occidentalis* L., and *Sequoia sempervirens* (D.Don) Endl.

## Leads not pursued or not verifiable

- From the Schlunssen-2018 table, these are not indexed in PubMed and were not verified:
  - Spiewak 1994, *Ann Agric Environ Med* 1:73-76 (oak and pine asthma)
  - Aldunate 1998, *Allergy* 53:2019 (conference abstract, oak)
  - Abendroth 1992 and Kersten 1994, *Allergologie* (cherry *Prunus avium* "allergy", pine)
  - Oertmann 1993, *Allergologie* (oak)
  - Dutkiewicz 2001 (pine, sawmill)
- Pickering 1972, *Clin Allergy* 2:213-218 (WRC and iroko asthma): PMID not looked up.
- Soylu 2023, *Dermatitis* 34:571-572, PMID 37040275 (juglone bullous dermatitis): closed access, not used.
- Mitchell 1974, PMID 4817210 ("respiratory allergy from Thuja"): not reviewed.
- Black walnut shavings and laminitis in horses (Galey 1991, PMID 2061431): veterinary, not added as a human hazard.
- Spalted maple: PubMed has no human health reports. Do not link spalting to maple bark disease.
- Cherry: I found no species-specific reports for *Prunus serotina* wood. "African cherry" in Obata 2000 is makore (*Tieghemella heckelii*).

## Possible key collisions

- `Kespohl-2012` (softwood IgE allergy, PMID 21986217) may also be defined by the spruce/fir batch. If so, it is the same paper. Other agents may define a `Kespohl-2011` or `Kespohl-2010` for different papers.
