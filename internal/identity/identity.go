// Package identity embeds the Portiko identity provider in the site's own
// binary and signs the owner in to the admin console with Portiko's rp and
// rp/session (PLAN §5, docs/portiko-integration.md).
//
// The provider serves only its identity paths on the site's mux; the site
// keeps /, /health and /profile. The site's relying party talks to the
// provider in-process (no network hop, no dependency on Apache at boot).
package identity

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"time"

	portikomigrations "github.com/Kolonnade/Portiko/migrations"
	"github.com/Kolonnade/Portiko/provider"
	"github.com/Kolonnade/Portiko/rp"
	"github.com/Kolonnade/Portiko/rp/session"
	"github.com/Kolonnade/Portiko/server"

	"github.com/brennanMKE/WoodDust/internal/config"
	"github.com/brennanMKE/WoodDust/internal/db"
	"github.com/brennanMKE/WoodDust/internal/middleware"
)

// ProviderPaths are the patterns the embedded provider serves on the site's
// mux. Portiko's router also claims GET /, /health and /profile; those stay
// with the site (see docs/portiko-integration.md).
var ProviderPaths = []string{
	"/oauth/",
	"/accounts/",
	"GET /.well-known/openid-configuration",
	"GET /.well-known/jwks.json",
	"GET /.well-known/portiko-configuration",
	"GET /.well-known/eureka-accounts-configuration",
	"GET /login",
	"POST /login/select",
	"GET /signin",
	"GET /u/",
}

// ProviderCSP replaces the site CSP on provider pages: Portiko's templates
// carry inline <script> and <style> with per-request values, so hashes are
// not possible. Everything else stays same-origin.
const ProviderCSP = "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; " +
	"img-src 'self' data:; connect-src 'self'; object-src 'none'; base-uri 'none'; form-action 'self'; frame-ancestors 'none'"

// ProviderConfig builds Portiko's config from the site's: issuer and origin
// are BASE_URL, the RP ID is its host, TRUSTED_PROXIES is shared.
func ProviderConfig(cfg config.Config) (*provider.Config, error) {
	key, err := decode32(cfg.Identity.CryptoKey)
	if err != nil {
		return nil, fmt.Errorf("IDENTITY_CRYPTO_KEY %w", err)
	}
	trusted, err := middleware.ParseTrustedProxies(cfg.TrustedProxies)
	if err != nil {
		return nil, fmt.Errorf("TRUSTED_PROXIES: %w", err)
	}
	return &provider.Config{
		Issuer:            cfg.BaseURL,
		DatabaseURL:       cfg.Identity.DatabaseURL,
		ServiceName:       cfg.SiteName,
		WebAuthnRPID:      cfg.Host(),
		WebAuthnRPOrigins: []string{cfg.BaseURL},
		SMTPHost:          cfg.Identity.SMTPHost,
		SMTPPort:          cfg.Identity.SMTPPort,
		SMTPUsername:      cfg.Identity.SMTPUsername,
		SMTPPassword:      cfg.Identity.SMTPPassword,
		EmailFrom:         cfg.Identity.EmailFrom,
		TemplateDir:       cfg.Identity.TemplateDir,
		BootstrapCode:     cfg.Identity.BootstrapCode,
		DevInsecure:       cfg.DevInsecure,
		CryptoKey:         key,
		TrustedProxies:    trusted,
	}, nil
}

func decode32(s string) ([32]byte, error) {
	var out [32]byte
	var raw []byte
	if b, err := hex.DecodeString(s); err == nil {
		raw = b
	} else if b, err := base64.StdEncoding.DecodeString(s); err == nil {
		raw = b
	} else {
		return out, errors.New("must be 32 bytes as hex or base64 (openssl rand -hex 32)")
	}
	if len(raw) != 32 {
		return out, fmt.Errorf("must be exactly 32 bytes, got %d", len(raw))
	}
	copy(out[:], raw)
	return out, nil
}

// Migrate applies Portiko's embedded migrations to the identity database.
// Portiko leaves this to the operator; the site does it on every start.
func Migrate(databaseURL string) error {
	return db.MigrateUpFS(databaseURL, portikomigrations.FS)
}

// Options are the site's hooks into sign-in.
type Options struct {
	Sessions session.Store
	// OnSignIn records the person locally (and makes the first one admin).
	OnSignIn func(ctx context.Context, s *session.Session) error
	Logger   *slog.Logger
}

// Identity is the running provider plus the site's relying party.
type Identity struct {
	cfg      config.Config
	provider *provider.Provider
	rp       *rp.Client
	handler  http.Handler
	enroll   *enrollment
	log      *slog.Logger
}

