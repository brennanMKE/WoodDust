// Package memstore is an in-memory catalog built from a data.Dataset. It is
// the fake behind the handler tests, and `wooddust serve` uses it when
// DATABASE_URL is unset so the site (and the SPA's dev server) can run
// straight from data/ with no Postgres. Its search mirrors the SQL in
// internal/store via model.MatchScore.
package memstore

import (
	"context"
	"sort"
	"strings"

	"github.com/brennanMKE/WoodDust/internal/data"
	"github.com/brennanMKE/WoodDust/internal/model"
)

// Store is a read-only catalog.
type Store struct {
	species []model.Species // featured_rank, then name
	bySlug  map[string]model.Species
	names   map[string][]string // normalized name -> slugs
	sources map[string]model.Source
	summary model.RegulatorySummary
}

// New builds a store from a dataset.
func New(ds *data.Dataset) *Store {
	s := &Store{
		bySlug:  map[string]model.Species{},
		names:   map[string][]string{},
		sources: map[string]model.Source{},
		summary: ds.Summary,
	}
	s.species = append(s.species, ds.Species...)
	sort.SliceStable(s.species, func(i, j int) bool {
		a, b := s.species[i], s.species[j]
		switch {
		case a.FeaturedRank != nil && b.FeaturedRank != nil && *a.FeaturedRank != *b.FeaturedRank:
			return *a.FeaturedRank < *b.FeaturedRank
		case a.FeaturedRank != nil && b.FeaturedRank == nil:
			return true
		case a.FeaturedRank == nil && b.FeaturedRank != nil:
			return false
		}
		an, bn := strings.ToLower(a.DisplayName()), strings.ToLower(b.DisplayName())
		if an != bn {
			return an < bn
		}
		return a.Slug < b.Slug
	})
	for _, sp := range s.species {
		s.bySlug[sp.Slug] = sp
		for _, n := range sp.Names() {
			k := model.Normalize(n.Name)
			s.names[k] = appendUnique(s.names[k], sp.Slug)
		}
	}
	for k, v := range ds.Sources {
		v.Key = k
		s.sources[k] = v
	}
	return s
}

func appendUnique(list []string, v string) []string {
	for _, x := range list {
		if x == v {
			return list
		}
	}
	return append(list, v)
}

// ListSpecies returns every species ordered by featured_rank, then name.
func (s *Store) ListSpecies(context.Context) ([]model.Species, error) {
	return append([]model.Species(nil), s.species...), nil
}

// GetSpecies returns one species by canonical slug.
func (s *Store) GetSpecies(_ context.Context, slug string) (model.Species, bool, error) {
	sp, ok := s.bySlug[slug]
	return sp, ok, nil
}

// ResolveName returns every species slug that has the (normalized) name.
func (s *Store) ResolveName(_ context.Context, name string) ([]string, error) {
	k := model.Normalize(name)
	if k == "" {
		return nil, nil
	}
	out := append([]string(nil), s.names[k]...)
	sort.Strings(out)
	return out, nil
}

// Search ranks species by their best-matching name.
func (s *Store) Search(_ context.Context, q string, limit int) ([]model.SearchHit, error) {
	qn := model.Normalize(q)
	out := []model.SearchHit{}
	if qn == "" {
		return out, nil
	}
	for _, sp := range s.species {
		var best *model.SearchHit
		for _, n := range sp.Names() {
			score, ok := model.MatchScore(model.Normalize(n.Name), qn)
			if !ok || (best != nil && score <= best.Score) {
				continue
			}
			best = &model.SearchHit{
				Slug: sp.Slug, Name: sp.DisplayName(), BotanicalName: sp.BotanicalName,
				MatchedName: n.Name, MatchKind: n.Kind, Score: score, FeaturedRank: sp.FeaturedRank,
			}
		}
		if best != nil {
			out = append(out, *best)
		}
	}
	// s.species is already in featured/name order, so a stable sort by
	// score keeps the same tie-break as the SQL.
	sort.SliceStable(out, func(i, j int) bool { return out[i].Score > out[j].Score })
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

// Sources returns the whole registry.
func (s *Store) Sources(context.Context) (map[string]model.Source, error) {
	out := make(map[string]model.Source, len(s.sources))
	for k, v := range s.sources {
		out[k] = v
	}
	return out, nil
}

// Regulatory returns the site-wide limits summary.
func (s *Store) Regulatory(context.Context) (model.RegulatorySummary, error) {
	return s.summary, nil
}

// Ping always succeeds.
func (s *Store) Ping(context.Context) error { return nil }
