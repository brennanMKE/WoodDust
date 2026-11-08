// Package store is the Postgres-backed catalog of published data. It owns
// the SQL for the tables in migrations/000001 and the seed that rebuilds them.
package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/brennanMKE/WoodDust/internal/data"
	"github.com/brennanMKE/WoodDust/internal/model"
)

// Store reads and seeds the published-data tables.
type Store struct {
	pool *pgxpool.Pool
}

// New wraps a pool.
func New(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

// ListSpecies returns every species ordered by featured_rank, then name.
func (s *Store) ListSpecies(ctx context.Context) ([]model.Species, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT doc FROM species
		ORDER BY featured_rank NULLS LAST, lower(display_name), slug`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.Species
	for rows.Next() {
		var b []byte
		if err := rows.Scan(&b); err != nil {
			return nil, err
		}
		var sp model.Species
		if err := json.Unmarshal(b, &sp); err != nil {
			return nil, err
		}
		out = append(out, sp)
	}
	return out, rows.Err()
}

// GetSpecies returns one species by canonical slug.
func (s *Store) GetSpecies(ctx context.Context, slug string) (model.Species, bool, error) {
	var b []byte
	err := s.pool.QueryRow(ctx, `SELECT doc FROM species WHERE slug = $1`, slug).Scan(&b)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.Species{}, false, nil
	}
	if err != nil {
		return model.Species{}, false, err
	}
	var sp model.Species
	if err := json.Unmarshal(b, &sp); err != nil {
		return model.Species{}, false, err
	}
	return sp, true, nil
}

// ResolveName returns the slugs of every species that has name (compared
// normalized, so "thuja-plicata" finds "Thuja plicata").
func (s *Store) ResolveName(ctx context.Context, name string) ([]string, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT DISTINCT species_slug FROM species_names
		WHERE normalized = wd_normalize($1) AND normalized <> ''
		ORDER BY species_slug`, name)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowTo[string])
}