// New migrates the identity database, starts the provider and builds the
// relying party against it.
func New(ctx context.Context, cfg config.Config, o Options) (*Identity, error) {
	if o.Logger == nil {
		o.Logger = slog.Default()
	}
	pcfg, err := ProviderConfig(cfg)
	if err != nil {
		return nil, err
	}
	if err := Migrate(cfg.Identity.DatabaseURL); err != nil {
		return nil, fmt.Errorf("identity migrate: %w", err)
	}
	p, err := provider.New(ctx, pcfg, provider.Options{})
	if err != nil {
		return nil, fmt.Errorf("identity provider: %w", err)
	}
	id := &Identity{cfg: cfg, provider: p, log: o.Logger,
		enroll: newEnrollment(cfg.Identity.BootstrapCode, pcfg.CryptoKey[:], !cfg.DevInsecure)}
	id.handler = id.enroll.gate(p.Handler())

	client, err := rp.New(ctx, rp.Config{
		Issuer:       cfg.BaseURL,
		ClientID:     cfg.Identity.ClientID,
		ClientSecret: cfg.Identity.ClientSecret,
		RedirectURL:  cfg.BaseURL + "/auth/callback",
		Sessions:     o.Sessions,
		Scopes:       []string{"profile"},
		HTTPClient:   &http.Client{Timeout: 10 * time.Second, Transport: inProcess{h: p.Handler(), host: hostOf(cfg.BaseURL)}},
		Insecure:     cfg.DevInsecure,
		CookieKey:    cfg.Identity.CookieKey,
		OnSignIn:     o.OnSignIn,
	})
	if err != nil {
		p.Close()
		return nil, fmt.Errorf("identity relying party: %w", err)
	}
	id.rp = client
	if cfg.Identity.BootstrapCode == "" {
		o.Logger.Warn("IDENTITY_BOOTSTRAP_CODE is unset: passkey enrollment is closed (existing passkeys still sign in)")
	}
	return id, nil
}

func hostOf(base string) string {
	_, rest, _ := strings.Cut(base, "://")
	return rest
}

// inProcess serves requests for the issuer's host through the provider
// handler directly, so the relying party never leaves the process. Other
// hosts use the default transport.
type inProcess struct {
	h    http.Handler
	host string
}

func (t inProcess) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.URL.Host != t.host {
		return http.DefaultTransport.RoundTrip(req)
	}
	r := req.Clone(req.Context())
	r.RemoteAddr = "127.0.0.1:0"
	r.RequestURI = req.URL.RequestURI()
	r.Host = req.URL.Host
	if r.Body == nil {
		r.Body = http.NoBody
	}
	rec := httptest.NewRecorder()
	t.h.ServeHTTP(rec, r)
	res := rec.Result()
	res.Request = req
	return res, nil
}

// Sweep runs the provider's sweeper until ctx ends.
func (i *Identity) Sweep(ctx context.Context) { i.provider.Sweep(ctx) }

// Close releases the provider.
func (i *Identity) Close() { i.provider.Close() }

// Resolve returns the signed-in session, refreshing its tokens if needed.
func (i *Identity) Resolve(r *http.Request) (*session.Session, bool) {
	var got *session.Session
	i.rp.RequireUserAPI(http.HandlerFunc(func(_ http.ResponseWriter, r2 *http.Request) {
		got, _ = rp.SessionFrom(r2.Context())
	})).ServeHTTP(discard{h: http.Header{}}, r)
	return got, got != nil
}

type discard struct{ h http.Header }

func (d discard) Header() http.Header         { return d.h }
func (d discard) Write(b []byte) (int, error) { return len(b), nil }
func (d discard) WriteHeader(int)             {}

// providerPage sets the provider CSP and noindex on its responses.
func providerPage(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy", ProviderCSP)
		w.Header().Set("X-Robots-Tag", "noindex, nofollow")
		h.ServeHTTP(w, r)
	})
}

// Mount registers the provider's paths and the site's /auth/* routes.
func (i *Identity) Mount(mux *http.ServeMux) {
	ph := providerPage(i.handler)
	for _, p := range ProviderPaths {
		mux.Handle(p, ph)
	}
	login := i.rp.LoginHandler()
	mux.Handle("GET /auth/login", noindex(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// The contract uses ?return=; rp reads ?return_to=.
		q := r.URL.Query()
		if v := q.Get("return"); v != "" && q.Get("return_to") == "" {
			q.Set("return_to", v)
			q.Del("return")
			r.URL.RawQuery = q.Encode()
		}
		if q.Get("return_to") == "" {
			q.Set("return_to", "/admin")
			r.URL.RawQuery = q.Encode()
		}
		login.ServeHTTP(w, r)
	})))
	mux.Handle("GET /auth/callback", noindex(i.rp.CallbackHandler()))
	logout := noindex(http.HandlerFunc(i.logout))
	mux.Handle("GET /auth/logout", logout)
	mux.Handle("POST /auth/logout", logout)
	mux.Handle("POST /auth/backchannel-logout", i.rp.BackchannelLogoutHandler())
	mux.Handle("GET /auth/enroll", noindex(http.HandlerFunc(i.enroll.form)))
	mux.Handle("POST /auth/enroll", noindex(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		i.enroll.submit(w, r, i.cfg.BaseURL)
	})))
}

