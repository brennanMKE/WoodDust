package contrib

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNotFound is returned for a missing row.
var ErrNotFound = errors.New("not found")

// Store is the Postgres store for submissions, users, sessions, API keys and
// the audit log (migration 000002).
type Store struct {
	pool *pgxpool.Pool
}

// NewStore wraps a pool.
func NewStore(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

// ---- types --------------------------------------------------------------------

// Submission is the full record (admin view).
type Submission struct {
	ID            int64      `json:"id"`
	Reference     string     `json:"reference"`
	Kind          string     `json:"kind"`
	Status        string     `json:"status"`
	TargetSlug    *string    `json:"target_slug"`
	SpeciesName   *string    `json:"species_name"`
	BotanicalName *string    `json:"botanical_name"`
	HazardType    *string    `json:"hazard_type"`
	Details       string     `json:"details"`
	Citations     []Citation `json:"citations"`
	Warnings      []string   `json:"warnings"`
	Email         *string    `json:"email"`
	IPHash        *string    `json:"ip_hash"`
	SpamReason    *string    `json:"spam_reason,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// Event is one submission_events row.
type Event struct {
	ID                 int64     `json:"id"`
	At                 time.Time `json:"at"`
	Actor              string    `json:"actor"`
	Kind               string    `json:"type"`
	FromStatus         *string   `json:"from_status,omitempty"`
	ToStatus           *string   `json:"to_status,omitempty"`
	Note               *string   `json:"note,omitempty"`
	VisibleToSubmitter bool      `json:"visible_to_submitter"`
}

// Reference formats a submission id as DA-0042.
func Reference(id int64) string { return fmt.Sprintf("DA-%04d", id) }

// NewSubmission is what CreateSubmission stores.
type NewSubmission struct {
	Request
	Warnings   []string
	StatusHash []byte
	IPHash     string
	Spam       string // non-empty: stored as spam with this reason
}

// User is a local users row.
type User struct {
	ID          int64     `json:"id"`
	Subject     string    `json:"subject"`
	DisplayName string    `json:"display_name"`
	IsAdmin     bool      `json:"is_admin"`
	Active      bool      `json:"active"`
	CreatedAt   time.Time `json:"created_at"`
	LastSeenAt  time.Time `json:"last_seen_at"`
}

// APIKey is an api_keys row (never the key itself).
type APIKey struct {
	ID         int64      `json:"-"`
	Name       string     `json:"name"`
	Prefix     string     `json:"prefix"`
	Scopes     []string   `json:"scopes"`
	CreatedAt  time.Time  `json:"created_at"`
	LastUsedAt *time.Time `json:"last_used_at"`
	ExpiresAt  *time.Time `json:"expires_at"`
	RevokedAt  *time.Time `json:"revoked_at"`
	hash       []byte
}

// Active reports whether the key may be used at t.
func (k APIKey) Active(t time.Time) bool {
	return k.RevokedAt == nil && (k.ExpiresAt == nil || t.Before(*k.ExpiresAt))
}

// HasScope reports whether the key carries scope.
func (k APIKey) HasScope(s string) bool { return contains(k.Scopes, s) }

// ---- submissions -------------------------------------------------------------

func nullable(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

const submissionCols = `id, kind, status, target_slug, species_name, botanical_name, hazard_type, details,
	citations, warnings, email, ip_hash, spam_reason, created_at, updated_at`

func scanSubmission(row pgx.Row) (Submission, error) {
	var s Submission
	var cites, warns []byte
	err := row.Scan(&s.ID, &s.Kind, &s.Status, &s.TargetSlug, &s.SpeciesName, &s.BotanicalName, &s.HazardType,
		&s.Details, &cites, &warns, &s.Email, &s.IPHash, &s.SpamReason, &s.CreatedAt, &s.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return s, ErrNotFound
	}
	if err != nil {
		return s, err
	}
	_ = json.Unmarshal(cites, &s.Citations)
	_ = json.Unmarshal(warns, &s.Warnings)
	if s.Citations == nil {
		s.Citations = []Citation{}
	}
	if s.Warnings == nil {
		s.Warnings = []string{}
	}
	s.Reference = Reference(s.ID)
	return s, nil
}

// CreateSubmission stores a submission and its "created" event.
func (st *Store) CreateSubmission(ctx context.Context, n NewSubmission) (Submission, error) {
	cites, _ := json.Marshal(n.Citations)
	if n.Citations == nil {
		cites = []byte("[]")
	}
	warns, _ := json.Marshal(n.Warnings)
	if n.Warnings == nil {
		warns = []byte("[]")
	}
	status, email := "new", nullable(n.Email)
	if n.Spam != "" {
		status, email = "spam", nil
	}
	var out Submission
	err := pgx.BeginFunc(ctx, st.pool, func(tx pgx.Tx) error {
		var err error
		out, err = scanSubmission(tx.QueryRow(ctx, `INSERT INTO submissions
			(kind, status, target_slug, species_name, botanical_name, hazard_type, details, citations, warnings,
			 email, status_token_hash, ip_hash, spam_reason)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13) RETURNING `+submissionCols,
			n.Kind, status, nullable(n.TargetSlug), nullable(n.SpeciesName), nullable(n.BotanicalName),
			nullable(n.HazardType), n.Details, cites, warns, email, n.StatusHash, nullable(n.IPHash), nullable(n.Spam)))
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `INSERT INTO submission_events (submission_id, actor, kind, to_status) VALUES ($1, 'submitter', 'created', $2)`,
			out.ID, status)
		return err
	})
	return out, err
}

// Note is a submitter-visible note on the status page.
type Note struct {
	At   time.Time `json:"at"`
	Text string    `json:"text"`
}

// SubmissionByToken finds a submission by its status token hash, with the
// notes visible to the submitter.
func (st *Store) SubmissionByToken(ctx context.Context, hash []byte) (Submission, []Note, error) {
	s, err := scanSubmission(st.pool.QueryRow(ctx, `SELECT `+submissionCols+` FROM submissions WHERE status_token_hash = $1`, hash))
	if err != nil {
		return s, nil, err
	}
	rows, err := st.pool.Query(ctx, `SELECT at, note FROM submission_events
		WHERE submission_id = $1 AND kind = 'note' AND visible_to_submitter ORDER BY at, id`, s.ID)
	if err != nil {
		return s, nil, err
	}
	notes, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (Note, error) {
		var n Note
		return n, r.Scan(&n.At, &n.Text)
	})
	if notes == nil {
		notes = []Note{}
	}
	return s, notes, err
}

// ListFilter selects submissions for the admin list.
type ListFilter struct {
	Status string // empty: everything except spam
	Kind   string
	Since  *time.Time
	Limit  int
	Cursor int64 // return ids below this
}

// ListSubmissions returns newest first, and the cursor for the next page (0
// when there is none).
func (st *Store) ListSubmissions(ctx context.Context, f ListFilter) ([]Submission, int64, error) {
	var where []string
	var args []any
	arg := func(v any) string { args = append(args, v); return "$" + strconv.Itoa(len(args)) }
	if f.Status == "" {
		where = append(where, "status <> 'spam'")
	} else {
		where = append(where, "status = "+arg(f.Status))
	}
	if f.Kind != "" {
		where = append(where, "kind = "+arg(f.Kind))
	}
	if f.Since != nil {
		where = append(where, "created_at >= "+arg(*f.Since))
	}
	if f.Cursor > 0 {
		where = append(where, "id < "+arg(f.Cursor))
	}
	q := `SELECT ` + submissionCols + ` FROM submissions WHERE ` + strings.Join(where, " AND ") +
		` ORDER BY id DESC LIMIT ` + arg(f.Limit+1)
	rows, err := st.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []Submission{}
	for rows.Next() {
		s, err := scanSubmission(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, s)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	var next int64
	if len(out) > f.Limit {
		out = out[:f.Limit]
		next = out[len(out)-1].ID
	}
	return out, next, nil
}

// GetSubmission returns one submission and its events.
func (st *Store) GetSubmission(ctx context.Context, id int64) (Submission, []Event, error) {
	s, err := scanSubmission(st.pool.QueryRow(ctx, `SELECT `+submissionCols+` FROM submissions WHERE id = $1`, id))
	if err != nil {
		return s, nil, err
	}
	rows, err := st.pool.Query(ctx, `SELECT id, at, actor, kind, from_status, to_status, note, visible_to_submitter
		FROM submission_events WHERE submission_id = $1 ORDER BY at, id`, id)
	if err != nil {
		return s, nil, err
	}
	evs, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (Event, error) {
		var e Event
		return e, r.Scan(&e.ID, &e.At, &e.Actor, &e.Kind, &e.FromStatus, &e.ToStatus, &e.Note, &e.VisibleToSubmitter)
	})
	if evs == nil {
		evs = []Event{}
	}
	return s, evs, err
}

// Update is a PATCH to a submission.
type Update struct {
	Status *string
	Note   *struct {
		Text               string `json:"text"`
		VisibleToSubmitter bool   `json:"visible_to_submitter"`
	}
}

// UpdateSubmission applies a status change and/or note, records events and
// an audit entry, and nulls the email when the submission is closed.
func (st *Store) UpdateSubmission(ctx context.Context, id int64, actor string, u Update) (Submission, error) {
	var out Submission
	err := pgx.BeginFunc(ctx, st.pool, func(tx pgx.Tx) error {
		cur, err := scanSubmission(tx.QueryRow(ctx, `SELECT `+submissionCols+` FROM submissions WHERE id = $1 FOR UPDATE`, id))
		if err != nil {
			return err
		}
		detail := map[string]any{}
		if u.Status != nil && *u.Status != cur.Status {
			if _, err := tx.Exec(ctx, `UPDATE submissions SET status = $2, updated_at = now(),
				email = CASE WHEN $3 THEN NULL ELSE email END WHERE id = $1`, id, *u.Status, closedStatuses[*u.Status]); err != nil {
				return err
			}
			if _, err := tx.Exec(ctx, `INSERT INTO submission_events (submission_id, actor, kind, from_status, to_status)
				VALUES ($1, $2, 'status', $3, $4)`, id, actor, cur.Status, *u.Status); err != nil {
				return err
			}
			detail["from"], detail["to"] = cur.Status, *u.Status
		}
		if u.Note != nil {
			if _, err := tx.Exec(ctx, `UPDATE submissions SET updated_at = now() WHERE id = $1`, id); err != nil {
				return err
			}
			if _, err := tx.Exec(ctx, `INSERT INTO submission_events (submission_id, actor, kind, note, visible_to_submitter)
				VALUES ($1, $2, 'note', $3, $4)`, id, actor, u.Note.Text, u.Note.VisibleToSubmitter); err != nil {
				return err
			}
			detail["note_visible"] = u.Note.VisibleToSubmitter
		}
		if len(detail) > 0 {
			if err := audit(ctx, tx, actor, "submission.update", Reference(id), detail); err != nil {
				return err
			}
		}
		out, err = scanSubmission(tx.QueryRow(ctx, `SELECT `+submissionCols+` FROM submissions WHERE id = $1`, id))
		return err
	})
	return out, err
}

// Stats is GET /api/v1/stats (contract §7).
type Stats struct {
	Open        int            `json:"open"`
	ByStatus    map[string]int `json:"by_status"`
	ByKind      map[string]int `json:"by_kind"`
	NewestAt    *time.Time     `json:"newest_at"`
	NewLast24h  int            `json:"new_last_24h"`
	SpamLast24h int            `json:"spam_last_24h"`
}

// openStatuses still need the owner's attention.
const openSQL = `status NOT IN ('rejected', 'merged', 'spam')`

// Stats counts submissions. "open" is everything not rejected, merged or
// spam; by_kind counts open submissions only.
func (st *Store) Stats(ctx context.Context) (Stats, error) {
	s := Stats{ByStatus: map[string]int{}, ByKind: map[string]int{}}
	for _, v := range Statuses {
		s.ByStatus[v] = 0
	}
	for _, k := range Kinds {
		s.ByKind[k] = 0
	}
	rows, err := st.pool.Query(ctx, `SELECT status, kind, `+openSQL+`, count(*) FROM submissions GROUP BY 1, 2, 3`)
	if err != nil {
		return s, err
	}
	defer rows.Close()
	for rows.Next() {
		var status, kind string
		var open bool
		var n int
		if err := rows.Scan(&status, &kind, &open, &n); err != nil {
			return s, err
		}
		s.ByStatus[status] += n
		if open {
			s.Open += n
			s.ByKind[kind] += n
		}
	}
	if err := rows.Err(); err != nil {
		return s, err
	}
	err = st.pool.QueryRow(ctx, `SELECT
		max(created_at) FILTER (WHERE status <> 'spam'),
		count(*) FILTER (WHERE status <> 'spam' AND created_at > now() - interval '24 hours'),
		count(*) FILTER (WHERE status = 'spam' AND created_at > now() - interval '24 hours')
		FROM submissions`).Scan(&s.NewestAt, &s.NewLast24h, &s.SpamLast24h)
	return s, err
}

// RecentTimes returns creation times in the last 24 hours, oldest first, to
// seed the global submission ceiling after a restart.
func (st *Store) RecentTimes(ctx context.Context) ([]time.Time, error) {
	rows, err := st.pool.Query(ctx, `SELECT created_at FROM submissions WHERE created_at > now() - interval '24 hours' ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowTo[time.Time])
}

