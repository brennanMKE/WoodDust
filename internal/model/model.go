// Package model holds the published-data types shared by the loader, the
// stores and the HTTP handlers. The structs mirror data/schema/*.json; the
// JSON files in data/ are the source of truth.
package model

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// Species is one data/species/<slug>.json record.
type Species struct {
	ID                    string            `json:"id"`
	Slug                  string            `json:"slug"`
	BotanicalName         string            `json:"botanical_name"`
	BotanicalNameVerified *bool             `json:"botanical_name_verified,omitempty"`
	BotanicalNameSource   string            `json:"botanical_name_source,omitempty"`
	Synonyms              []string          `json:"synonyms,omitempty"`
	Family                string            `json:"family"`
	CommonNames           []string          `json:"common_names"`
	WoodType              string            `json:"wood_type"`
	Region                string            `json:"region"`
	Description           string            `json:"description,omitempty"`
	Hazards               []Hazard          `json:"hazards"`
	ExposureLimits        []ExposureLimit   `json:"exposure_limits,omitempty"`
	CarcinogenClass       []CarcinogenClass `json:"carcinogen_class,omitempty"`
	KnownCompounds        []KnownCompound   `json:"known_compounds,omitempty"`
	RelatedSpeciesNote    string            `json:"related_species_note,omitempty"`
	ImagePrompt           string            `json:"image_prompt,omitempty"`
	ImageAlt              string            `json:"image_alt,omitempty"`
	Images                []Image           `json:"images,omitempty"`
	FeaturedRank          *int              `json:"featured_rank,omitempty"`
	LastReviewed          string            `json:"last_reviewed"`
	NotesInternal         string            `json:"notes_internal,omitempty"`
}

// Hazard is one entry in a species' hazards array.
type Hazard struct {
	Type          string   `json:"type"`
	Effects       []string `json:"effects"`
	Route         []string `json:"route"`
	SourcePart    []string `json:"source_part,omitempty"`
	Summary       string   `json:"summary"`
	EvidenceLevel int      `json:"evidence_level"`
	Sources       []string `json:"sources"`
}

// ExposureLimit is a species-specific occupational limit.
type ExposureLimit struct {
	Body      string  `json:"body"`
	ValueMgM3 float64 `json:"value_mg_m3"`
	Fraction  string  `json:"fraction,omitempty"`
	Note      string  `json:"note,omitempty"`
	Source    string  `json:"source"`
}

// CarcinogenClass is an agency classification for a species.
type CarcinogenClass struct {
	Body   string `json:"body"`
	Class  string `json:"class"`
	Source string `json:"source"`
}

// KnownCompound is a named chemical in the wood.
type KnownCompound struct {
	Name   string `json:"name"`
	Role   string `json:"role,omitempty"`
	Source string `json:"source"`
}

// Image is generated or licensed artwork metadata (internal; not published).
type Image struct {
	File        string `json:"file"`
	License     string `json:"license"`
	Attribution string `json:"attribution"`
	Generated   *bool  `json:"generated,omitempty"`
}

// Source is one entry of data/sources.json. Key is filled in from the map key.
type Source struct {
	Key              string `json:"key"`
	Tier             string `json:"tier"`
	Citation         string `json:"citation"`
	Title            string `json:"title,omitempty"`
	DOI              string `json:"doi,omitempty"`
	PMID             string `json:"pmid,omitempty"`
	PMCID            string `json:"pmcid,omitempty"`
	URL              string `json:"url,omitempty"`
	Accessed         string `json:"accessed,omitempty"`
	License          string `json:"license,omitempty"`
	Verified         bool   `json:"verified"`
	VerificationNote string `json:"verification_note,omitempty"`
}

// Link returns the best outbound link for a source: its URL, else its DOI,
// else its PubMed record. Empty when it has none.
func (s Source) Link() string {
	switch {
	case s.URL != "":
		return s.URL
	case s.DOI != "":
		return "https://doi.org/" + s.DOI
	case s.PMID != "":
		return "https://pubmed.ncbi.nlm.nih.gov/" + s.PMID + "/"
	}
	return ""
}

// RegulatorySummary is the site-wide limits summary from data/regulatory.json
// (contract §3, GET /species/{slug}).
type RegulatorySummary struct {
	EUHardwoodMgM3 *float64 `json:"eu_hardwood_mg_m3"`
	UKHardwoodMgM3 *float64 `json:"uk_hardwood_mg_m3"`
	UKSoftwoodMgM3 *float64 `json:"uk_softwood_mg_m3"`
	NIOSHRELMgM3   *float64 `json:"niosh_rel_mg_m3"`
}

// Name kinds stored in species_names and reported by search.
const (
	KindCommon    = "common"
	KindTrade     = "trade"
	KindBotanical = "botanical"
	KindSynonym   = "synonym"
)

// NameEntry is one searchable name for a species.
type NameEntry struct {
	Name string
	Kind string
}

