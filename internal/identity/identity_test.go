package identity

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Kolonnade/Portiko/rp/session"
	"github.com/descope/virtualwebauthn"
	"github.com/jackc/pgx/v5"

	"github.com/brennanMKE/WoodDust/internal/config"
	"github.com/brennanMKE/WoodDust/internal/content"
	"github.com/brennanMKE/WoodDust/internal/contrib"
	"github.com/brennanMKE/WoodDust/internal/data"
	"github.com/brennanMKE/WoodDust/internal/db"
	"github.com/brennanMKE/WoodDust/internal/memstore"
	"github.com/brennanMKE/WoodDust/internal/middleware"
	"github.com/brennanMKE/WoodDust/internal/site"
)

// TestSignInToAdmin drives the real embedded provider with a virtual
// authenticator: enrollment is gated by the bootstrap code, the first person
// to sign in becomes admin, a second person gets 403, and session writes
// need a same-origin Origin header.
//
// Needs TEST_DATABASE_URL and TEST_IDENTITY_DATABASE_URL (both disposable;
// the identity one has its public schema dropped).
func TestSignInToAdmin(t *testing.T) {
	siteURL, idURL := os.Getenv("TEST_DATABASE_URL"), os.Getenv("TEST_IDENTITY_DATABASE_URL")
	if siteURL == "" || idURL == "" {
		t.Skip("TEST_DATABASE_URL and TEST_IDENTITY_DATABASE_URL are required")
	}
	ctx := context.Background()
	resetIdentity(t, idURL)
	if err := db.MigrateDown(siteURL, 0); err != nil {
		t.Fatal(err)
	}
	if err := db.MigrateUp(siteURL); err != nil {
		t.Fatal(err)
	}
	pool, err := db.Open(ctx, siteURL)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	cstore := contrib.NewStore(pool)

	// "localhost" so the WebAuthn RP ID is a name, not an IP.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	_, port, _ := net.SplitHostPort(ln.Addr().String())
	base := "http://localhost:" + port
	key := func() string { b := make([]byte, 32); _, _ = rand.Read(b); return hex.EncodeToString(b) }
	cookieKey, _ := hex.DecodeString(key())
	cfg := config.Config{
		SiteName: "Dust Almanac", BaseURL: base, TrustedProxies: "127.0.0.1,::1", DevInsecure: true,
		Identity: config.Identity{
			DatabaseURL: idURL, CryptoKey: key(), BootstrapCode: "let-me-in", ClientID: "wooddust",
			ClientSecret: "test-secret", CookieKey: cookieKey, SMTPPort: 587,
		},
	}
	// ServerMain sets Portiko's variables in the process environment.
	for _, k := range []string{"ISSUER", "DATABASE_URL", "PORT", "SERVICE_NAME", "WEBAUTHN_RP_ID", "WEBAUTHN_RP_ORIGINS",
		"CRYPTO_KEY", "BOOTSTRAP_CODE", "DEV_INSECURE", "SMTP_HOST", "SMTP_PORT", "SMTP_USERNAME", "SMTP_PASSWORD",
		"EMAIL_FROM", "TEMPLATE_DIR", "TRUSTED_PROXIES"} {
		t.Setenv(k, os.Getenv(k))
	}
	if code := ServerMain(cfg, []string{"register-site"}); code != 0 {
		t.Fatalf("register-site exited %d", code)
	}

	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	ds, err := data.Load("../../data")
	if err != nil {
		t.Fatal(err)
	}
	pages, err := content.Load("../../content/pages")
	if err != nil {
		t.Fatal(err)
	}
	cat := memstore.New(ds)
	auth := &contrib.Auth{Users: cstore, Origin: base}
	api := &contrib.API{Store: cstore, Auth: auth, Tokens: contrib.NewFormTokens(cookieKey, 0, time.Hour),
		IPs: contrib.NewIPHasher(cookieKey), Limiter: contrib.NewSubmitLimiter(5, 20, 200),
		Species: cat.GetSpecies, ClientIP: middleware.IP, Log: log}
	srv := site.New(site.Options{Catalog: cat, Pages: pages, SiteName: cfg.SiteName, BaseURL: base, Logger: log, Contrib: api})
	mux := srv.Routes()

	// The relying party discovers the provider at startup, in process.
	id, err := New(ctx, cfg, Options{
		Sessions: cstore.Sessions(),
		OnSignIn: func(ctx context.Context, s *session.Session) error {
			_, err := cstore.SignedIn(ctx, s.Subject, s.Name)
			return err
		},
		Logger: log,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer id.Close()
	auth.Resolve = id.Resolve
	id.Mount(mux)
	trusted, _ := middleware.ParseTrustedProxies(cfg.TrustedProxies)
	hs := httptest.NewUnstartedServer(middleware.Chain(mux, middleware.ClientIP(trusted), middleware.SecurityHeaders))
	hs.Listener = ln
	hs.Start()
	defer hs.Close()

	idPool, err := pgx.Connect(ctx, idURL)
	if err != nil {
		t.Fatal(err)
	}
	defer idPool.Close(ctx)

	owner := newBrowser(t, base, idPool)

	// 1. No session: /admin redirects to sign in.
	resp := owner.get("/admin", false)
	if resp.StatusCode != http.StatusFound || resp.Header.Get("Location") != "/auth/login?return=%2Fadmin" {
		t.Fatalf("/admin anonymous: %d %q", resp.StatusCode, resp.Header.Get("Location"))
	}
	// Provider pages carry their own CSP (inline scripts) and noindex.
	resp = owner.get("/signin", false)
	if !strings.Contains(resp.Header.Get("Content-Security-Policy"), "'unsafe-inline'") || resp.Header.Get("X-Robots-Tag") == "" {
		t.Errorf("/signin headers: %v", resp.Header)
	}

	// 2. Registration is closed without the bootstrap code.
	if r := owner.postJSON("/accounts/v1/register/start", map[string]string{"email": "owner@example.com"}); r.StatusCode != http.StatusForbidden {
		t.Fatalf("register without code: %d", r.StatusCode)
	}
	owner.enroll("wrong", http.StatusForbidden)
	owner.enroll("let-me-in", http.StatusSeeOther)

	// 3. Enroll a passkey, then sign in to the site.
	owner.register("owner@example.com")
	resp = owner.get("/auth/login?return=/admin", true)
	if resp.StatusCode != 200 || resp.Request.URL.Path != "/admin" {
		t.Fatalf("sign-in landed on %s with %d: %s", resp.Request.URL, resp.StatusCode, body(resp))
	}
	page := body(resp)
	if !strings.Contains(page, `"route":"admin"`) || !strings.Contains(page, `"is_admin":true`) {
		t.Fatalf("admin page initial data: %.400s", page)
	}
	if r := owner.get("/api/v1/me", false); r.StatusCode != 200 || !strings.Contains(body(r), `"is_admin":true`) {
		t.Errorf("/api/v1/me: %d", r.StatusCode)
	}

	// 4. Session writes need a same-origin Origin header.
	keyReq := map[string]any{"name": "ha", "scopes": []string{"stats:read"}}
	if r := owner.postJSON("/api/v1/admin/api-keys", keyReq); r.StatusCode != http.StatusForbidden {
		t.Errorf("create key without Origin: %d", r.StatusCode)
	}
	r := owner.postJSON("/api/v1/admin/api-keys", keyReq, "Origin", base)
	var created struct{ Key string }
	_ = json.NewDecoder(r.Body).Decode(&created)
	r.Body.Close()
	if r.StatusCode != 201 || !strings.HasPrefix(created.Key, "da_") {
		t.Fatalf("create key: %d", r.StatusCode)
	}
	anon := newBrowser(t, base, idPool)
	if r := anon.get("/api/v1/stats", false, "Authorization", "Bearer "+created.Key); r.StatusCode != 200 {
		t.Errorf("stats with new key: %d", r.StatusCode)
	}

	// 5. Somebody else who enrolls and signs in is not an admin.
	other := newBrowser(t, base, idPool)
	other.enroll("let-me-in", http.StatusSeeOther)
	other.register("other@example.com")
	if r := other.get("/auth/login?return=/admin", true); r.StatusCode != http.StatusForbidden || r.Request.URL.Path != "/admin" {
		t.Errorf("second person on /admin: %d at %s", r.StatusCode, r.Request.URL)
	}
	if r := other.get("/api/v1/submissions", false); r.StatusCode != http.StatusForbidden {
		t.Errorf("second person on the API: %d", r.StatusCode)
	}

	// 6. Sign out ends the site session and the provider's, so signing in
	// again needs the passkey.
	owner.get("/auth/logout", true)
	if r := owner.get("/admin", false); r.StatusCode != http.StatusFound {
		t.Errorf("after logout: %d", r.StatusCode)
	}
	if r := owner.get("/auth/login?return=/admin", true); r.Request.URL.Path != "/signin" {
		t.Errorf("after logout, sign-in went to %s without a passkey", r.Request.URL)
	}
}

func resetIdentity(t *testing.T, url string) {
	t.Helper()
	conn, err := pgx.Connect(context.Background(), url)
	if err != nil {
		t.Skipf("identity test database unavailable: %v", err)
	}
	defer conn.Close(context.Background())
	if _, err := conn.Exec(context.Background(), "DROP SCHEMA public CASCADE; CREATE SCHEMA public;"); err != nil {
		t.Fatal(err)
	}
}

type browser struct {
	t      *testing.T
	base   string
	client *http.Client
	idDB   *pgx.Conn
	rp     virtualwebauthn.RelyingParty
	auth   virtualwebauthn.Authenticator
}

func newBrowser(t *testing.T, base string, idDB *pgx.Conn) *browser {
	jar, _ := cookiejar.New(nil)
	return &browser{t: t, base: base, idDB: idDB,
		client: &http.Client{Jar: jar, Timeout: 10 * time.Second},
		rp:     virtualwebauthn.RelyingParty{Name: "Dust Almanac", ID: "localhost", Origin: base},
		auth:   virtualwebauthn.NewAuthenticatorWithOptions(virtualwebauthn.AuthenticatorOptions{BackupEligible: true, BackupState: true}),
	}
}

func (b *browser) do(req *http.Request, follow bool) *http.Response {
	b.t.Helper()
	c := *b.client
	if !follow {
		c.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	}
	resp, err := c.Do(req)
	if err != nil {
		b.t.Fatalf("%s %s: %v", req.Method, req.URL, err)
	}
	return resp
}

func (b *browser) get(path string, follow bool, hdr ...string) *http.Response {
	req, _ := http.NewRequest("GET", b.base+path, nil)
	for i := 0; i+1 < len(hdr); i += 2 {
		req.Header.Set(hdr[i], hdr[i+1])
	}
	return b.do(req, follow)
}

func (b *browser) postJSON(path string, v any, hdr ...string) *http.Response {
	blob, _ := json.Marshal(v)
	req, _ := http.NewRequest("POST", b.base+path, bytes.NewReader(blob))
	req.Header.Set("Content-Type", "application/json")
	for i := 0; i+1 < len(hdr); i += 2 {
		req.Header.Set(hdr[i], hdr[i+1])
	}
	return b.do(req, false)
}

func (b *browser) enroll(code string, want int) {
	b.t.Helper()
	req, _ := http.NewRequest("POST", b.base+"/auth/enroll", strings.NewReader(url.Values{"code": {code}}.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", b.base)
	if r := b.do(req, false); r.StatusCode != want {
		b.t.Fatalf("enroll %q: %d, want %d", code, r.StatusCode, want)
	}
}

// register runs Portiko's email-code enrollment with the virtual
// authenticator, as the sign-in page's script would.
func (b *browser) register(email string) {
	b.t.Helper()
	if r := b.postJSON("/accounts/v1/register/start", map[string]string{"email": email}); r.StatusCode != http.StatusAccepted {
		b.t.Fatalf("register/start: %d %s", r.StatusCode, body(r))
	}
	var code string
	if err := b.idDB.QueryRow(context.Background(),
		`SELECT code FROM pending_registrations WHERE email = $1 ORDER BY id DESC LIMIT 1`, email).Scan(&code); err != nil {
		b.t.Fatalf("reading the code: %v", err)
	}
	r := b.postJSON("/accounts/v1/register/verify", map[string]string{"email": email, "code": code})
	if r.StatusCode != 200 {
		b.t.Fatalf("register/verify: %d %s", r.StatusCode, body(r))
	}
	var verified struct {
		CreationOptions json.RawMessage `json:"creation_options"`
	}
	_ = json.NewDecoder(r.Body).Decode(&verified)
	r.Body.Close()
	opts, err := virtualwebauthn.ParseAttestationOptions(string(verified.CreationOptions))
	if err != nil {
		b.t.Fatal(err)
	}
	cred := virtualwebauthn.NewCredential(virtualwebauthn.KeyTypeEC2)
	att := virtualwebauthn.CreateAttestationResponse(b.rp, b.auth, cred, *opts)
	req, _ := http.NewRequest("POST", b.base+"/accounts/v1/register/finish", strings.NewReader(att))
	req.Header.Set("Content-Type", "application/json")
	if fin := b.do(req, false); fin.StatusCode != 200 {
		b.t.Fatalf("register/finish: %d %s", fin.StatusCode, body(fin))
	}
	b.auth.AddCredential(cred)
}

func body(r *http.Response) string {
	defer r.Body.Close()
	b, _ := io.ReadAll(r.Body)
	return string(b)
}