// logout ends the site session and, unlike rp's handler alone, the
// provider's browser session too. Portiko deliberately keeps the provider
// signed in when one site signs out (single sign-on); here the provider
// exists only for this site, and an admin "sign out" that leaves a passkey
// session behind would sign straight back in on the next /auth/login.
func (i *Identity) logout(w http.ResponseWriter, r *http.Request) {
	pr := httptest.NewRequestWithContext(r.Context(), http.MethodPost, "/accounts/v1/logout", http.NoBody)
	pr.RemoteAddr = r.RemoteAddr
	pr.Header.Set("Cookie", r.Header.Get("Cookie"))
	pr.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	i.handler.ServeHTTP(rec, pr)
	for _, c := range rec.Result().Cookies() {
		http.SetCookie(w, c)
	}
	i.rp.LogoutHandler().ServeHTTP(w, r)
}

func noindex(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Robots-Tag", "noindex, nofollow")
		w.Header().Set("Cache-Control", "no-store")
		h.ServeHTTP(w, r)
	})
}

// MountDisabled answers /auth/* with 503 when sign-in is not configured.
func MountDisabled(mux *http.ServeMux, reason string) {
	h := noindex(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(reason + "\n"))
	}))
	mux.Handle("/auth/", h)
}

// ---- enrollment gate ------------------------------------------------------------

// Portiko defines BOOTSTRAP_CODE but (at the pinned commit) never reads it:
// anyone who can receive email could enroll. Only the owner may sign in
// here, so the site gates Portiko's registration endpoints itself. Visiting
// /auth/enroll and entering IDENTITY_BOOTSTRAP_CODE sets a short-lived
// signed cookie; without it, register/start|verify|finish answer 403.
// Existing passkeys sign in without it.
type enrollment struct {
	code   string
	key    []byte
	secure bool
	tries  *middleware.RateLimiter
}

const (
	enrollCookie = "da_enroll"
	enrollTTL    = 30 * time.Minute
)

func newEnrollment(code string, cryptoKey []byte, secure bool) *enrollment {
	m := hmac.New(sha256.New, cryptoKey)
	m.Write([]byte("wooddust enroll cookie\x00" + code))
	return &enrollment{code: code, key: m.Sum(nil), secure: secure, tries: middleware.NewRateLimiter(5, 5)}
}

func (e *enrollment) sign(exp int64) string {
	m := hmac.New(sha256.New, e.key)
	m.Write([]byte(strconv.FormatInt(exp, 10)))
	return strconv.FormatInt(exp, 10) + "." + base64.RawURLEncoding.EncodeToString(m.Sum(nil))
}

func (e *enrollment) valid(r *http.Request) bool {
	if e.code == "" {
		return false
	}
	ck, err := r.Cookie(enrollCookie)
	if err != nil {
		return false
	}
	expStr, _, ok := strings.Cut(ck.Value, ".")
	exp, err := strconv.ParseInt(expStr, 10, 64)
	if !ok || err != nil || time.Now().Unix() > exp {
		return false
	}
	return hmac.Equal([]byte(ck.Value), []byte(e.sign(exp)))
}

// gate refuses Portiko registration without the enrollment cookie.
func (e *enrollment) gate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/accounts/v1/register/") && !e.valid(r) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"error":"registration_closed","error_description":"enrollment needs the bootstrap code at /auth/enroll"}` + "\n"))
			return
		}
		next.ServeHTTP(w, r)
	})
}

var enrollTmpl = template.Must(template.New("enroll").Parse(`<!doctype html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1">
<meta name="robots" content="noindex"><title>Enroll a passkey</title></head>
<body><main>
<h1>Enroll the owner's passkey</h1>
{{if .Closed}}<p>Enrollment is closed: IDENTITY_BOOTSTRAP_CODE is not set on this server.</p>
{{else}}<p>Enter the bootstrap code from the server configuration. You then have 30 minutes to sign in with your email and create a passkey. The first account to sign in becomes the site's admin.</p>
{{if .Error}}<p role="alert">{{.Error}}</p>{{end}}
<form method="post" action="/auth/enroll">
<label for="code">Bootstrap code</label>
<input id="code" name="code" type="password" autocomplete="off" required>
<button type="submit">Continue</button>
</form>{{end}}
<p><a href="/">Back to the site</a></p>
</main></body></html>
`))

