package contrib

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/brennanMKE/WoodDust/internal/model"
)

// ExportNotice heads every export: the output is a starting point for a
// human, never data to commit as is.
const ExportNotice = "DRAFT for human editing. Rewrite every summary in our own words, check each source " +
	"(tier, DOI/PMID, verified), assign real source keys and an evidence level, then open a pull request. " +
	"Nothing here has been reviewed."

// Export is GET /api/v1/submissions/{id}/export.
type Export struct {
	Draft     bool   `json:"draft"`
	Notice    string `json:"notice"`
	Reference string `json:"reference"`
	Kind      string `json:"kind"`
	// Action is "create" for a new data/species file, "patch" for a change to
	// an existing one.
	Action string `json:"action"`
	File   string `json:"file"`
	// Species is the proposed file content (create) or the current record
	// with the proposed changes applied (patch).
	Species map[string]any `json:"species"`
	// Changes lists what a patch changed, in words.
	Changes []string `json:"changes,omitempty"`
	// ProposedSources are new data/sources.json entries under placeholder keys.
	ProposedSources map[string]map[string]any `json:"proposed_sources"`
	// SubmitterDetails is the submitter's text, for reference only.
	SubmitterDetails string `json:"submitter_details"`
}

var nonSlug = regexp.MustCompile(`[^a-z0-9]+`)

func slugify(s string) string {
	return strings.Trim(nonSlug.ReplaceAllString(model.Normalize(s), "-"), "-")
}

func todoKey(ref string, i int) string { return fmt.Sprintf("TODO-%s-%d", ref, i+1) }

// BuildExport makes the best-effort draft for an accepted submission. sp is
// the target species when the submission names one.
func BuildExport(s Submission, sp *model.Species) (Export, error) {
	if s.Kind == "reaction_report" {
		return Export{}, errors.New("reaction reports are leads only and are never exported")
	}
	ex := Export{Draft: true, Notice: ExportNotice, Reference: s.Reference, Kind: s.Kind,
		ProposedSources: map[string]map[string]any{}, SubmitterDetails: s.Details}
	var keys []string
	for i, c := range s.Citations {
		k := todoKey(s.Reference, i)
		keys = append(keys, k)
		entry := map[string]any{"tier": "TODO (A, B or C)", "citation": c.Text, "verified": false,
			"accessed": time.Now().UTC().Format("2006-01-02")}
		if c.Text == "" {
			entry["citation"] = "TODO: full citation"
		}
		for f, v := range map[string]string{"doi": c.DOI, "pmid": c.PMID, "url": c.URL} {
			if v != "" {
				entry[f] = v
			}
		}
		ex.ProposedSources[k] = entry
	}
	if len(keys) == 0 {
		keys = []string{"TODO-source-key"}
	}
	hazard := func() map[string]any {
		t := "TODO: hazard type"
		if s.HazardType != nil {
			t = *s.HazardType
		}
		return map[string]any{
			"type": t, "effects": []string{"TODO"}, "route": []string{"inhalation"},
			"summary":        "TODO: write in our own words; do not paste the submitter's text or an abstract.",
			"evidence_level": "TODO (1, 2 or 3)", "sources": keys,
		}
	}

	if s.Kind == "new_species" || sp == nil {
		name := ""
		if s.SpeciesName != nil {
			name = *s.SpeciesName
		}
		botanical := "TODO"
		if s.BotanicalName != nil && *s.BotanicalName != "" {
			botanical = *s.BotanicalName
		}
		slug := slugify(name)
		if slug == "" {
			slug = "todo"
		}
		ex.Action, ex.File = "create", "data/species/"+slug+".json"
		ex.Species = map[string]any{
			"id": slugify(botanical), "slug": slug, "botanical_name": botanical, "botanical_name_verified": false,
			"family": "TODO", "common_names": []string{name}, "wood_type": "TODO (hardwood or softwood)",
			"region": "TODO", "hazards": []any{hazard()}, "last_reviewed": "TODO",
		}
		return ex, nil
	}

	// Patch an existing species: start from the published record.
	ex.Action, ex.File = "patch", "data/species/"+sp.Slug+".json"
	cur := *sp
	cur.Hazards = append([]model.Hazard(nil), sp.Hazards...)
	var extra []any
	switch {
	case s.Kind == "new_source" && s.HazardType != nil:
		added := false
		for i, h := range cur.Hazards {
			if h.Type == *s.HazardType && h.EvidenceLevel > 0 {
				cur.Hazards[i].Sources = append(append([]string(nil), h.Sources...), keys...)
				ex.Changes = append(ex.Changes, fmt.Sprintf("hazards[%d] (%s): add sources %s", i, h.Type, strings.Join(keys, ", ")))
				added = true
				break
			}
		}
		if !added {
			extra = append(extra, hazard())
			ex.Changes = append(ex.Changes, "add a hazard entry ("+*s.HazardType+") citing "+strings.Join(keys, ", "))
		}
	case s.HazardType != nil:
		extra = append(extra, hazard())
		ex.Changes = append(ex.Changes, "add or correct the "+*s.HazardType+" hazard; see submitter_details")
	default:
		ex.Changes = append(ex.Changes, "TODO: apply the correction described in submitter_details")
	}
	ex.Changes = append(ex.Changes, "last_reviewed: set to the review date")
	doc := toMap(cur)
	if len(extra) > 0 {
		hs, _ := doc["hazards"].([]any)
		doc["hazards"] = append(hs, extra...)
	}
	doc["last_reviewed"] = "TODO"
	ex.Species = doc
	return ex, nil
}

func (a *API) export(w http.ResponseWriter, r *http.Request, rawID string) {
	if !a.needStore(w) {
		return
	}
	if _, ok := a.authorize(w, r, ScopeSubmissionsRead); !ok {
		return
	}
	id, ok := parseID(w, rawID)
	if !ok {
		return
	}
	s, _, err := a.Store.GetSubmission(r.Context(), id)
	if errors.Is(err, ErrNotFound) {
		writeError(w, http.StatusNotFound, "not_found", "No such submission.")
		return
	}
	if err != nil {
		a.internal(w, r, err)
		return
	}
	if s.Status != "accepted" && s.Status != "exported" && s.Status != "merged" {
		writeError(w, http.StatusConflict, "not_accepted", "Only accepted submissions can be exported (status is "+s.Status+").")
		return
	}
	var sp *model.Species
	if s.TargetSlug != nil {
		got, found, err := a.Species(r.Context(), *s.TargetSlug)
		if err != nil {
			a.internal(w, r, err)
			return
		}
		if found {
			sp = &got
		}
	}
	ex, err := BuildExport(s, sp)
	if err != nil {
		writeError(w, http.StatusConflict, "not_exportable", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, ex)
}

// toMap round-trips a value through JSON into a generic map.
func toMap(v any) map[string]any {
	b, _ := json.Marshal(v)
	var m map[string]any
	_ = json.Unmarshal(b, &m)
	return m
}
