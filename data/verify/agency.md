# Agency / regulatory verification log (agency agent, 2026-11-07)

Items that could not be fully verified, or need a human decision. Everything else in
`data/sources/agency.json` and `data/regulatory.json` was confirmed against the live page or document.

## Needs a decision from Brennan

1. **HSE WIS30 licence.** WIS30 has been withdrawn from hse.gov.uk (404 since about January 2020). The PDF says only "© Crown copyright" and points to HSE's copyright page. In 2013 that page allowed reuse of HSE website and print material under the Open Government Licence (v1.0 named at the time). Today's footer says OGL v3.0, but WIS30 is no longer on the site. So OGL probably applies, but nothing confirms v3 for this document. Recommendation: keep it facts-only with our own wording. To be certain, email copyright@hse.gov.uk.
2. **HSE-WIS30 cites an archive URL.** The only working copy is on the Internet Archive. `verified: true` in agency.json means the content was confirmed from that archive, not that HSE still publishes it. Decide whether the site should show a "withdrawn by HSE" note when it cites WIS30.
3. **ACGIH "mahogany" (handoff task 5): unresolved.** The OSHA page says only "mahogany", with no botanical name. EH40 and WIS30 also use only the common name. The ACGIH TLV Documentation for wood dust might say more, but it is paywalled and was not checked. Do not map the A2 class to *Khaya*. The *Swietenia* file should also say that ACGIH names no species.

## Not verified / blocked

- **POWO** (powo.science.kew.org): returned HTTP 403 to curl and WebFetch. The citation format comes from search snippets, and the licence was not confirmed, so `verified: false`. WFO was confirmed (CC BY 4.0).
- **OSHA pages** (osha.gov) block curl (403). They were read through WebFetch summaries, which quote the text but did not give a raw page dump. Values were cross-checked against the NIOSH Pocket Guide, whose OSHA PEL matches.
- **IARC Vol 62 chapter text:** the evaluation was confirmed through the IPCS INCHEM summary, not the IARC PDF. NCBI Bookshelf (NBK493450) was blocked by reCAPTCHA. Vol 100C was confirmed from the IARC PDF itself.
- **EU-CMRD licence line:** I did not read the EUR-Lex legal notice or Decision 2011/833/EU this session.
- **EH40 ISBN mismatch:** the HSE publication page shows 9780717667031, but the PDF shows 978 0 7176 6733 8. The PDF value is used.
- **EH40 future revision:** the HSE page says revised limits are coming for "three" outstanding EU limits but does not name them. I did not confirm whether hardwood dust (EU 2 mg/m³ vs GB 3 mg/m³) is one of them.
- **SENSOR-2017 date:** the handoff says "Summer 2017"; CDC Stacks says published 2017-05-03 (Vol 28, Issue 3, pp. 1-4). The Stacks value is used. No licence is stated.
- **Schlunssen-2018 licence:** "cc by" comes from Europe PMC metadata, and the version was not shown. Baatjies-2023 is an NIH author manuscript with no open licence.
- **HSE WIS14** (RPE for wood dust): its old URLs now redirect to the WIS23 page. Its status (withdrawn or merged) was not confirmed, so it was not added. HSG53 and INDG479 are used instead.
- **HSE-WIS32 subtitle:** the subtitle comes from the publication page description. The PDF title page was not read.
- **HSE-WoodFire, HSE-COSHH-Woodworking, HSE-FitTesting, HSE-HSG258/HSG53/INDG479:** these pages are live (HTTP 200) and their titles were confirmed. The body content and PDF licence lines were not reviewed.
- **NIOSH HC4/HC7/HC9** (sander controls): only the redirects to CDC Stacks were confirmed. Only HC8 was added.
- **OSHA Combustible Dust NEP directive number:** search results disagree about whether the 2023 reissue kept number CPL 03-00-008, so it was not added.

## WIS30 facts file caveats (`data/wis30-facts.json`)

- WIS30 Table 2 gives **trade names only**, with no botanical names. The only botanical name in the sheet is *Tectona grandis* (teak). Keys are therefore WIS30 trade names, not botanical names as the brief asked. The `project_botanical_name` fields come from this repo's species files, not from WIS30.
- These 16 WIS30 timbers have no project species file: abura/bahia, afzelia/doussie, agba/tola, andiroba/crabwood, avodire, ayan/movingui, basralocus/angelique, bubinga, cedar (Central/South American), freijo/cordia, gedu nohor/edinam, guarea, gum (southern blue), idigbo, utile, whitewood (American). I did not confirm their botanical names.
- Trade names that cover several species (mahogany, ebony, rosewood, walnut "not African", padauk) may be narrower in the project files than WIS30 intended.
- WIS30's statement that hardwood and softwood dust both have a 5 mg/m³ WEL is out of date. Use HSE-EH40 for limits.
