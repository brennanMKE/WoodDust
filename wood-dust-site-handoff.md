# Handoff: Wood Dust Hazards Website

## Goal

Build a modern, fast, mobile-friendly website that tells woodworkers which wood species carry specific health hazards as dust (and from skin contact), how strong the evidence is, and how to protect themselves. Every hazard claim on the site must link to a citable source. The site replaces dated, uncited charts that get copied from forum to forum.

Audience: hobby and professional woodworkers, makerspace members, shop instructors. Reading level: plain language first, medical terms explained on hover or in a glossary.

Site name and branding: TBD. Do not add any organization name or logo.

---

## 1. Copyright and data policy (read first)

Not legal advice. When in doubt, flag it for Brennan rather than guessing.

**What we can use:** facts. Under US law, facts (species X has been reported to cause asthma; ACGIH lists oak as A1) are not copyrightable. Citations (author, year, journal, DOI) are facts.

**What we cannot copy:** the *selection, arrangement, and wording* of someone else's compilation. That rules out copying wholesale, or closely mirroring, any of these:

- The Wood Database "Wood Allergies and Toxicity" chart (Eric Meier). Includes its potency (+ to ++++) and incidence ratings, which are his editorial judgments. Many sites (Scroll Saw Village, woodbin, Advantage Lumber, etc.) republish this chart; those copies are equally off limits.
- Hausen, *Woods Injurious to Human Health* (1981) tables.
- AAW / American Woodturner charts (Grabowski 1990, Woodcock 1991, Andi Wolfe tables).
- Fine Woodworking, Woodcarving Illustrated, and other magazine charts.
- ACGIH TLV booklet text (cite values via OSHA's page instead, which restates them).
- Any article abstract or paper text. Cite and link; do not paste.

**Rules for building the dataset:**

1. Build the dataset **species by species from primary sources** (peer-reviewed case reports and studies, government/agency listings). Do not start from someone else's chart and "rephrase" it.
2. Write every description **in our own words**. No sentences lifted from sources.
3. **No potency/incidence star ratings** copied from anywhere. Use our own evidence-level system (section 3), which is derived from the type of source, not someone's judgment.
4. Secondary compilations (Wood Database, Haz-Map) may be used **only to find leads** (which species to research), never as the cited source for a claim.
5. Images: only our own photos or images with a clear license (CC0 / CC BY) with attribution recorded in the data. No hotlinking.
6. **HSE WIS30:** much HSE material is published under the UK Open Government Licence v3, but confirm OGL applies to WIS30 before reusing anything beyond facts. Until confirmed, treat it like the others: cite it as a source for facts, write our own text.

**License for our output:** suggest code under MIT, dataset under CC BY 4.0, so others can reuse the data with attribution. Confirm with Brennan.

---

## 2. Sources

### Tier A: agency / regulatory listings

| Key | Source | What it gives us | URL |
|---|---|---|---|
| `IARC-62` | IARC Monographs Vol. 62 (1995), Wood Dust; reaffirmed Vol. 100C (2012) | All wood dust is Group 1 (carcinogenic to humans); causes nasal and sinus adenocarcinoma | https://publications.iarc.who.int/80 |
| `OSHA-CSI` | OSHA Occupational Chemical Database, Wood Dust | ACGIH TLVs (western red cedar 0.5 mg/m³; all others 1 mg/m³ inhalable); ACGIH carcinogen classes (A1 oak, beech; A2 birch, mahogany, teak, walnut; A4 others incl. western red cedar); NIOSH REL 1 mg/m³; western red cedar DSEN/RSEN | https://www.osha.gov/chemicaldata/798 |
| `HSE-WIS30` | UK HSE Woodworking Information Sheet 30 rev 1, "Toxic woods" (Nov 2012) | ~50 commercial timbers with reported health effects | Original: hse.gov.uk/pubns/wis30.pdf (verify live URL) |
| `EU-CMRD` | EU Directive 2004/37/EC as amended (2017/2398) | Binding hardwood dust limit 2 mg/m³ since 17 Jan 2023 | Verify on eur-lex.europa.eu |
| `HSE-EH40` | HSE EH40/2005 Workplace Exposure Limits | UK limits: hardwood 3 mg/m³, softwood 5 mg/m³ | Verify on hse.gov.uk |

### Tier B: peer-reviewed reviews (use to find the primary studies)

| Key | Source | Notes |
|---|---|---|
| `Schlunssen-2018` | Schlünssen V, Sigsgaard T, Raulf-Heimsoth M, Kespohl S. Workplace exposure to wood dust and the prevalence of wood-specific sensitization. *Allergologie Select* 2018;2(1):101-110. doi:10.5414/ALX01503E. PMID 31826037 | Table of ~45 species linked to asthma, rhinitis, alveolitis, with first-author citations. Open access: https://pmc.ncbi.nlm.nih.gov/articles/PMC6881851 |
| `SENSOR-2017` | Project SENSOR newsletter, Vol 28 No 3 (Summer 2017), "Occupational Asthma and Wood Dust." Michigan State Univ., NIOSH-funded | Species with positive specific inhalation challenge tests, full references. https://stacks.cdc.gov/view/cdc/217198 |
| `Baatjies-2023` | Baatjies R, Chamba P, Jeebhay MF. Wood dust and asthma. *Curr Opin Allergy Clin Immunol* 2023;23(2):76-84 | Recent overview; open copy at https://pmc.ncbi.nlm.nih.gov/articles/PMC9977320 |
| `Goldsmith-1988` | Goldsmith DF, Shy CM. Respiratory health effects from occupational exposure to wood dusts. *Scand J Work Environ Health* 1988;14:1-15. PMID 3281247 | Older but foundational |

Note: Schlünssen and SENSOR disagree on some botanical names (e.g. SENSOR maps "makore" to *Prunus africana*; makore is usually *Tieghemella heckelii*). Resolve against the primary paper.

### Tier C: primary studies (cite these on species pages)

Each needs its PMID or DOI looked up and stored. From SENSOR-2017 and Schlünssen-2018:

| Key | Citation | Species |
|---|---|---|
| `ChanYeung-1973` | Chan-Yeung M et al. Am Rev Respir Dis 1973;108:1094-1102 | Western red cedar |
| `ChanYeung-1976` | Chan-Yeung M, Abboud R. Am Rev Respir Dis 1976;114:1027-1031 | California redwood |
| `Malo-1994` | Malo JL et al. Am J Respir Crit Care Med 1994;150:1697-1701 | Eastern white cedar |
| `Malo-1995` | Malo JL et al. Chest 1995;108:856-858 | Oak |
| `FernandezRivas-1997` | Fernández-Rivas M et al. Allergy 1997;52:196+ (page range misprinted in SENSOR; verify) | Ash |
| `Booth-1976` | Booth BH et al. J Allergy Clin Immunol 1976;57:352-357 | Abiruana |
| `Obata-2000` | Obata H et al. Intern Med 2000;39:947-949 | Makore |
| `Ferrer-2001` | Ferrer A et al. J Investig Allergol Clin Immunol 2001;11:199-203 | Obeche (samba) |
| `Hinojosa-1986` | Hinojosa M et al. Clin Allergy 1986;16:145-153 | Obeche, ramin (cross-reactivity) |
| `Bush-1978` | Bush RK et al. Am Rev Respir Dis 1978;117:601-603 | Zebrawood |
| `Bush-1983` | Bush RK, Clayton D. Clin Allergy 1983;13:389-394 | Central American walnut |
| `Alday-2005` | Alday E et al. J Allergy Clin Immunol 2005;115:634-636 | Angelim pedra |
| `Paggiaro-1981` | Paggiaro PL et al. Clin Allergy 1981;11:605-610 | Tanganyika aningré |
| `Garces-1995` | Garcés Sotillos MM et al. J Investig Allergol Clin Immunol 1995;5:113-114 | Mukali |
| `Higuero-2001` | Higuero NC et al. J Allergy Clin Immunol 2001;107:554-556 | Antiaris |
| `WoodBaker-1997` | Wood-Baker R, Markos J. Aust N Z J Med 1997;27:452-453 | Australian blackwood |
| `Innocenti-1991` | Innocenti A et al. Med Lav 1991;82:446-450 | Cabreuva (asthma + systemic toxic reaction) |
| `Eire-2006` | Eire MA et al. J Investig Allergol Clin Immunol 2006;16:385-387 | Cedroarana |
| `Lee-2009` | Lee LT, Tan KL. Occup Med (Lond) 2009;59:357-359 | Chengal |
| `Eaton-1973` | Eaton KK. Clin Allergy 1973;3:307-310 | Cocobolo |
| `Maestrelli-1987` | Maestrelli P et al. Ann Allergy 1987;59:347-349 | Ebony (*Diospyros crassiflora*) |
| `Jeebhay-1996` | Jeebhay MF et al. J Allergy Clin Immunol 1996;97:1025-1027 | Imbuia |
| `Algranti-2005` | Algranti E et al. J Investig Allergol Clin Immunol 2005;15:81-83 | Ipe |
| `Ricciardi-2003` | Ricciardi L et al. Ann Allergy Asthma Immunol 2003;91:393-397 | Iroko |
| `Ordman-1949` | Ordman D. S Afr Med J 1949;23:973-975 | Kejaat |
| `GozaloReques-1988` | Gozalo Reques F, Pelta Fernandez R. Rev Mal Respir 1988;5:71-73 | Kotibe |
| `Sosman-1969` | Sosman AJ et al. N Engl J Med 1969;281:977-980 | Mahogany |
| `GodnicCvar-1990` | Godnić-Cvar J, Gomzi M. Am J Ind Med 1990;18:541-545 | Palisander (*Dalbergia nigra*) |
| `Basomba-1991` | Basomba A et al. Allergy 1991;46:316-318 | Pau marfim |
| `Raghuprasad-1980` | Raghuprasad PK et al. J Allergy Clin Immunol 1980;65:285-287 | Soapbark |
| `Quirce-2004` | Quirce S et al. (verify full citation) | Jatoba, tali |
| `Kespohl-2006` | Kespohl S et al. (verify) | Black locust |
| `Doyen-padauk` | Doyen V et al. "Eosinophilic occupational asthma caused by padauk wood dust." J Allergy Clin Immunol Pract (verify year/volume) | Padauk |

The Schlünssen-2018 table also cites studies for alder, beech, birch, fir, gaboon, limba, African mahogany, meranti, pine, sapele, spruce, teak, and others (Oertmann 1993, Kersten 1994, Wilhelmsson 1984, Åhman 1995/1996, Kespohl 2010/2011, etc.). Pull those references from its bibliography.

### Tier D: technical references (non-health facts, names)

- USDA Forest Products Laboratory tech sheets (public domain US government work), e.g. cocobolo: https://www.fpl.fs.usda.gov/documnts/TechSheets/Chudnoff/TropAmerican/html_files/dalber3new.html . Good for botanical names, trade names, and brief dermatitis notes.
- BS EN 13556 (timber nomenclature) for standard trade name to botanical name mapping. Paywalled; use for verification only.
- Plants of the World Online (Kew) or World Flora Online for current accepted botanical names and synonyms (e.g. *Chlorophora excelsa* is now *Milicia excelsa*; *Tabebuia* ipe is now mostly *Handroanthus*).

### Leads only (do not cite, do not copy)

The Wood Database, Haz-Map, scrollsawvillage.com, woodbin.com, wooduchoose.com, Fine Woodworking, AAW. Use them to spot species we're missing, then find a Tier A-C source.

---

## 3. Evidence levels (our own system)

Shown as a label on every hazard, not as stars.

| Level | Label | Criteria |
|---|---|---|
| 1 | **Confirmed in clinical testing** | Peer-reviewed report with specific inhalation challenge, patch test, or sIgE confirming the species |
| 2 | **Listed by an agency** | IARC, ACGIH (via OSHA), NIOSH, HSE, EU directive names the species or its group |
| 3 | **Reported** | Case reports or handbook mentions without confirmation testing |
| 0 | **General** | Applies to all wood dust (cancer risk, irritation, explosion/fire) |

A species can carry several hazards at different levels. Never imply a species is "safe"; absence of reports is not evidence of safety, and every species page must say so.

---

## 4. Data model

One JSON (or YAML) file per species in `data/species/`, validated by a JSON Schema in CI. Example:

```json
{
  "id": "thuja-plicata",
  "botanical_name": "Thuja plicata",
  "synonyms": [],
  "family": "Cupressaceae",
  "common_names": ["western red cedar", "red cedar", "WRC"],
  "wood_type": "softwood",
  "region": "Pacific Northwest, North America",
  "hazards": [
    {
      "type": "respiratory_sensitizer",
      "effects": ["asthma", "rhinitis"],
      "route": ["inhalation"],
      "summary": "Our own one or two sentences.",
      "evidence_level": 1,
      "sources": ["ChanYeung-1973", "OSHA-CSI"]
    },
    {
      "type": "skin",
      "effects": ["dermatitis"],
      "route": ["skin"],
      "summary": "...",
      "evidence_level": 2,
      "sources": ["HSE-WIS30"]
    }
  ],
  "exposure_limits": [
    { "body": "ACGIH TLV", "value_mg_m3": 0.5, "fraction": "inhalable", "source": "OSHA-CSI" }
  ],
  "carcinogen_class": [ { "body": "ACGIH", "class": "A4", "source": "OSHA-CSI" } ],
  "known_compounds": [ { "name": "plicatic acid", "source": "..." } ],
  "related_species_note": "Cross-reactivity possible with other Thuja; cite if claimed.",
  "images": [],
  "last_reviewed": "2026-11-07",
  "notes_internal": "..."
}
```

`data/sources.json`: every source key with full citation, DOI/PMID, URL, access date, license, tier.

Controlled vocabularies (enums, defined in the schema):

- `type`: `respiratory_sensitizer`, `respiratory_irritant`, `skin_sensitizer`, `skin_irritant`, `eye_irritant`, `hypersensitivity_pneumonitis`, `systemic_toxic`, `cardiac`, `carcinogen`, `splinter_infection`, `photosensitization`
- `route`: `inhalation`, `skin`, `eyes`, `splinter`, `ingestion`
- `source part` (optional field): `dust`, `heartwood`, `sap`, `bark`, `leaves`

Build step generates `dist/data/species.json` and `species.csv` for public download.

---

## 5. Seed list

Starting set of species to research, each with at least one source already identified. Botanical names marked † must be verified against POWO/WFO before entry. Write descriptions fresh from the cited sources.

| Common name | Botanical name | Known hazard leads | Sources to start from |
|---|---|---|---|
| Western red cedar | *Thuja plicata* | Asthma, rhinitis, dermatitis; lowest ACGIH limit | ChanYeung-1973, OSHA-CSI, HSE-WIS30 |
| Eastern white cedar | *Thuja occidentalis* | Asthma | Malo-1994 |
| Redwood | *Sequoia sempervirens* | Asthma; hypersensitivity pneumonitis | ChanYeung-1976, Goldsmith-1988 |
| Oak | *Quercus* spp. | Asthma; ACGIH A1 carcinogen | Malo-1995, OSHA-CSI, HSE-WIS30 |
| Beech | *Fagus sylvatica* | Asthma/rhinitis; ACGIH A1 | Schlunssen-2018, OSHA-CSI, HSE-WIS30 |
| Birch | *Betula* spp. | Asthma/rhinitis; ACGIH A2; dermatitis | Schlunssen-2018, OSHA-CSI, HSE-WIS30 |
| Ash | *Fraxinus* spp. | Asthma; reduced lung function | FernandezRivas-1997, HSE-WIS30 |
| Alder | *Alnus* spp. | Asthma; dermatitis, rhinitis | Schlunssen-2018, HSE-WIS30 |
| Walnut | *Juglans* spp. | ACGIH A2; asthma (*J. olanchana*); rhinitis | OSHA-CSI, Bush-1983, HSE-WIS30 |
| Maple | *Acer* spp. | Reduced lung function (spalted maple: mold, verify) | HSE-WIS30 |
| Mahogany | *Swietenia* spp. | Asthma; ACGIH A2; dermatitis | Sosman-1969, OSHA-CSI, HSE-WIS30 |
| African mahogany | *Khaya* spp. | Asthma; ACGIH A2 per Haz-Map wording (verify which mahogany ACGIH means) | Schlunssen-2018 |
| Teak | *Tectona grandis* | ACGIH A2; potent dermatitis; asthma | OSHA-CSI, HSE-WIS30, Schlunssen-2018 |
| Iroko | *Milicia excelsa* (syn. *Chlorophora excelsa*) | Asthma; dermatitis | Ricciardi-2003, HSE-WIS30 |
| Obeche | *Triplochiton scleroxylon* | Asthma (allergen Trip s 1); urticaria | Ferrer-2001, Hinojosa-1986, Baatjies-2023 |
| Cocobolo | *Dalbergia retusa* | Respiratory allergy; dermatitis | Eaton-1973, USDA FPL tech sheet |
| Brazilian rosewood / palisander | *Dalbergia nigra* | Asthma | GodnicCvar-1990 |
| Rosewoods (genus) | *Dalbergia* spp. | Dermatitis, respiratory | HSE-WIS30 |
| Zebrawood | *Microberlinia brazzavillensis* | Asthma | Bush-1978 |
| Ebony (African) | *Diospyros crassiflora* | Asthma; dermatitis | Maestrelli-1987, HSE-WIS30 |
| Ipe | *Handroanthus* spp.† (syn. *Tabebuia*) | Asthma | Algranti-2005 |
| Jatoba | *Hymenaea courbaril* | Asthma | Quirce-2004 |
| Padauk | *Pterocarpus soyauxii*† | Asthma; itching, eye irritation | Doyen-padauk, HSE-WIS30 |
| Kejaat | *Pterocarpus angolensis* | Asthma | Ordman-1949 |
| Imbuia | *Ocotea porosa* | Asthma | Jeebhay-1996 |
| Cabreuva | *Myrocarpus fastigiatus* | Asthma, alveolitis, systemic reaction | Innocenti-1991 |
| Ramin | *Gonystylus bancanus* | Asthma (cross-reacts with obeche); dermatitis | Hinojosa-1986, HSE-WIS30 |
| Makore | *Tieghemella heckelii*† | Asthma; dermatitis; systemic | Obata-2000 (check species), HSE-WIS30 |
| Limba | *Terminalia superba* | Asthma; nosebleeds; reduced lung function | Schlunssen-2018, HSE-WIS30 |
| Gaboon / okoume | *Aucoumea klaineana* | Asthma; eye and skin effects | Schlunssen-2018, HSE-WIS30 |
| Sapele | *Entandrophragma cylindricum* | Asthma; skin irritation | Schlunssen-2018, HSE-WIS30 |
| Meranti | *Shorea* spp. | Asthma; skin irritation | Schlunssen-2018, HSE-WIS30 |
| Australian blackwood | *Acacia melanoxylon* | Asthma | WoodBaker-1997 |
| Black locust | *Robinia pseudoacacia* | Asthma | Kespohl-2006 |
| Pine | *Pinus* spp. | Asthma (resin acids / colophony); skin, photosensitization | Schlunssen-2018, HSE-WIS30, Baatjies-2023 |
| Spruce | *Picea* spp. | Asthma; respiratory | Schlunssen-2018, HSE-WIS30 |
| Fir | *Abies* spp. | Asthma | Schlunssen-2018 |
| Douglas fir | *Pseudotsuga menziesii*† | Dermatitis, rhinitis, septic splinters | HSE-WIS30 |
| Western hemlock | *Tsuga heterophylla*† | Rhinitis, bronchial effects | HSE-WIS30 |
| Cedar of Lebanon | *Cedrus libani* | Respiratory, rhinitis | Schlunssen-2018 (Greenberg 1972), HSE-WIS30 |
| Yew | *Taxus baccata*† | Dermatitis; systemic and cardiac toxicity | HSE-WIS30 (find primary toxicology source) |
| Greenheart | *Chlorocardium rodiei*† | Septic splinters; cardiac, throat irritation | HSE-WIS30 |
| Wenge | *Millettia laurentii*† | Splinters, dermatitis, neurological effects | HSE-WIS30 |
| Mansonia | *Mansonia altissima*† | Sensitization, respiratory, cardiac | HSE-WIS30 |
| Opepe | *Nauclea diderrichii*† | Dermatitis, neurological, nosebleeds | HSE-WIS30 |
| Peroba | *Aspidosperma* spp.† | Systemic effects, irritation | HSE-WIS30 |
| Afrormosia | *Pericopsis elata*† | Skin irritation, splinters, neurological | HSE-WIS30 |
| Poplar | *Populus* spp. | Sneezing, eye irritation | HSE-WIS30 |
| Larch | *Larix* spp. | Urticaria, dermatitis | HSE-WIS30 |
| Sweet chestnut | *Castanea sativa* | Dermatitis | HSE-WIS30 |

Also research (leads only so far): angelim pedra, aningré/mukali, antiaris, cedroarana, chengal, kotibe, pau marfim, soapbark, tali, abiruana, olive, satinwood, snakewood, purpleheart, goncalo alves, boxwood, sassafras, willow, oleander (not a lumber species but a turning hazard; consider a "do not burn or turn" note).

Target for v1: 50 species fully sourced. Quality over count.

---

## 6. Site requirements

### Pages

- **Home:** search box (common or botanical name, fuzzy, aliases), headline message that *all* wood dust is hazardous, quick links to most-searched species and safety basics.
- **Species page:** names, photo (if licensed), hazards grouped by type with evidence level badges, exposure limits, sources with links, "absence of reports is not safety" notice, last reviewed date, "report a correction" link.
- **Browse / table view:** sortable, filterable by hazard type, evidence level, region, hardwood/softwood. Works on mobile (cards below a breakpoint).
- **Safety basics:** dust collection, respirators (N95 vs P100 vs PAPR, fit testing), sanding vs sawing particle size, skin protection, finishing, shop cleanup (no compressed air), fire/explosion risk. Cite NIOSH/HSE/OSHA guidance; write our own text.
- **Exposure limits explainer:** OSHA vs NIOSH vs ACGIH vs UK vs EU, what mg/m³ means in a shop.
- **Glossary:** sensitizer, irritant, asthma, rhinitis, dermatitis, hypersensitivity pneumonitis, inhalable vs respirable, etc.
- **Sources & methodology:** evidence levels explained, data policy, how to contribute.
- **Printable shop card:** one-page PDF/print stylesheet per species and a summary sheet for posting in a shop.
- **Data download:** JSON + CSV, with license.

### Technical

- Static site, no backend, no user accounts. Suggested: Astro (content collections map well to per-species files) or another static generator Claude Code prefers; justify the choice.
- Client-side search (e.g. Pagefind or Fuse.js) over a generated index.
- JSON Schema validation of all species and source files in CI; build fails on unknown source keys, missing evidence level, or a hazard with no source.
- Link checker in CI for source URLs.
- Accessibility: WCAG 2.2 AA, evidence badges not color-only, keyboard navigable.
- Performance: no tracking scripts, no third-party fonts required, Lighthouse 95+.
- Hosting: any static host (GitHub Pages, Cloudflare Pages). Repo public so corrections come via pull requests.

### Content tone

Plain, calm, practical. No fearmongering, no "this wood is safe." Medical disclaimer in the footer: information only, see a doctor for symptoms.

---

## 7. Contribution workflow

- `CONTRIBUTING.md`: every new hazard needs a Tier A-C source with DOI/PMID or official URL. Leads-only sources rejected.
- PR template with checklist: own words, source verified, evidence level justified, botanical name checked against POWO/WFO.
- Issue template for "report a reaction" that collects species and symptoms but is clearly not a medical service.

---

## 8. Open tasks for Claude Code

1. Scaffold repo, schema, CI validation, and site skeleton.
2. Build `sources.json` from section 2, looking up and recording DOI/PMID for each Tier C citation. Mark any that can't be verified.
3. Enter the first 10 species end to end (western red cedar, oak, beech, walnut, maple, cherry, cocobolo, padauk, teak, iroko) to prove the model, then pause for review.
4. Confirm HSE WIS30 license status and current URL; confirm EU and UK limit figures from official pages.
5. Resolve the botanical name conflicts noted above (makore, ipe, ACGIH "mahogany").
6. Draft Safety basics and Exposure limits pages from NIOSH/OSHA/HSE guidance.

If web access is restricted in the Claude Code environment, list every unverified item in `VERIFY.md` rather than filling gaps from memory.