// Names returns every searchable name for the species, in priority order
// (display name first), with duplicates (after Normalize) removed. The slug
// and botanical id are included so /wood/<id> resolves.
func (s Species) Names() []NameEntry {
	var out []NameEntry
	seen := map[string]bool{}
	add := func(name, kind string) {
		n := Normalize(name)
		if n == "" || seen[n] {
			return
		}
		seen[n] = true
		out = append(out, NameEntry{Name: name, Kind: kind})
	}
	for _, n := range s.CommonNames {
		add(n, KindCommon)
	}
	add(s.BotanicalName, KindBotanical)
	for _, n := range s.Synonyms {
		add(n, KindSynonym)
	}
	add(s.Slug, KindCommon)
	add(s.ID, KindBotanical)
	return out
}

// DisplayName is the first common name with its first letter upper-cased
// ("western red cedar" -> "Western red cedar"; "WRC" stays "WRC").
func (s Species) DisplayName() string {
	if len(s.CommonNames) == 0 {
		return s.Slug
	}
	return SentenceCase(s.CommonNames[0])
}

// SentenceCase upper-cases the first rune only.
func SentenceCase(v string) string {
	r, size := utf8.DecodeRuneInString(v)
	if r == utf8.RuneError {
		return v
	}
	return string(unicode.ToUpper(r)) + v[size:]
}

// HazardTypes returns the distinct hazard types in order of first appearance.
func (s Species) HazardTypes() []string {
	out := []string{}
	seen := map[string]bool{}
	for _, h := range s.Hazards {
		if !seen[h.Type] {
			seen[h.Type] = true
			out = append(out, h.Type)
		}
	}
	return out
}

// MaxEvidenceLevel is the strongest specific level (1 strongest), ignoring
// level 0. nil when the species only has level-0 hazards.
func (s Species) MaxEvidenceLevel() *int {
	var best *int
	for _, h := range s.Hazards {
		if h.EvidenceLevel <= 0 {
			continue
		}
		if best == nil || h.EvidenceLevel < *best {
			v := h.EvidenceLevel
			best = &v
		}
	}
	return best
}

// HasHazardType reports whether any hazard has the given type.
func (s Species) HasHazardType(t string) bool {
	for _, h := range s.Hazards {
		if h.Type == t {
			return true
		}
	}
	return false
}

// HasEvidenceLevel reports whether any hazard has the given level.
func (s Species) HasEvidenceLevel(level int) bool {
	for _, h := range s.Hazards {
		if h.EvidenceLevel == level {
			return true
		}
	}
	return false
}

// CitedKeys returns every source key the record cites, in order of first use.
func (s Species) CitedKeys() []string {
	var out []string
	seen := map[string]bool{}
	add := func(k string) {
		if k != "" && !seen[k] {
			seen[k] = true
			out = append(out, k)
		}
	}
	for _, h := range s.Hazards {
		for _, k := range h.Sources {
			add(k)
		}
	}
	for _, e := range s.ExposureLimits {
		add(e.Source)
	}
	for _, c := range s.CarcinogenClass {
		add(c.Source)
	}
	for _, c := range s.KnownCompounds {
		add(c.Source)
	}
	return out
}

// Public returns a copy without the internal-only fields (contract §3).
func (s Species) Public() Species {
	s.NotesInternal = ""
	s.ImagePrompt = ""
	s.Images = nil
	return s
}

// SearchHit is one search result row (best-matching name per species).
type SearchHit struct {
	Slug          string
	Name          string
	BotanicalName string
	MatchedName   string
	MatchKind     string
	Score         float64
	FeaturedRank  *int
}

// EvidenceLabel holds the display strings for one evidence level (contract §5).
type EvidenceLabel struct {
	Label string
	Short string
}

// EvidenceLabels maps evidence_level to its display text.
var EvidenceLabels = map[int]EvidenceLabel{
	1: {"Confirmed in clinical testing", "Confirmed"},
	2: {"Listed by an agency", "Agency-listed"},
	3: {"Reported", "Reported"},
	0: {"Applies to all wood dust", "General"},
}

// HazardTypeNames maps hazard type ids to display names (contract §5).
var HazardTypeNames = map[string]string{
	"respiratory_sensitizer":       "Asthma / respiratory allergy",
	"respiratory_irritant":         "Respiratory irritation",
	"skin_sensitizer":              "Skin allergy",
	"skin_irritant":                "Skin irritation",
	"eye_irritant":                 "Eye irritation",
	"hypersensitivity_pneumonitis": "Hypersensitivity pneumonitis",
	"systemic_toxic":               "Whole-body (systemic) effects",
	"cardiac":                      "Heart effects",
	"carcinogen":                   "Cancer",
	"splinter_infection":           "Splinter infection",
	"photosensitization":           "Sun sensitivity (photosensitization)",
}

// HazardTypeName returns the display name for a hazard type.
func HazardTypeName(t string) string {
	if n, ok := HazardTypeNames[t]; ok {
		return n
	}
	return strings.ReplaceAll(t, "_", " ")
}
