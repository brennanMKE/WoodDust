# Education pages verification log (content agent, 2026-11-07)

Pages in `content/pages/`. Facts were checked against the live agency pages or PDFs listed in `data/sources.json`. These items could not be fully verified, or they need a decision.

## Statements without a direct source (advice or reasoning, flagged for review)

- **Gloves near rotating machinery** (`protection/skin-and-eyes`): "Do not wear gloves where they could be caught by a spinning blade, cutter, bit or lathe work." This is standard machine-safety advice, but I found no HSE, NIOSH or OSHA page that says it for woodworking. Search results only paraphrased HSE inspectors. A citation is needed, perhaps HSE L114 (woodworking machinery ACOP) or PUWER guidance.
- **Goggles vs safety glasses for fine dust, face shields at the lathe, rinsing the eye with water** (`protection/skin-and-eyes`): NIOSH-EyeSafety lists the protector types and says to choose by hazard and fit. It does not compare them. No first-aid source was checked.
- **Ambient air cleaners** (`protection/dust-collection`): no agency source evaluates ceiling-hung or box-fan air cleaners for hobby shops. The text treats them as an extra layer after capture at source, reasoning from HSE-WIS23 and the NIOSH hierarchy of controls.
- **Collector vs shop vacuum roles** (`protection/dust-collection`): this is drawn from HSE-WIS23 (collectors carry chips and fine dust) and NIOSH-HC8 (suction through the sander pad). No agency source compares hobby collectors and shop vacs directly. There is also no Tier A source for collector filter ratings (micron or efficiency specs).
- **"If your vacuum doesn't carry a dust class, look for a high-efficiency filter"** (`protection/cleanup`): this is practical advice. Not checked: that dust class M (EN 60335-2-69) is a European rating not commonly used in the US.
- **Oily-rag disposal steps** (`protection/fire-and-explosion`, `shop-poster`): the steps are dry flat outdoors, or a sealed metal can with water. CambridgeFD-2023 confirms the hazard but gives no disposal steps. The Massachusetts DFS page "Disposing of oily rags" (https://www.mass.gov/info-details/disposing-of-oily-rags) returned HTTP 403 and was not read. Search snippets show it advises a listed oily waste container. Add it as a source if it can be read, or cite the finish maker's label guidance.
- **Fire housekeeping extras** (`protection/fire-and-explosion`): "Keep area around heaters, motors and panels clear", "store finishes closed", and "extinguisher within reach" are general fire-safety advice and are not cited.
- **Symptom page advice**: the emergency line (breathing trouble, chest pain, facial swelling), "a specialist can arrange tests", and "don't stop prescribed medicines" are generic medical-safety wording without a citation. The site's medical reviewer, if there is one, should confirm.
- **Glossary definitions without a citation**: specific inhalation challenge, patch test, specific IgE, HEPA, and assigned protection factor ("a factor of 10 means about a tenth gets through"). These are standard definitions. A Tier A or B citation could be added.
- **"What 1 mg/m³ looks like"** (`limits`): the 45 m³ garage example is our own arithmetic, not a sourced figure.
- **Hand plane or scraper instead of sanding** (`protection`): this is reasoning from EH40 para 41 (sanding makes fine dust). No source recommends the substitution itself.

## Facts checked, with caveats

- **ACGIH class names** ("confirmed human carcinogen" for A1, "suspected" for A2): regulatory.json records only the class letters from OSHA-CSI. The meanings are ACGIH's standard definitions but were not re-read on the OSHA page this session. A4 is described only by its letter.
- **Baatjies-2023 statements** (risks, limits): these come from a WebFetch summary of the PMC author manuscript: a pooled asthma risk higher in exposed workers (RR 1.53 from a meta-analysis), cumulative exposure linked to more symptoms, and effects at exposure levels below some limits. Specific numbers were deliberately not published. A human read of the article is recommended.
- **HSE-WoodFire 60 g/m³ for wood flour**: this was read through a WebFetch summary, not page source. The comparison "60,000 times 1 mg/m³" is our arithmetic.
- **APF figures** (OSHA 1910.134 Table 1): read through WebFetch. Half mask 10, full facepiece 50, PAPR 25 to 1,000 depending on headpiece. HSE's "APF 20 / FFP3" comes from WIS23 (PDF read). The two systems are presented separately and not equated.
- **NIOSH-HC8 "about 90%"**: this is from the CDC Stacks abstract only. The PDF was not opened. The 1996 device was not commercially available at the time, so the page uses it only as evidence that on-tool extraction works.
- **NCI-PDQ-Sinonasal**: NCI's reuse terms for PDQ text were not re-read. Only facts are used, in our own words.
- **UK softwood WEL fraction**: regulatory.json leaves the softwood fraction blank. EH40 says dust WELs refer to the inhalable fraction unless stated otherwise, so softwood 5 mg/m³ is probably inhalable too. The limits table copies regulatory.json exactly and shows no fraction for softwood.
- **HSE-WIS30**: cited only for species-effect facts and its statement that unlisted woods may also be harmful. It is never cited for limits, and its 5 mg/m³ figures are not used anywhere.

## Not used, and why

- **NFPA 664 / NFPA 660**: the edition status is unclear. Search results suggest NFPA 664 was folded into NFPA 660 (2025), and the 5,000 sq ft scope threshold wording differs between editions. The NFPA site page was not readable. No NFPA source was added, so fire content rests on HSE-WIS32, HSE-WoodFire, HSE-WIS23 and OSHA-CombustibleDust. Add an NFPA citation once the edition is confirmed.
- **HSE skin and asthma web pages**: `hse.gov.uk/asthma/symptoms.htm` and `hse.gov.uk/skin/employ/barrier.htm` return 404. HSG262 (PDF) and INDG172 (PDF) were used instead.
- **NIOSH "A Guide to Air-Purifying Respirators" (2018-176)**: the page fetch returned no content.

## Decisions for Brennan

- **Footnote markers**: pages use `[^KEY]` markers only, with no footnote definitions. The Go renderer is expected to resolve keys from `data/sources.json`. Standard Markdown footnote renderers will show the raw markers.
- **Glossary length**: the glossary has about 1,100 words, slightly over the 1,000-word target. It has 42 terms, kept so that cards can link hover definitions to every hazard type.
- **Dataset licence**: the methodology page says only that licence details are on the download page. The CC BY 4.0 proposal is still unconfirmed (PLAN §10).
- **Glossary anchors** are generated from headings (lowercase, non-alphanumerics become `-`), for example `/glossary#assigned-protection-factor`. The renderer's slug function must match.
