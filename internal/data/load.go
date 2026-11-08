// Package data loads the published data set from the repo's data/ directory
// (species, sources, regulatory) and checks its cross-file integrity. JSON
// Schema validation itself is done by ajv-cli in scripts/check.sh and CI;
// this package checks what a schema cannot: that cited keys exist, slugs are
// unique, and names do not collide across species.
package data

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/brennanMKE/WoodDust/internal/model"
)

// Dataset is everything seed writes to the database.
type Dataset struct {
	Species []model.Species         // sorted by slug
	Sources map[string]model.Source // keyed by source key, Key filled in
	// Regulatory is the raw data/regulatory.json document.
	Regulatory json.RawMessage
	// Summary is the derived site-wide limits summary.
	Summary model.RegulatorySummary
	// SHA256 is a digest over every input file, recorded in seed_runs.
	SHA256 string
	// Files maps each species slug to the file it came from (for messages).
	Files map[string]string
}

// Load reads dir/species/*.json, dir/sources.json and dir/regulatory.json.
// Unknown fields are an error so schema drift is caught here as well as by ajv.
func Load(dir string) (*Dataset, error) {
	ds := &Dataset{Sources: map[string]model.Source{}, Files: map[string]string{}}
	h := sha256.New()

	files, err := filepath.Glob(filepath.Join(dir, "species", "*.json"))
	if err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("no species files in %s", filepath.Join(dir, "species"))
	}
	sort.Strings(files)
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			return nil, err
		}
		h.Write([]byte(filepath.Base(f)))
		h.Write(b)
		var sp model.Species
		if err := decodeStrict(b, &sp); err != nil {
			return nil, fmt.Errorf("%s: %w", f, err)
		}
		if prev, dup := ds.Files[sp.Slug]; dup {
			return nil, fmt.Errorf("duplicate slug %q in %s and %s", sp.Slug, prev, f)
		}
		ds.Files[sp.Slug] = f
		ds.Species = append(ds.Species, sp)
	}
	sort.Slice(ds.Species, func(i, j int) bool { return ds.Species[i].Slug < ds.Species[j].Slug })

	b, err := os.ReadFile(filepath.Join(dir, "sources.json"))
	if err != nil {
		return nil, err
	}
	h.Write(b)
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(b, &raw); err != nil {
		return nil, fmt.Errorf("sources.json: %w", err)
	}
	for k, v := range raw {
		var s model.Source
		if err := decodeStrict(v, &s); err != nil {
			return nil, fmt.Errorf("sources.json %s: %w", k, err)
		}
		s.Key = k
		ds.Sources[k] = s
	}

	b, err = os.ReadFile(filepath.Join(dir, "regulatory.json"))
	if err != nil {
		return nil, err
	}
	h.Write(b)
	if !json.Valid(b) {
		return nil, fmt.Errorf("regulatory.json: invalid JSON")
	}
	ds.Regulatory = b
	ds.Summary, err = summarize(b)
	if err != nil {
		return nil, fmt.Errorf("regulatory.json: %w", err)
	}
	ds.SHA256 = hex.EncodeToString(h.Sum(nil))
	return ds, nil
}

func decodeStrict(b []byte, v any) error {
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}

// summarize derives the contract's regulatory summary. Missing values stay
// null rather than failing, so a data edit cannot take the site down.
func summarize(b []byte) (model.RegulatorySummary, error) {
	var doc struct {
		EU struct {
			Value *float64 `json:"value_mg_m3"`
		} `json:"eu_binding_limit"`
		UK struct {
			Hardwood struct {
				Value *float64 `json:"value_mg_m3"`
			} `json:"hardwood_dust"`
			Softwood struct {
				Value *float64 `json:"value_mg_m3"`
			} `json:"softwood_dust"`
		} `json:"uk_eh40"`
		NIOSH struct {
			Value *float64 `json:"value_mg_m3"`
		} `json:"niosh_rel"`
	}
	if err := json.Unmarshal(b, &doc); err != nil {
		return model.RegulatorySummary{}, err
	}
	return model.RegulatorySummary{
		EUHardwoodMgM3: doc.EU.Value,
		UKHardwoodMgM3: doc.UK.Hardwood.Value,
		UKSoftwoodMgM3: doc.UK.Softwood.Value,
		NIOSHRELMgM3:   doc.NIOSH.Value,
	}, nil
}

// regulatorySourceKeys walks regulatory.json and returns every value under a
// "source" or "sources" key.
func regulatorySourceKeys(b []byte) []string {
	var v any
	if json.Unmarshal(b, &v) != nil {
		return nil
	}
	var out []string
	var walk func(any)
	walk = func(n any) {
		switch t := n.(type) {
		case map[string]any:
			for k, c := range t {
				switch k {
				case "source":
					if s, ok := c.(string); ok {
						out = append(out, s)
					}
				case "sources":
					if arr, ok := c.([]any); ok {
						for _, x := range arr {
							if s, ok := x.(string); ok {
								out = append(out, s)
							}
						}
					}
				default:
					walk(c)
				}
			}
		case []any:
			for _, c := range t {
				walk(c)
			}
		}
	}
	walk(v)
	sort.Strings(out)
	return out
}

// SourceKeys returns the registry keys in sorted order.
func (ds *Dataset) SourceKeys() []string {
	keys := make([]string, 0, len(ds.Sources))
	for k := range ds.Sources {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func fileBase(ds *Dataset, slug string) string {
	if f, ok := ds.Files[slug]; ok {
		return filepath.Base(f)
	}
	return slug
}

func joinSorted(m map[string]bool) string {
	var s []string
	for k := range m {
		s = append(s, k)
	}
	sort.Strings(s)
	return strings.Join(s, ", ")
}
