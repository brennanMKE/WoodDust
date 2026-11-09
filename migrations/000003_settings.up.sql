-- Operator settings (docs/posters.md "The key in admin Settings").
-- Keys are allowlisted in Go (internal/settings). A secret value is
-- AES-256-GCM under SETTINGS_KEY with the key name as additional data, stored
-- as nonce || ciphertext; only `hint` (a short display prefix) is in clear.
-- A plain value is stored as its UTF-8 bytes.
CREATE TABLE settings (
    key          text PRIMARY KEY,
    value        bytea NOT NULL,
    secret       boolean NOT NULL,
    hint         text NOT NULL DEFAULT '',
    updated_by   text NOT NULL,
    updated_at   timestamptz NOT NULL DEFAULT now(),
    -- Last connection test of the service this setting unlocks (secrets only).
    checked_at   timestamptz,
    check_ok     boolean,
    check_error  text
);

-- Who set, cleared or tested which setting, and when. Never the value.
CREATE TABLE settings_audit (
    id      bigserial PRIMARY KEY,
    at      timestamptz NOT NULL DEFAULT now(),
    key     text NOT NULL,
    action  text NOT NULL CHECK (action IN ('set', 'clear', 'test')),
    actor   text NOT NULL,
    hint    text NOT NULL DEFAULT '',
    detail  text
);
CREATE INDEX settings_audit_key_at ON settings_audit (key, at DESC);
