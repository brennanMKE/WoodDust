package model

import "testing"

func TestNormalize(t *testing.T) {
	cases := map[string]string{
		"Western Red-Cedar": "western red cedar",
		"Ipê":               "ipe",
		"  thuja-plicata  ": "thuja plicata",
		"Quercus spp.":      "quercus spp",
		"SPF (spruce/pine)": "spf spruce pine",
		"Okoumé":            "okoume",
		"Smørrebrød Ærø":    "smorrebrod aero",
		"---":               "",
	}
	for in, want := range cases {
		if got := Normalize(in); got != want {
			t.Errorf("Normalize(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestSimilarityMatchesPgTrgm(t *testing.T) {
	if s := Similarity("walnut", "walnut"); s != 1 {
		t.Fatalf("identical similarity = %v", s)
	}
	if s := Similarity("walnut", "zzz"); s != 0 {
		t.Fatalf("disjoint similarity = %v", s)
	}
	if s := Similarity("walnut", "wallnut"); s < 0.3 {
		t.Fatalf("typo similarity too low: %v", s)
	}
}

func TestMatchScoreOrdering(t *testing.T) {
	exact, _ := MatchScore("oak", "oak")
	prefix, _ := MatchScore("oakwood", "oak")
	word, _ := MatchScore("red oak", "oak")
	fuzzy, ok := MatchScore("walnut", "wallnut")
	if !ok || !(exact > prefix && prefix > word && word > fuzzy) {
		t.Fatalf("want exact > prefix > word > fuzzy, got %v %v %v %v", exact, prefix, word, fuzzy)
	}
	if _, ok := MatchScore("teak", "zzz"); ok {
		t.Fatal("unrelated name matched")
	}
}

func TestSpeciesHelpers(t *testing.T) {
	one, two := 1, 2
	_ = two
	sp := Species{
		Slug: "western-red-cedar", ID: "thuja-plicata", BotanicalName: "Thuja plicata",
		CommonNames: []string{"western red cedar", "WRC"},
		Hazards: []Hazard{
			{Type: "carcinogen", EvidenceLevel: 0},
			{Type: "respiratory_sensitizer", EvidenceLevel: 2, Sources: []string{"A", "B"}},
			{Type: "respiratory_sensitizer", EvidenceLevel: 1, Sources: []string{"B", "C"}},
		},
		NotesInternal: "x", ImagePrompt: "y",
	}
	if sp.DisplayName() != "Western red cedar" {
		t.Errorf("DisplayName = %q", sp.DisplayName())
	}
	if l := sp.MaxEvidenceLevel(); l == nil || *l != one {
		t.Errorf("MaxEvidenceLevel = %v", l)
	}
	if got := sp.HazardTypes(); len(got) != 2 {
		t.Errorf("HazardTypes = %v", got)
	}
	if got := sp.CitedKeys(); len(got) != 3 || got[0] != "A" || got[2] != "C" {
		t.Errorf("CitedKeys = %v", got)
	}
	if p := sp.Public(); p.NotesInternal != "" || p.ImagePrompt != "" {
		t.Error("Public kept internal fields")
	}
	names := sp.Names()
	// slug duplicates the first common name; id duplicates the botanical name.
	if len(names) != 3 {
		t.Errorf("Names = %v", names)
	}
	only0 := Species{Hazards: []Hazard{{EvidenceLevel: 0}}}
	if only0.MaxEvidenceLevel() != nil {
		t.Error("level-0-only species should have nil max evidence")
	}
}