func (e *enrollment) render(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_ = enrollTmpl.Execute(w, map[string]any{"Closed": e.code == "", "Error": msg})
}

func (e *enrollment) form(w http.ResponseWriter, r *http.Request) { e.render(w, http.StatusOK, "") }

func (e *enrollment) submit(w http.ResponseWriter, r *http.Request, origin string) {
	if e.code == "" {
		e.render(w, http.StatusForbidden, "")
		return
	}
	if o := r.Header.Get("Origin"); o != "" && !strings.EqualFold(o, origin) {
		http.Error(w, "cross-origin request refused", http.StatusForbidden)
		return
	}
	if ok, _ := e.tries.Allow(middleware.IP(r)); !ok {
		e.render(w, http.StatusTooManyRequests, "Too many attempts. Wait a minute and try again.")
		return
	}
	if subtle.ConstantTimeCompare([]byte(r.PostFormValue("code")), []byte(e.code)) != 1 {
		e.render(w, http.StatusForbidden, "That code is not right.")
		return
	}
	exp := time.Now().Add(enrollTTL).Unix()
	http.SetCookie(w, &http.Cookie{Name: enrollCookie, Value: e.sign(exp), Path: "/accounts/v1/register/",
		MaxAge: int(enrollTTL.Seconds()), HttpOnly: true, Secure: e.secure, SameSite: http.SameSiteStrictMode})
	http.Redirect(w, r, "/auth/login?return=/admin", http.StatusSeeOther)
}

// ---- operator commands ---------------------------------------------------------------

// ServerMain runs a Portiko operator command (server.Main) against the
// identity database, mapping the site's IDENTITY_* configuration onto the
// variables Portiko reads. Every variable is set, even when empty, so a
// stray .env cannot supply the site's own DATABASE_URL or PORT to Portiko.
func ServerMain(cfg config.Config, args []string) int {
	if !cfg.Identity.Enabled() {
		fmt.Fprintln(os.Stderr, "IDENTITY_DATABASE_URL is not set")
		return 2
	}
	if len(args) > 0 && args[0] == "serve" {
		fmt.Fprintln(os.Stderr, "the provider is embedded: run `wooddust serve`")
		return 2
	}
	if len(args) > 0 && args[0] == "migrate" {
		if err := Migrate(cfg.Identity.DatabaseURL); err != nil {
			fmt.Fprintln(os.Stderr, "identity migrate:", err)
			return 1
		}
		fmt.Println("identity database migrated")
		return 0
	}
	if err := Migrate(cfg.Identity.DatabaseURL); err != nil {
		fmt.Fprintln(os.Stderr, "identity migrate:", err)
		return 1
	}
	if len(args) > 0 && args[0] == "register-site" {
		// Shortcut for the one-time registration of this site as the
		// provider's first-party client.
		args = []string{"register-client", "-first-party", cfg.Identity.ClientID, cfg.BaseURL,
			cfg.BaseURL + "/auth/callback", cfg.Identity.ClientSecret}
	}
	insecure := ""
	if cfg.DevInsecure {
		insecure = "true"
	}
	for k, v := range map[string]string{
		"ISSUER":              cfg.BaseURL,
		"DATABASE_URL":        cfg.Identity.DatabaseURL,
		"PORT":                "",
		"SERVICE_NAME":        cfg.SiteName,
		"WEBAUTHN_RP_ID":      cfg.Host(),
		"WEBAUTHN_RP_ORIGINS": cfg.BaseURL,
		"CRYPTO_KEY":          cfg.Identity.CryptoKey,
		"BOOTSTRAP_CODE":      cfg.Identity.BootstrapCode,
		"DEV_INSECURE":        insecure,
		"SMTP_HOST":           cfg.Identity.SMTPHost,
		"SMTP_PORT":           strconv.Itoa(cfg.Identity.SMTPPort),
		"SMTP_USERNAME":       cfg.Identity.SMTPUsername,
		"SMTP_PASSWORD":       cfg.Identity.SMTPPassword,
		"EMAIL_FROM":          cfg.Identity.EmailFrom,
		"TEMPLATE_DIR":        cfg.Identity.TemplateDir,
		"TRUSTED_PROXIES":     cfg.TrustedProxies,
	} {
		_ = os.Setenv(k, v)
	}
	return server.Main("wooddust identity", args, provider.Options{})
}
