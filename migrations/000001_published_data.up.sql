-- Published data, rebuilt from data/ by `wooddust seed` (PLAN §8).
-- Every table here is replaced in one transaction on each seed; nothing else
-- may write to them. Future contributor tables (submissions, users,
-- api_keys) must reference species by slug text, not by foreign key, so a
-- reseed can truncate these tables.

CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE EXTENSION IF NOT EXISTS unaccent;

-- Lowercase, strip diacritics, collapse non-alphanumerics to one space.
-- Must agree with model.Normalize in Go (checked by a DB-backed test).
-- Declared IMMUTABLE (with an explicit dictionary) so it can back a
-- generated column and its indexes.
CREATE FUNCTION wd_normalize(t text) RETURNS text
LANGUAGE sql IMMUTABLE STRICT PARALLEL SAFE
AS $$
  SELECT btrim(regexp_replace(lower(public.unaccent('public.unaccent'::regdictionary, t)), '[^a-z0-9]+', ' ', 'g'))
$$;

CREATE TABLE sources (
    key        text PRIMARY KEY,
    tier       text NOT NULL CHECK (tier IN ('A', 'B', 'C', 'D')),
    citation   text NOT NULL,
    title      text,
    doi        text,
    pmid       text,
    url        text,
    verified   boolean NOT NULL,
    doc        jsonb NOT NULL
);

CREATE TABLE species (
    slug           text PRIMARY KEY,
    id             text NOT NULL UNIQUE,
    display_name   text NOT NULL,
    botanical_name text NOT NULL,
    family         text NOT NULL,
    wood_type      text NOT NULL CHECK (wood_type IN ('hardwood', 'softwood')),
    region         text NOT NULL,
    featured_rank  integer UNIQUE CHECK (featured_rank BETWEEN 1 AND 5),
    last_reviewed  date NOT NULL,
    -- The full data/species/<slug>.json record, internal fields included;
    -- the API strips notes_internal, image_prompt and images.
    doc            jsonb NOT NULL
);

CREATE TABLE species_names (
    id           bigserial PRIMARY KEY,
    species_slug text NOT NULL REFERENCES species (slug) ON DELETE CASCADE,
    name         text NOT NULL,
    kind         text NOT NULL CHECK (kind IN ('common', 'trade', 'botanical', 'synonym')),
    position     integer NOT NULL,
    normalized   text GENERATED ALWAYS AS (wd_normalize(name)) STORED,
    UNIQUE (species_slug, normalized)
);
CREATE INDEX species_names_normalized_trgm ON species_names USING gin (normalized gin_trgm_ops);
CREATE INDEX species_names_normalized_prefix ON species_names (normalized text_pattern_ops);

CREATE TABLE hazards (
    id             bigserial PRIMARY KEY,
    species_slug   text NOT NULL REFERENCES species (slug) ON DELETE CASCADE,
    position       integer NOT NULL,
    type           text NOT NULL,
    evidence_level smallint NOT NULL CHECK (evidence_level BETWEEN 0 AND 3),
    summary        text NOT NULL,
    effects        text[] NOT NULL,
    route          text[] NOT NULL,
    source_part    text[] NOT NULL DEFAULT '{}',
    UNIQUE (species_slug, position)
);
CREATE INDEX hazards_type ON hazards (type);

CREATE TABLE hazard_sources (
    hazard_id  bigint NOT NULL REFERENCES hazards (id) ON DELETE CASCADE,
    source_key text NOT NULL REFERENCES sources (key),
    position   integer NOT NULL,
    PRIMARY KEY (hazard_id, source_key)
);
CREATE INDEX hazard_sources_source ON hazard_sources (source_key);

CREATE TABLE exposure_limits (
    id           bigserial PRIMARY KEY,
    species_slug text NOT NULL REFERENCES species (slug) ON DELETE CASCADE,
    body         text NOT NULL,
    value_mg_m3  numeric NOT NULL,
    fraction     text,
    note         text,
    source_key   text NOT NULL REFERENCES sources (key)
);

CREATE TABLE carcinogen_classes (
    id           bigserial PRIMARY KEY,
    species_slug text NOT NULL REFERENCES species (slug) ON DELETE CASCADE,
    body         text NOT NULL,
    class        text NOT NULL,
    source_key   text NOT NULL REFERENCES sources (key)
);

-- Single row: data/regulatory.json and the derived summary.
CREATE TABLE regulatory (
    id      boolean PRIMARY KEY DEFAULT true CHECK (id),
    summary jsonb NOT NULL,
    doc     jsonb NOT NULL
);

CREATE TABLE seed_runs (
    id            bigserial PRIMARY KEY,
    started_at    timestamptz NOT NULL,
    finished_at   timestamptz NOT NULL DEFAULT now(),
    data_sha256   text NOT NULL,
    species_count integer NOT NULL,
    source_count  integer NOT NULL,
    name_count    integer NOT NULL,
    warnings      jsonb NOT NULL DEFAULT '[]'
);
