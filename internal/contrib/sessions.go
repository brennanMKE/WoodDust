package contrib

import (
	"context"
	"errors"

	"github.com/Kolonnade/Portiko/rp/session"
	"github.com/jackc/pgx/v5"
)

// SessionStore implements Portiko's session.Store (plus LogoutStore and
// ProfileStore) over the rp_sessions table. Only the SHA-256 of the session
// id is stored.
type SessionStore struct{ st *Store }

// Sessions returns the session store backed by this Store.
func (st *Store) Sessions() *SessionStore { return &SessionStore{st: st} }

var (
	_ session.Store        = (*SessionStore)(nil)
	_ session.LogoutStore  = (*SessionStore)(nil)
	_ session.ProfileStore = (*SessionStore)(nil)
)

// Get implements session.Store.
func (s *SessionStore) Get(ctx context.Context, id string) (*session.Session, error) {
	sess := &session.Session{ID: id}
	err := s.st.pool.QueryRow(ctx, `SELECT subject, email, name, avatar, sid, profile_version, access_token,
		refresh_token, access_expiry, expires_at FROM rp_sessions WHERE id_hash = $1 AND expires_at > now()`, HashToken(id)).
		Scan(&sess.Subject, &sess.Email, &sess.Name, &sess.Avatar, &sess.SID, &sess.ProfileVersion,
			&sess.AccessToken, &sess.RefreshToken, &sess.AccessExpiry, &sess.ExpiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, session.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return sess, nil
}

// Put implements session.Store.
func (s *SessionStore) Put(ctx context.Context, sess *session.Session) error {
	_, err := s.st.pool.Exec(ctx, `INSERT INTO rp_sessions (id_hash, subject, email, name, avatar, sid, profile_version,
		access_token, refresh_token, access_expiry, expires_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		ON CONFLICT (id_hash) DO UPDATE SET subject = EXCLUDED.subject, email = EXCLUDED.email, name = EXCLUDED.name,
		avatar = EXCLUDED.avatar, sid = EXCLUDED.sid, profile_version = EXCLUDED.profile_version,
		access_token = EXCLUDED.access_token, refresh_token = EXCLUDED.refresh_token,
		access_expiry = EXCLUDED.access_expiry, expires_at = EXCLUDED.expires_at`,
		HashToken(sess.ID), sess.Subject, sess.Email, sess.Name, sess.Avatar, sess.SID, sess.ProfileVersion,
		sess.AccessToken, sess.RefreshToken, sess.AccessExpiry, sess.ExpiresAt)
	return err
}

// Delete implements session.Store.
func (s *SessionStore) Delete(ctx context.Context, id string) error {
	_, err := s.st.pool.Exec(ctx, `DELETE FROM rp_sessions WHERE id_hash = $1`, HashToken(id))
	return err
}

// DeleteForLogout implements session.LogoutStore.
func (s *SessionStore) DeleteForLogout(ctx context.Context, subject, sid string) (int, error) {
	tag, err := s.st.pool.Exec(ctx, `DELETE FROM rp_sessions WHERE subject = $1 AND ($2 = '' OR sid = '' OR sid = $2)`, subject, sid)
	return int(tag.RowsAffected()), err
}

// UpdateProfile implements session.ProfileStore.
func (s *SessionStore) UpdateProfile(ctx context.Context, subject, name, avatar string, version int64) (int, error) {
	tag, err := s.st.pool.Exec(ctx, `UPDATE rp_sessions SET name = $2, avatar = $3, profile_version = $4
		WHERE subject = $1 AND profile_version < $4`, subject, name, avatar, version)
	return int(tag.RowsAffected()), err
}

// Sweep deletes expired sessions.
func (s *SessionStore) Sweep(ctx context.Context) (int64, error) {
	tag, err := s.st.pool.Exec(ctx, `DELETE FROM rp_sessions WHERE expires_at <= now()`)
	return tag.RowsAffected(), err
}