// Search ranks species by their best-matching name. The scoring formula is
// model.MatchScore; keep the two in step.
func (s *Store) Search(ctx context.Context, q string, limit int) ([]model.SearchHit, error) {
	if model.Normalize(q) == "" {
		return []model.SearchHit{}, nil
	}
	rows, err := s.pool.Query(ctx, `
		WITH q AS (SELECT wd_normalize($1) AS q),
		scored AS (
			SELECT n.species_slug, n.name, n.kind, n.position,
				CASE
					WHEN n.normalized = q.q THEN 1.0
					WHEN n.normalized LIKE q.q || '%' THEN 0.9 + 0.09 * length(q.q)::float8 / length(n.normalized)
					WHEN n.normalized LIKE '% ' || q.q || '%' THEN 0.8 + 0.09 * length(q.q)::float8 / length(n.normalized)
					ELSE similarity(n.normalized, q.q) * 0.79
				END AS score
			FROM species_names n, q
			WHERE n.normalized LIKE q.q || '%'
			   OR n.normalized LIKE '% ' || q.q || '%'
			   OR n.normalized % q.q
		),
		best AS (
			SELECT DISTINCT ON (species_slug) species_slug, name, kind, score
			FROM scored
			ORDER BY species_slug, score DESC, position
		)
		SELECT b.species_slug, sp.display_name, sp.botanical_name, b.name, b.kind, b.score, sp.featured_rank
		FROM best b JOIN species sp ON sp.slug = b.species_slug
		ORDER BY b.score DESC, sp.featured_rank NULLS LAST, lower(sp.display_name)
		LIMIT $2`, q, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.SearchHit{}
	for rows.Next() {
		var h model.SearchHit
		if err := rows.Scan(&h.Slug, &h.Name, &h.BotanicalName, &h.MatchedName, &h.MatchKind, &h.Score, &h.FeaturedRank); err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	return out, rows.Err()
}

// Sources returns the whole registry.
func (s *Store) Sources(ctx context.Context) (map[string]model.Source, error) {
	rows, err := s.pool.Query(ctx, `SELECT key, doc FROM sources`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]model.Source{}
	for rows.Next() {
		var key string
		var b []byte
		if err := rows.Scan(&key, &b); err != nil {
			return nil, err
		}
		var src model.Source
		if err := json.Unmarshal(b, &src); err != nil {
			return nil, err
		}
		src.Key = key
		out[key] = src
	}
	return out, rows.Err()
}

// Regulatory returns the site-wide limits summary (zero value if unseeded).
func (s *Store) Regulatory(ctx context.Context) (model.RegulatorySummary, error) {
	var sum model.RegulatorySummary
	var b []byte
	err := s.pool.QueryRow(ctx, `SELECT summary FROM regulatory`).Scan(&b)
	if errors.Is(err, pgx.ErrNoRows) {
		return sum, nil
	}
	if err != nil {
		return sum, err
	}
	return sum, json.Unmarshal(b, &sum)
}

// Ping checks the database.
func (s *Store) Ping(ctx context.Context) error { return s.pool.Ping(ctx) }

// SeedResult summarizes one seed run.
type SeedResult struct {
	Species, Sources, Names, Hazards int
}

// Seed replaces every published-data table with ds in one transaction. It is
// idempotent: seeding the same data twice leaves the same rows. A
// transaction-scoped advisory lock serializes concurrent seeds.
func (s *Store) Seed(ctx context.Context, ds *data.Dataset, warnings []string) (SeedResult, error) {
	var res SeedResult
	started := time.Now()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return res, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op after commit

	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(7713001)`); err != nil {
		return res, err
	}
	if _, err := tx.Exec(ctx, `TRUNCATE hazard_sources, hazards, species_names, exposure_limits,
		carcinogen_classes, species, sources, regulatory RESTART IDENTITY`); err != nil {
		return res, fmt.Errorf("truncate: %w", err)
	}

	for _, key := range ds.SourceKeys() {
		src := ds.Sources[key]
		doc, _ := json.Marshal(src)
		if _, err := tx.Exec(ctx, `
			INSERT INTO sources (key, tier, citation, title, doi, pmid, url, verified, doc)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
			key, src.Tier, src.Citation, nullable(src.Title), nullable(src.DOI), nullable(src.PMID),
			nullable(src.URL), src.Verified, doc); err != nil {
			return res, fmt.Errorf("source %s: %w", key, err)
		}
		res.Sources++
	}

	for _, sp := range ds.Species {
		doc, _ := json.Marshal(sp)
		if _, err := tx.Exec(ctx, `
			INSERT INTO species (slug, id, display_name, botanical_name, family, wood_type, region,
				featured_rank, last_reviewed, doc)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
			sp.Slug, sp.ID, sp.DisplayName(), sp.BotanicalName, sp.Family, sp.WoodType, sp.Region,
			sp.FeaturedRank, sp.LastReviewed, doc); err != nil {
			return res, fmt.Errorf("species %s: %w", sp.Slug, err)
		}
		res.Species++

		for i, n := range sp.Names() {
			tag, err := tx.Exec(ctx, `
				INSERT INTO species_names (species_slug, name, kind, position)
				VALUES ($1, $2, $3, $4) ON CONFLICT (species_slug, normalized) DO NOTHING`,
				sp.Slug, n.Name, n.Kind, i)
			if err != nil {
				return res, fmt.Errorf("species %s name %q: %w", sp.Slug, n.Name, err)
			}
			res.Names += int(tag.RowsAffected())
		}

		for i, h := range sp.Hazards {
			var id int64
			if err := tx.QueryRow(ctx, `
				INSERT INTO hazards (species_slug, position, type, evidence_level, summary, effects, route, source_part)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id`,
				sp.Slug, i, h.Type, h.EvidenceLevel, h.Summary, nonNil(h.Effects), nonNil(h.Route),
				nonNil(h.SourcePart)).Scan(&id); err != nil {
				return res, fmt.Errorf("species %s hazard %d: %w", sp.Slug, i, err)
			}
			res.Hazards++
			for j, key := range h.Sources {
				if _, err := tx.Exec(ctx, `
					INSERT INTO hazard_sources (hazard_id, source_key, position) VALUES ($1, $2, $3)
					ON CONFLICT DO NOTHING`, id, key, j); err != nil {
					return res, fmt.Errorf("species %s hazard %d source %s: %w", sp.Slug, i, key, err)
				}
			}
		}
		for _, e := range sp.ExposureLimits {
			if _, err := tx.Exec(ctx, `
				INSERT INTO exposure_limits (species_slug, body, value_mg_m3, fraction, note, source_key)
				VALUES ($1, $2, $3, $4, $5, $6)`,
				sp.Slug, e.Body, e.ValueMgM3, nullable(e.Fraction), nullable(e.Note), e.Source); err != nil {
				return res, fmt.Errorf("species %s exposure limit: %w", sp.Slug, err)
			}
		}
		for _, c := range sp.CarcinogenClass {
			if _, err := tx.Exec(ctx, `
				INSERT INTO carcinogen_classes (species_slug, body, class, source_key) VALUES ($1, $2, $3, $4)`,
				sp.Slug, c.Body, c.Class, c.Source); err != nil {
				return res, fmt.Errorf("species %s carcinogen class: %w", sp.Slug, err)
			}
		}
	}

	summary, _ := json.Marshal(ds.Summary)
	if _, err := tx.Exec(ctx, `INSERT INTO regulatory (summary, doc) VALUES ($1, $2)`, summary, []byte(ds.Regulatory)); err != nil {
		return res, fmt.Errorf("regulatory: %w", err)
	}
	if warnings == nil {
		warnings = []string{}
	}
	warn, _ := json.Marshal(warnings)
	if _, err := tx.Exec(ctx, `
		INSERT INTO seed_runs (started_at, data_sha256, species_count, source_count, name_count, warnings)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		started, ds.SHA256, res.Species, res.Sources, res.Names, warn); err != nil {
		return res, fmt.Errorf("seed_runs: %w", err)
	}
	return res, tx.Commit(ctx)
}

func nullable(v string) any {
	if v == "" {
		return nil
	}
	return v
}

func nonNil(v []string) []string {
	if v == nil {
		return []string{}
	}
	return v
}
