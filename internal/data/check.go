package data

import (
	"fmt"
	"strings"

	"github.com/brennanMKE/WoodDust/internal/model"
)

// Report lists integrity problems. Errors block seeding; warnings do not.
type Report struct {
	Errors   []string
	Warnings []string
}

// OK reports whether there are no errors.
func (r Report) OK() bool { return len(r.Errors) == 0 }

func (r *Report) errorf(format string, a ...any) {
	r.Errors = append(r.Errors, fmt.Sprintf(format, a...))
}

func (r *Report) warnf(format string, a ...any) {
	r.Warnings = append(r.Warnings, fmt.Sprintf(format, a...))
}

// Merge appends another report's findings.
func (r *Report) Merge(o Report) {
	r.Errors = append(r.Errors, o.Errors...)
	r.Warnings = append(r.Warnings, o.Warnings...)
}

// Collision is one normalized name claimed by more than one species.
type Collision struct {
	Normalized string
	Slugs      []string
}

// Check validates cross-file integrity:
//   - every cited source key exists in sources.json (species and regulatory);
//   - slugs and ids are unique, and each slug matches its file name;
//   - featured_rank values are unique;
//   - every hazard has at least one source;
//   - a species slug is not another species' alias (error), and alias names
//     are unique across species (collisions are warnings: those names stay
//     searchable but do not redirect).
func Check(ds *Dataset) Report {
	var r Report
	ids := map[string]string{}
	ranks := map[int]string{}
	for _, sp := range ds.Species {
		file := fileBase(ds, sp.Slug)
		if want := sp.Slug + ".json"; file != want && ds.Files[sp.Slug] != "" {
			r.errorf("%s: slug %q does not match file name (want %s)", file, sp.Slug, want)
		}
		if prev, dup := ids[sp.ID]; dup {
			r.errorf("%s: id %q already used by %s", file, sp.ID, prev)
		}
		ids[sp.ID] = sp.Slug
		if sp.FeaturedRank != nil {
			if prev, dup := ranks[*sp.FeaturedRank]; dup {
				r.errorf("%s: featured_rank %d already used by %s", file, *sp.FeaturedRank, prev)
			}
			ranks[*sp.FeaturedRank] = sp.Slug
		}
		if len(sp.CommonNames) == 0 {
			r.errorf("%s: no common_names", file)
		}
		for i, h := range sp.Hazards {
			if len(h.Sources) == 0 {
				r.errorf("%s: hazards[%d] (%s) has no sources", file, i, h.Type)
			}
			for _, k := range h.Sources {
				if _, ok := ds.Sources[k]; !ok {
					r.errorf("%s: hazards[%d] cites unknown source %q", file, i, k)
				}
			}
			if _, ok := model.EvidenceLabels[h.EvidenceLevel]; !ok {
				r.errorf("%s: hazards[%d] has invalid evidence_level %d", file, i, h.EvidenceLevel)
			}
		}
		for i, e := range sp.ExposureLimits {
			if _, ok := ds.Sources[e.Source]; !ok {
				r.errorf("%s: exposure_limits[%d] cites unknown source %q", file, i, e.Source)
			}
		}
		for i, c := range sp.CarcinogenClass {
			if _, ok := ds.Sources[c.Source]; !ok {
				r.errorf("%s: carcinogen_class[%d] cites unknown source %q", file, i, c.Source)
			}
		}
		for i, c := range sp.KnownCompounds {
			if _, ok := ds.Sources[c.Source]; !ok {
				r.errorf("%s: known_compounds[%d] cites unknown source %q", file, i, c.Source)
			}
		}
	}
	for _, k := range regulatorySourceKeys(ds.Regulatory) {
		if _, ok := ds.Sources[k]; !ok {
			r.errorf("regulatory.json cites unknown source %q", k)
		}
	}

	slugNorm := map[string]string{}
	for _, sp := range ds.Species {
		slugNorm[model.Normalize(sp.Slug)] = sp.Slug
	}
	for _, c := range Collisions(ds) {
		if owner, isSlug := slugNorm[c.Normalized]; isSlug {
			r.errorf("name %q is the slug of %s but is also a name of %s", c.Normalized, owner, strings.Join(without(c.Slugs, owner), ", "))
			continue
		}
		r.warnf("name %q is shared by %s; it stays searchable but /wood/%s will not redirect", c.Normalized, strings.Join(c.Slugs, ", "), strings.ReplaceAll(c.Normalized, " ", "-"))
	}
	return r
}

// Collisions lists normalized names that belong to more than one species.
func Collisions(ds *Dataset) []Collision {
	owners := map[string]map[string]bool{}
	var order []string
	for _, sp := range ds.Species {
		for _, n := range sp.Names() {
			k := model.Normalize(n.Name)
			if owners[k] == nil {
				owners[k] = map[string]bool{}
				order = append(order, k)
			}
			owners[k][sp.Slug] = true
		}
	}
	var out []Collision
	for _, k := range order {
		if len(owners[k]) > 1 {
			out = append(out, Collision{Normalized: k, Slugs: strings.Split(joinSorted(owners[k]), ", ")})
		}
	}
	return out
}

func without(list []string, drop string) []string {
	var out []string
	for _, v := range list {
		if v != drop {
			out = append(out, v)
		}
	}
	return out
}

// CheckKeys reports unknown source keys cited from elsewhere (e.g. content
// pages). label prefixes each message.
func CheckKeys(ds *Dataset, label string, keys []string) Report {
	var r Report
	for _, k := range keys {
		if _, ok := ds.Sources[k]; !ok {
			r.errorf("%s cites unknown source %q", label, k)
		}
	}
	return r
}