// ---- audit ------------------------------------------------------------------------

type execer interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

func audit(ctx context.Context, db execer, actor, action, target string, detail any) error {
	b, _ := json.Marshal(detail)
	_, err := db.Exec(ctx, `INSERT INTO audit_log (actor, action, target, detail) VALUES ($1, $2, $3, $4)`, actor, action, nullable(target), b)
	return err
}

// Audit records one admin action.
func (st *Store) Audit(ctx context.Context, actor, action, target string, detail any) error {
	return audit(ctx, st.pool, actor, action, target, detail)
}

// ---- users ----------------------------------------------------------------------------

const userCols = `id, subject, display_name, is_admin, active, created_at, last_seen_at`

func scanUser(row pgx.Row) (User, error) {
	var u User
	err := row.Scan(&u.ID, &u.Subject, &u.DisplayName, &u.IsAdmin, &u.Active, &u.CreatedAt, &u.LastSeenAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return u, ErrNotFound
	}
	return u, err
}

// SignedIn records a sign-in. The first subject ever recorded becomes admin;
// everyone after is a plain user (403 on admin routes) until promoted with
// `wooddust users admin <subject>`.
func (st *Store) SignedIn(ctx context.Context, subject, name string) (User, error) {
	var u User
	err := pgx.BeginFunc(ctx, st.pool, func(tx pgx.Tx) error {
		// Serialize first sign-ins so two cannot both become the first admin.
		if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(4207001)`); err != nil {
			return err
		}
		var err error
		u, err = scanUser(tx.QueryRow(ctx, `INSERT INTO users (subject, display_name, is_admin)
			VALUES ($1, $2, NOT EXISTS (SELECT 1 FROM users))
			ON CONFLICT (subject) DO UPDATE SET display_name = CASE WHEN EXCLUDED.display_name <> '' THEN EXCLUDED.display_name ELSE users.display_name END,
				last_seen_at = now()
			RETURNING `+userCols, subject, name))
		if err != nil {
			return err
		}
		return audit(ctx, tx, "user:"+subject, "user.sign_in", subject, map[string]any{"is_admin": u.IsAdmin})
	})
	return u, err
}

// UserBySubject looks a user up.
func (st *Store) UserBySubject(ctx context.Context, subject string) (User, error) {
	return scanUser(st.pool.QueryRow(ctx, `SELECT `+userCols+` FROM users WHERE subject = $1`, subject))
}

// Users lists every user.
func (st *Store) Users(ctx context.Context) ([]User, error) {
	rows, err := st.pool.Query(ctx, `SELECT `+userCols+` FROM users ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []User
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

// SetAdmin grants or removes admin.
func (st *Store) SetAdmin(ctx context.Context, subject string, admin bool, actor string) error {
	tag, err := st.pool.Exec(ctx, `UPDATE users SET is_admin = $2 WHERE subject = $1`, subject, admin)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return st.Audit(ctx, actor, "user.set_admin", subject, map[string]any{"is_admin": admin})
}

// ---- API keys ----------------------------------------------------------------------------

const keyCols = `id, name, prefix, hash, scopes, created_at, last_used_at, expires_at, revoked_at`

func scanKey(row pgx.Row) (APIKey, error) {
	var k APIKey
	err := row.Scan(&k.ID, &k.Name, &k.Prefix, &k.hash, &k.Scopes, &k.CreatedAt, &k.LastUsedAt, &k.ExpiresAt, &k.RevokedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return k, ErrNotFound
	}
	return k, err
}

// CreateAPIKey stores a new key and returns it with the full key, which is
// never stored and cannot be shown again.
func (st *Store) CreateAPIKey(ctx context.Context, name string, scopes []string, expires *time.Time, ownerID *int64, actor string) (APIKey, string, error) {
	key, prefix, hash := NewAPIKey()
	var k APIKey
	err := pgx.BeginFunc(ctx, st.pool, func(tx pgx.Tx) error {
		var err error
		k, err = scanKey(tx.QueryRow(ctx, `INSERT INTO api_keys (name, prefix, hash, scopes, owner_user_id, expires_at)
			VALUES ($1, $2, $3, $4, $5, $6) RETURNING `+keyCols, name, prefix, hash, scopes, ownerID, expires))
		if err != nil {
			return err
		}
		return audit(ctx, tx, actor, "api_key.create", prefix, map[string]any{"name": name, "scopes": scopes})
	})
	return k, key, err
}

// APIKeys lists every key, newest first.
func (st *Store) APIKeys(ctx context.Context) ([]APIKey, error) {
	rows, err := st.pool.Query(ctx, `SELECT `+keyCols+` FROM api_keys ORDER BY id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []APIKey{}
	for rows.Next() {
		k, err := scanKey(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, k)
	}
	return out, rows.Err()
}

// RevokeAPIKey revokes by prefix.
func (st *Store) RevokeAPIKey(ctx context.Context, prefix, actor string) error {
	return pgx.BeginFunc(ctx, st.pool, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `UPDATE api_keys SET revoked_at = now() WHERE prefix = $1 AND revoked_at IS NULL`, prefix)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return ErrNotFound
		}
		return audit(ctx, tx, actor, "api_key.revoke", prefix, nil)
	})
}

// AuthenticateKey returns the active key matching the presented key and
// records its use. The hash is compared in constant time.
func (st *Store) AuthenticateKey(ctx context.Context, key string) (APIKey, error) {
	prefix, ok := ParseAPIKey(key)
	if !ok {
		return APIKey{}, ErrNotFound
	}
	k, err := scanKey(st.pool.QueryRow(ctx, `SELECT `+keyCols+` FROM api_keys WHERE prefix = $1`, prefix))
	if err != nil {
		// Spend the same compare on a miss so timing does not reveal prefixes.
		KeyMatches(key, make([]byte, 32))
		return k, err
	}
	if !KeyMatches(key, k.hash) || !k.Active(time.Now()) {
		return APIKey{}, ErrNotFound
	}
	_, _ = st.pool.Exec(ctx, `UPDATE api_keys SET last_used_at = now() WHERE id = $1`, k.ID)
	return k, nil
}
