package data

import (
	"strings"
	"testing"

	"github.com/brennanMKE/WoodDust/internal/model"
)

// The repo's own data must always pass (warnings allowed).
func TestRepoDataIntegrity(t *testing.T) {
	ds, err := Load("../../data")
	if err != nil {
		t.Fatal(err)
	}
	rep := Check(ds)
	for _, e := range rep.Errors {
		t.Error(e)
	}
	if len(ds.Species) == 0 || len(ds.Sources) == 0 {
		t.Fatal("empty dataset")
	}
	if ds.Summary.NIOSHRELMgM3 == nil || *ds.Summary.NIOSHRELMgM3 != 1 {
		t.Errorf("regulatory summary: %+v", ds.Summary)
	}
}

func fixture() *Dataset {
	return &Dataset{
		Species: []model.Species{
			{ID: "a-a", Slug: "alpha", BotanicalName: "Alpha alpha", CommonNames: []string{"alpha", "shared"},
				Hazards: []model.Hazard{{Type: "carcinogen", Sources: []string{"S1"}}}},
			{ID: "b-b", Slug: "beta", BotanicalName: "Beta beta", CommonNames: []string{"beta", "Shared"},
				Hazards:        []model.Hazard{{Type: "carcinogen", Sources: []string{"S1", "MISSING"}}},
				ExposureLimits: []model.ExposureLimit{{Body: "X", ValueMgM3: 1, Source: "ALSO-MISSING"}}},
		},
		Sources:    map[string]model.Source{"S1": {Key: "S1", Tier: "A", Citation: "c"}},
		Regulatory: []byte(`{"x":{"sources":["S1","REG-MISSING"]}}`),
		Files:      map[string]string{},
	}
}

func TestCheckFindsProblems(t *testing.T) {
	rep := Check(fixture())
	all := strings.Join(rep.Errors, "\n")
	for _, want := range []string{`"MISSING"`, `"ALSO-MISSING"`, `"REG-MISSING"`} {
		if !strings.Contains(all, want) {
			t.Errorf("missing error for %s in:\n%s", want, all)
		}
	}
	if len(rep.Warnings) != 1 || !strings.Contains(rep.Warnings[0], `"shared"`) {
		t.Errorf("want one collision warning, got %v", rep.Warnings)
	}
}

func TestSlugAliasCollisionIsError(t *testing.T) {
	ds := fixture()
	ds.Species[1].CommonNames = append(ds.Species[1].CommonNames, "Alpha")
	rep := Check(ds)
	if !strings.Contains(strings.Join(rep.Errors, "\n"), "is the slug of alpha") {
		t.Errorf("want slug collision error, got %v", rep.Errors)
	}
}

func TestDuplicateFeaturedRank(t *testing.T) {
	ds := fixture()
	one := 1
	ds.Species[0].FeaturedRank, ds.Species[1].FeaturedRank = &one, &one
	if rep := Check(ds); !strings.Contains(strings.Join(rep.Errors, "\n"), "featured_rank 1") {
		t.Errorf("want featured_rank error, got %v", rep.Errors)
	}
}
