package settings

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrUnknownKey is returned for a key outside the allowlist.
var ErrUnknownKey = errors.New("unknown setting")

// Info is what may be shown about a setting: never a secret value.
type Info struct {
	Key        string     `json:"key"`
	Secret     bool       `json:"secret"`
	Set        bool       `json:"set"`
	Hint       string     `json:"hint"` // secret: display prefix; plain: the value (or the default when unset)
	IsDefault  bool       `json:"is_default"`
	UpdatedBy  string     `json:"set_by,omitempty"`
	UpdatedAt  *time.Time `json:"set_at,omitempty"`
	CheckedAt  *time.Time `json:"checked_at,omitempty"`
	CheckOK    *bool      `json:"check_ok,omitempty"`
	CheckError string     `json:"check_error,omitempty"`
}

// Store reads and writes the settings tables (migration 000003).
type Store struct {
	pool   *pgxpool.Pool
	cipher *Cipher // nil: SETTINGS_KEY unset
}

// NewStore wraps a pool. cipher may be nil.
func NewStore(pool *pgxpool.Pool, c *Cipher) *Store { return &Store{pool: pool, cipher: c} }

// CanSeal reports whether secret settings can be stored and used.
func (s *Store) CanSeal() bool { return s.cipher != nil }

func audit(ctx context.Context, tx pgx.Tx, key, action, actor, hint, detail string) error {
	var d *string
	if detail != "" {
		d = &detail
	}
	_, err := tx.Exec(ctx, `INSERT INTO settings_audit (key, action, actor, hint, detail) VALUES ($1, $2, $3, $4, $5)`,
		key, action, actor, hint, d)
	return err
}

// Set validates, seals (for a secret) and upserts a value, and audits it.
// The connection-test result is reset, because it was for the old value.
func (s *Store) Set(ctx context.Context, key, value, actor string) (Info, error) {
	d, ok := Lookup(key)
	if !ok {
		return Info{}, ErrUnknownKey
	}
	v, err := d.Check(value)
	if err != nil {
		return Info{}, err
	}
	stored := []byte(v)
	if d.Secret {
		if s.cipher == nil {
			return Info{}, ErrNoCipher
		}
		stored = s.cipher.Seal(key, []byte(v))
	}
	hint := d.Hint(v)
	err = pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `INSERT INTO settings (key, value, secret, hint, updated_by, updated_at)
			VALUES ($1, $2, $3, $4, $5, now())
			ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, secret = EXCLUDED.secret, hint = EXCLUDED.hint,
			updated_by = EXCLUDED.updated_by, updated_at = now(), checked_at = NULL, check_ok = NULL, check_error = NULL`,
			key, stored, d.Secret, hint, actor); err != nil {
			return err
		}
		return audit(ctx, tx, key, "set", actor, hint, "")
	})
	if err != nil {
		return Info{}, err
	}
	return s.Info(ctx, key)
}

// Clear removes a setting (a plain one falls back to its default).
func (s *Store) Clear(ctx context.Context, key, actor string) error {
	if _, ok := Lookup(key); !ok {
		return ErrUnknownKey
	}
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		var hint string
		err := tx.QueryRow(ctx, `DELETE FROM settings WHERE key = $1 RETURNING hint`, key).Scan(&hint)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		return audit(ctx, tx, key, "clear", actor, hint, "")
	})
}

// Info describes a setting without revealing a secret.
func (s *Store) Info(ctx context.Context, key string) (Info, error) {
	d, ok := Lookup(key)
	if !ok {
		return Info{}, ErrUnknownKey
	}
	in := Info{Key: key, Secret: d.Secret}
	var at time.Time
	err := s.pool.QueryRow(ctx, `SELECT hint, updated_by, updated_at, checked_at, check_ok, coalesce(check_error, '')
		FROM settings WHERE key = $1`, key).Scan(&in.Hint, &in.UpdatedBy, &at, &in.CheckedAt, &in.CheckOK, &in.CheckError)
	if errors.Is(err, pgx.ErrNoRows) {
		if !d.Secret {
			in.Hint, in.IsDefault = d.Default, true
		}
		return in, nil
	}
	if err != nil {
		return Info{}, err
	}
	in.Set, in.UpdatedAt = true, &at
	return in, nil
}

// Value returns a setting for the server's own use, decrypting a secret.
// A plain setting that is unset returns its default; an unset secret returns
// ok=false.
func (s *Store) Value(ctx context.Context, key string) (string, bool, error) {
	d, ok := Lookup(key)
	if !ok {
		return "", false, ErrUnknownKey
	}
	var raw []byte
	err := s.pool.QueryRow(ctx, `SELECT value FROM settings WHERE key = $1`, key).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) {
		if d.Secret {
			return "", false, nil
		}
		return d.Default, true, nil
	}
	if err != nil {
		return "", false, err
	}
	if !d.Secret {
		return string(raw), true, nil
	}
	if s.cipher == nil {
		return "", false, ErrNoCipher
	}
	plain, err := s.cipher.Open(key, raw)
	if err != nil {
		return "", false, fmt.Errorf("%s: %w", key, err)
	}
	return string(plain), true, nil
}

// RecordCheck stores the result of using the setting (a connection test, or
// a real call). With auditAs non-empty it also writes a "test" audit row.
func (s *Store) RecordCheck(ctx context.Context, key string, ok bool, msg, auditAs string) error {
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		var hint string
		var e *string
		if !ok {
			e = &msg
		}
		err := tx.QueryRow(ctx, `UPDATE settings SET checked_at = now(), check_ok = $2, check_error = $3
			WHERE key = $1 RETURNING hint`, key, ok, e).Scan(&hint)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil || auditAs == "" {
			return err
		}
		detail := "ok"
		if !ok {
			detail = msg
		}
		return audit(ctx, tx, key, "test", auditAs, hint, detail)
	})
}
