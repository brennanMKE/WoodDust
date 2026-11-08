package store

import (
	"context"
	"os"
	"testing"

	"github.com/brennanMKE/WoodDust/internal/data"
	"github.com/brennanMKE/WoodDust/internal/db"
	"github.com/brennanMKE/WoodDust/internal/memstore"
	"github.com/brennanMKE/WoodDust/internal/model"
)

// These tests need a disposable database: TEST_DATABASE_URL, e.g.
// postgres://localhost/wooddust_test?sslmode=disable. They migrate it down
// and up, so never point this at a database you care about.
func setup(t *testing.T) (*Store, *data.Dataset) {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	if err := db.MigrateDown(url, 0); err != nil {
		t.Fatalf("migrate down: %v", err)
	}
	if err := db.MigrateUp(url); err != nil {
		t.Fatalf("migrate up: %v", err)
	}
	pool, err := db.Open(context.Background(), url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	ds, err := data.Load("../../data")
	if err != nil {
		t.Fatal(err)
	}
	return New(pool), ds
}

func TestSeedIdempotentAndQueries(t *testing.T) {
	s, ds := setup(t)
	ctx := context.Background()
	first, err := s.Seed(ctx, ds, nil)
	if err != nil {
		t.Fatal(err)
	}
	second, err := s.Seed(ctx, ds, []string{"w"})
	if err != nil {
		t.Fatal(err)
	}
	if first != second || first.Species != len(ds.Species) || first.Sources != len(ds.Sources) {
		t.Fatalf("seed not idempotent: %+v vs %+v", first, second)
	}
	var runs, hs int
	_ = s.pool.QueryRow(ctx, `SELECT count(*) FROM seed_runs`).Scan(&runs)
	_ = s.pool.QueryRow(ctx, `SELECT count(*) FROM hazard_sources`).Scan(&hs)
	if runs != 2 || hs == 0 {
		t.Errorf("seed_runs=%d hazard_sources=%d", runs, hs)
	}

	list, err := s.ListSpecies(ctx)
	if err != nil || len(list) != len(ds.Species) {
		t.Fatalf("list: %d %v", len(list), err)
	}
	if list[0].FeaturedRank == nil || *list[0].FeaturedRank != 1 {
		t.Errorf("first species should be featured rank 1, got %s", list[0].Slug)
	}
	sp, ok, err := s.GetSpecies(ctx, "oak")
	if err != nil || !ok || sp.NotesInternal == "" {
		t.Errorf("get oak: ok=%v err=%v", ok, err)
	}
	if _, ok, _ := s.GetSpecies(ctx, "nope"); ok {
		t.Error("nope found")
	}

	for name, want := range map[string]string{"wrc": "western-red-cedar", "thuja-plicata": "western-red-cedar", "IPÊ": "ipe"} {
		got, err := s.ResolveName(ctx, name)
		if err != nil || len(got) != 1 || got[0] != want {
			t.Errorf("ResolveName(%q) = %v %v", name, got, err)
		}
	}
	if got, _ := s.ResolveName(ctx, "palisander"); len(got) != 2 {
		t.Errorf("palisander should be ambiguous: %v", got)
	}

	hits, err := s.Search(ctx, "ced", 8)
	if err != nil || len(hits) == 0 {
		t.Fatalf("search ced: %v %v", hits, err)
	}
	for _, h := range hits {
		if h.Score < 0.8 {
			t.Errorf("prefix query returned a non-prefix hit first-page: %+v", h)
		}
	}
	hits, _ = s.Search(ctx, "wallnut", 8)
	if len(hits) == 0 || hits[0].Slug != "walnut" {
		t.Errorf("fuzzy walnut: %+v", hits)
	}
	if hits, _ := s.Search(ctx, "  ", 8); len(hits) != 0 {
		t.Error("blank query should return nothing")
	}

	srcs, err := s.Sources(ctx)
	if err != nil || len(srcs) != len(ds.Sources) || srcs["IARC-62"].Key != "IARC-62" {
		t.Errorf("sources: %d %v", len(srcs), err)
	}
	reg, err := s.Regulatory(ctx)
	if err != nil || reg.NIOSHRELMgM3 == nil {
		t.Errorf("regulatory: %+v %v", reg, err)
	}
}

// SQL search and the in-memory search share one formula; they must agree on
// which species match and in what order for representative queries.
func TestSearchParityWithMemstore(t *testing.T) {
	s, ds := setup(t)
	ctx := context.Background()
	if _, err := s.Seed(ctx, ds, nil); err != nil {
		t.Fatal(err)
	}
	mem := memstore.New(ds)
	for _, q := range []string{"ced", "wrc", "oak", "rosewood", "ipe", "teak", "wallnut", "thuja", "african"} {
		a, err := s.Search(ctx, q, 5)
		if err != nil {
			t.Fatal(err)
		}
		b, _ := mem.Search(ctx, q, 5)
		if len(a) != len(b) {
			t.Errorf("%q: sql %d results, mem %d", q, len(a), len(b))
			continue
		}
		for i := range a {
			if a[i].Slug != b[i].Slug || abs(a[i].Score-b[i].Score) > 0.001 {
				t.Errorf("%q[%d]: sql %s %.3f, mem %s %.3f", q, i, a[i].Slug, a[i].Score, b[i].Slug, b[i].Score)
			}
		}
	}
}

// wd_normalize (SQL) and model.Normalize (Go) must agree on every name.
func TestNormalizeParity(t *testing.T) {
	s, ds := setup(t)
	ctx := context.Background()
	for _, sp := range ds.Species {
		for _, n := range sp.Names() {
			var got string
			if err := s.pool.QueryRow(ctx, `SELECT wd_normalize($1)`, n.Name).Scan(&got); err != nil {
				t.Fatal(err)
			}
			if want := model.Normalize(n.Name); got != want {
				t.Errorf("%q: sql %q, go %q", n.Name, got, want)
			}
		}
	}
}

func abs(f float64) float64 {
	if f < 0 {
		return -f
	}
	return f
}
