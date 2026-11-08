# Verify list: americas-misc batch

Batch species: ipe, jatoba, imbuia, cabreuva, greenheart, peroba, australian-blackwood.
Reviewed 2026-11-07.

## Botanical names (RESOLVED 2026-11-07: all verified on WFO)

Rechecked via the WFO Plant List matching API (list.worldfloraonline.org, classification 2026-06). All seven names are accepted on WFO and `botanical_name_verified` is now `true`, with `botanical_name_source` set to the WFO taxon URL. No accepted-name changes.

| Slug | Name used | WFO result |
|---|---|---|
| ipe | Handroanthus spp. | Genus *Handroanthus* Mattos accepted (wfo-4000016828), Bignoniaceae. *H. serratifolius* (Vahl) S.O.Grose and *H. impetiginosus* (Mart. ex DC.) Mattos accepted. *Tabebuia serratifolia* (Vahl) G.Nicholson = syn. of *H. serratifolius*; *T. serratifolia* Rolfe is unplaced (explains GBIF result). *T. impetiginosa* and *Tecoma avellanedae* = syn. of *H. impetiginosus* |
| jatoba | Hymenaea courbaril | Accepted, *H. courbaril* L. (wfo-0000168642), Fabaceae |
| imbuia | Ocotea porosa | Accepted, (Nees & Mart.) Barroso (wfo-0000383291), Lauraceae. *Phoebe porosa* (Nees & Mart.) Mez = synonym |
| cabreuva | Myrocarpus fastigiatus | Accepted, Allemão (wfo-0001055406), Fabaceae |
| greenheart | Chlorocardium rodiei | Accepted, (R.H.Schomb.) Rohwer, H.G.Richt. & van der Werff (wfo-0000603136), Lauraceae. *Ocotea rodiei* (R.H.Schomb.) Mez and *Nectandra rodiei* R.H.Schomb. ex Meisn. = synonyms |
| peroba | Aspidosperma polyneuron | Accepted, Müll.Arg. (wfo-0000291421), Apocynaceae. *A. peroba* Saldanha = synonym |
| australian-blackwood | Acacia melanoxylon | Accepted, R.Br. (wfo-0000204086), Fabaceae |

- ~~Synonyms entered from general knowledge need confirmation~~ RESOLVED: all confirmed as WFO synonyms. Greenheart synonym spelling corrected from "Ocotea rodiaei" to *Ocotea rodiei* (WFO spelling).
- Ipe: which Handroanthus species are traded as ipe. A search summary of CITES CoP18 Doc.105.3 said H. serratifolius and H. impetiginosus are the most traded, but the PDF returned 403. CITES CoP19 Prop. 44 (Handroanthus, Tabebuia, Roseodendron to Appendix II) also returned 403. Confirm and consider adding as a Tier D source.
- Peroba: HSE-WIS30 says only "Peroba" with no botanical name. Mapping to *A. polyneuron* (peroba rosa) is assumed. "Peroba de campos" is a different tree (usually *Paratecoma peroba*).
- Greenheart: WIS30 gives no botanical name. Mapping to *Chlorocardium rodiei* is assumed.
- Cabreuva: the trade name may also cover *Myrocarpus frondosus* or *Myroxylon* spp. The common name "oleo pardo" is unsourced.

## Evidence levels that depend on unread full text

These papers are letters with no PubMed abstract. PMID and DOI are confirmed, but I did not read the full text.

- **Quirce-2004** (jatoba, tali). Set to level 3. Publisher returned 403. If the letter reports a positive specific inhalation challenge, upgrade to level 1. Also confirm how many patients there were and which wood each one reacted to.
- **Jeebhay-1996** (imbuia) and **WoodBaker-1997** (blackwood). Set to level 1 based on SENSOR-2017 Table 3, which says its listed species had positive specific antigen challenge tests. Confirm against the full text.
- **Yacoub-2005** (bethabara/ipe), **Pala-2010** and **Baur-2000** (cabreuva), **Clarke-1989** and **Tilsley-1990** (blackwood), **Landry-2018** (ipe skin): testing methods unknown. Tilsley-1990 and Landry-2018 may include patch tests, which would justify level 1 for skin.
- **Innocenti-1991**: no DOI found (Med Lav 1991).

## Claims not entered (leads without a usable primary)

- Ipe and lapachol dermatitis. No PubMed primary links lapachol in ipe wood to dermatitis. Estlander 2001 (PMID 11260236) reports patch reactions to lapachol in wood-dust patients but does not tie them to ipe. Cordero & Lynch 1951 (PMID 14844035, *Tecoma avellanedae* sawdust sensitization) has no abstract and its language is undetermined.
- Imbuia: Pitt ML et al. S Afr Med J 1985;67:82 (cited by Schlunssen-2018) was not found on PubMed.
- Peroba (other species): DeJong-1951 (peroba de campos eczema), Jemec-1991 (Aspidosperma sp., "Brazilian box tree") and CabezasArteaga-2019 (Aspidosperma per MeSH). These are defined in the sources file and mentioned in `related_species_note` only.
- Blackwood: Hausen-1981 mentions undefined "toxic effects" reported since 1925. Not entered.
- Greenheart and peroba: WIS30 gives no exposure route for cardiac, intestinal, or systemic effects. Route set to `inhalation` as an assumption.

## Source notes

- HSE-WIS30: hse.gov.uk/pubns/wis30.pdf returned an HTML page, not the PDF. Table 2 was read from a third-party copy (oneida-air.com download) whose footer says "licensed under the Open Government Licence". Confirm the live HSE URL.
- Descriptions and image-prompt wood appearance are from general knowledge. The USDA FPL tech-sheet server returned 502, so no Tier D source is cited for non-health facts.
- "Brazilian walnut" is used as an alias for both imbuia and ipe (decking trade). Decide whether to keep it on both pages.
