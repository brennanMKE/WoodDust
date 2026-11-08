-- Contributions, sign-in and API keys (PLAN §5–8, contract §6–7).
-- Nothing here is touched by `wooddust seed`. Species are referred to by slug
-- text, never by foreign key, so a reseed can truncate the published tables.

-- Local authorization for Portiko subjects. Portiko answers "who"; this table
-- answers "may they administer". The first subject to sign in becomes admin.
CREATE TABLE users (
    id            bigserial PRIMARY KEY,
    subject       text NOT NULL UNIQUE,
    display_name  text NOT NULL DEFAULT '',
    is_admin      boolean NOT NULL DEFAULT false,
    is_moderator  boolean NOT NULL DEFAULT false,
    active        boolean NOT NULL DEFAULT true,
    created_at    timestamptz NOT NULL DEFAULT now(),
    last_seen_at  timestamptz NOT NULL DEFAULT now()
);

-- Portiko rp/session sessions for this site. The browser holds the raw id;
-- only its SHA-256 is stored.
CREATE TABLE rp_sessions (
    id_hash          bytea PRIMARY KEY,
    subject          text NOT NULL,
    email            text NOT NULL DEFAULT '',
    name             text NOT NULL DEFAULT '',
    avatar           text NOT NULL DEFAULT '',
    sid              text NOT NULL DEFAULT '',
    profile_version  bigint NOT NULL DEFAULT 0,
    access_token     text NOT NULL,
    refresh_token    text NOT NULL DEFAULT '',
    access_expiry    timestamptz NOT NULL,
    expires_at       timestamptz NOT NULL
);
CREATE INDEX rp_sessions_subject ON rp_sessions (subject);
CREATE INDEX rp_sessions_expires ON rp_sessions (expires_at);

CREATE TABLE submissions (
    id                 bigserial PRIMARY KEY,
    kind               text NOT NULL CHECK (kind IN ('new_species', 'correction', 'new_source', 'reaction_report')),
    status             text NOT NULL DEFAULT 'new'
                       CHECK (status IN ('new', 'triaging', 'needs_info', 'accepted', 'rejected', 'exported', 'merged', 'spam')),
    target_slug        text,
    species_name       text,
    botanical_name     text,
    hazard_type        text,
    details            text NOT NULL,
    citations          jsonb NOT NULL DEFAULT '[]',
    warnings           jsonb NOT NULL DEFAULT '[]',
    -- Optional follow-up address. Nulled when the submission is closed
    -- (rejected, merged, spam); never published.
    email              text,
    -- SHA-256 of the 32-byte status token; the token itself is shown once.
    status_token_hash  bytea NOT NULL UNIQUE,
    -- HMAC of the client IP under a daily-rotating salt. Never the raw IP.
    ip_hash            text,
    spam_reason        text,
    created_at         timestamptz NOT NULL DEFAULT now(),
    updated_at         timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX submissions_status_created ON submissions (status, created_at DESC);
CREATE INDEX submissions_created ON submissions (created_at DESC);

CREATE TABLE submission_events (
    id                    bigserial PRIMARY KEY,
    submission_id         bigint NOT NULL REFERENCES submissions (id) ON DELETE CASCADE,
    at                    timestamptz NOT NULL DEFAULT now(),
    actor                 text NOT NULL,
    kind                  text NOT NULL CHECK (kind IN ('created', 'status', 'note')),
    from_status           text,
    to_status             text,
    note                  text,
    visible_to_submitter  boolean NOT NULL DEFAULT false
);
CREATE INDEX submission_events_submission ON submission_events (submission_id, at);

CREATE TABLE api_keys (
    id             bigserial PRIMARY KEY,
    name           text NOT NULL,
    prefix         text NOT NULL UNIQUE,
    hash           bytea NOT NULL,
    scopes         text[] NOT NULL,
    owner_user_id  bigint REFERENCES users (id) ON DELETE SET NULL,
    created_at     timestamptz NOT NULL DEFAULT now(),
    last_used_at   timestamptz,
    expires_at     timestamptz,
    revoked_at     timestamptz
);

CREATE TABLE audit_log (
    id      bigserial PRIMARY KEY,
    at      timestamptz NOT NULL DEFAULT now(),
    actor   text NOT NULL,
    action  text NOT NULL,
    target  text,
    detail  jsonb
);
CREATE INDEX audit_log_at ON audit_log (at DESC);
