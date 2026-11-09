package contrib

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"

	"github.com/Kolonnade/Portiko/rp/session"
)

// SessionResolver returns the signed-in Portiko session for a request, if
// any (refreshing it when needed). Nil means sign-in is disabled.
type SessionResolver func(r *http.Request) (*session.Session, bool)

// Principal is the caller of an authenticated endpoint: an admin's session
// or an API key.
type Principal struct {
	User *User
	Key  *APIKey
}

// Actor names the principal in events and the audit log.
func (p Principal) Actor() string {
	if p.Key != nil {
		return "key:" + p.Key.Prefix
	}
	if p.User != nil {
		return "user:" + p.User.Subject
	}
	return "anonymous"
}

// Users is the slice of the store the auth layer needs (a fake in tests).
type Users interface {
	UserBySubject(ctx context.Context, subject string) (User, error)
	AuthenticateKey(ctx context.Context, key string) (APIKey, error)
}

// Auth decides who may call admin routes (contract §7).
type Auth struct {
	Users   Users           // nil: no database
	Resolve SessionResolver // nil: sign-in disabled
	Origin  string          // BASE_URL, for the CSRF Origin check
}

// Errors written by the auth layer.
const (
	msgSignInDisabled = "Sign-in is disabled on this server (IDENTITY_DATABASE_URL is not set), so the admin console is unavailable. API keys still work for the API."
	msgNoDatabase     = "This server runs without a database (DATABASE_URL is not set), so submissions and admin are unavailable."
)

// Disabled reports why admin sessions cannot work, or "".
func (a *Auth) Disabled() string {
	switch {
	case a.Users == nil:
		return msgNoDatabase
	case a.Resolve == nil:
		return msgSignInDisabled
	}
	return ""
}

// errAuth carries an HTTP status and error code for a refused request.
type errAuth struct {
	status        int
	code, message string
}

func (e *errAuth) Error() string { return e.message }

func refuse(status int, code, msg string) *errAuth { return &errAuth{status, code, msg} }

// sessionUser resolves the admin behind a session.
func (a *Auth) sessionUser(r *http.Request) (*User, *errAuth) {
	if msg := a.Disabled(); msg != "" {
		return nil, refuse(http.StatusServiceUnavailable, "unavailable", msg)
	}
	sess, ok := a.Resolve(r)
	if !ok {
		return nil, refuse(http.StatusUnauthorized, "unauthorized", "Sign in, or send an API key as Authorization: Bearer da_…")
	}
	u, err := a.Users.UserBySubject(r.Context(), sess.Subject)
	if err != nil || !u.Active {
		return nil, refuse(http.StatusForbidden, "forbidden", "This account is not allowed here.")
	}
	return &u, nil
}

// bearer returns the token from an Authorization: Bearer header.
func bearer(r *http.Request) (string, bool) {
	h := r.Header.Get("Authorization")
	if h == "" {
		return "", false
	}
	scheme, tok, _ := strings.Cut(h, " ")
	if !strings.EqualFold(scheme, "Bearer") {
		// Home Assistant users sometimes put the bare key in the header.
		return strings.TrimSpace(h), true
	}
	return strings.TrimSpace(tok), true
}

func unsafeMethod(m string) bool {
	return m != http.MethodGet && m != http.MethodHead && m != http.MethodOptions
}

// sameOrigin checks Origin (or, failing that, Referer) against BASE_URL.
func (a *Auth) sameOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		ref, err := url.Parse(r.Header.Get("Referer"))
		if err != nil || ref.Host == "" {
			return false
		}
		origin = ref.Scheme + "://" + ref.Host
	}
	return strings.EqualFold(strings.TrimRight(origin, "/"), a.Origin)
}

// Authorize resolves the principal for an API request. scope "" means the
// endpoint is session-only (API keys are refused).
func (a *Auth) Authorize(r *http.Request, scope string) (Principal, error) {
	if key, ok := bearer(r); ok {
		if scope == "" {
			return Principal{}, refuse(http.StatusForbidden, "forbidden", "This endpoint needs a signed-in session, not an API key.")
		}
		if a.Users == nil {
			return Principal{}, refuse(http.StatusServiceUnavailable, "unavailable", msgNoDatabase)
		}
		k, err := a.Users.AuthenticateKey(r.Context(), key)
		if err != nil {
			return Principal{}, refuse(http.StatusUnauthorized, "unauthorized", "Invalid, expired or revoked API key.")
		}
		if !k.HasScope(scope) {
			return Principal{}, refuse(http.StatusForbidden, "insufficient_scope", "This API key lacks the "+scope+" scope.")
		}
		return Principal{Key: &k}, nil
	}
	if a.Users != nil && a.Resolve == nil {
		// No session can exist; say what would work.
		if scope == "" {
			return Principal{}, refuse(http.StatusServiceUnavailable, "unavailable", msgSignInDisabled)
		}
		return Principal{}, refuse(http.StatusUnauthorized, "unauthorized", "Send an API key as Authorization: Bearer da_… (sign-in is disabled on this server).")
	}
	u, e := a.sessionUser(r)
	if e != nil {
		return Principal{}, e
	}
	if !u.IsAdmin {
		return Principal{}, refuse(http.StatusForbidden, "forbidden", "Only the site owner can do this.")
	}
	if unsafeMethod(r.Method) && !a.sameOrigin(r) {
		return Principal{}, refuse(http.StatusForbidden, "csrf", "Cross-origin request refused (Origin header must match the site).")
	}
	return Principal{User: u}, nil
}

// writeAuthError writes a refusal from Authorize.
func writeAuthError(w http.ResponseWriter, err error) {
	var e *errAuth
	if !errors.As(err, &e) {
		writeError(w, http.StatusInternalServerError, "internal", "Something went wrong.")
		return
	}
	if e.status == http.StatusUnauthorized {
		w.Header().Set("WWW-Authenticate", `Bearer realm="api"`)
	}
	writeError(w, e.status, e.code, e.message)
}

// PageResult says what an admin page request should do.
type PageResult int

// Page outcomes.
const (
	PageOK PageResult = iota
	PageSignIn
	PageForbidden
	PageUnavailable
)

// PageUser resolves the admin for an /admin page.
func (a *Auth) PageUser(r *http.Request) (*User, PageResult, string) {
	u, e := a.sessionUser(r)
	switch {
	case e == nil && u.IsAdmin:
		return u, PageOK, ""
	case e == nil:
		return nil, PageForbidden, "Only the site owner can open the admin console."
	case e.status == http.StatusUnauthorized:
		return nil, PageSignIn, ""
	case e.status == http.StatusServiceUnavailable:
		return nil, PageUnavailable, e.message
	}
	return nil, PageForbidden, e.message
}

// Me resolves any signed-in user (admin or not) for GET /api/v1/me.
func (a *Auth) Me(r *http.Request) (*User, error) {
	u, e := a.sessionUser(r)
	if e != nil {
		return nil, e
	}
	return u, nil
}

// WriteAuthError writes a refusal from Authorize (for handlers in other packages).
func WriteAuthError(w http.ResponseWriter, err error) { writeAuthError(w, err) }
