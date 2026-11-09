package posters

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"testing/fstest"
	"time"

	"github.com/brennanMKE/WoodDust/internal/contrib"
	"github.com/brennanMKE/WoodDust/internal/settings"
)

const token = "slk_AbCdEfGh012345678901234567890123456789-_xyz"

var key32 = bytes.Repeat([]byte{3}, 32)

// fakeSettings is an in-memory SettingsStore.
type fakeSettings struct {
	mu     sync.Mutex
	values map[string]string
	checks []bool
	msg    string
}

func (f *fakeSettings) Value(_ context.Context, k string) (string, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if v, ok := f.values[k]; ok {
		return v, true, nil
	}
	d, _ := settings.Lookup(k)
	if d.Secret {
		return "", false, nil
	}
	return d.Default, true, nil
}

func (f *fakeSettings) Info(_ context.Context, k string) (settings.Info, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	d, _ := settings.Lookup(k)
	in := settings.Info{Key: k, Secret: d.Secret}
	if v, ok := f.values[k]; ok {
		in.Set, in.Hint = true, d.Hint(v)
		if k == settings.ShortLinksAPIKey && len(f.checks) > 0 {
			now := time.Now()
			ok := f.checks[len(f.checks)-1]
			in.CheckedAt, in.CheckOK, in.CheckError = &now, &ok, f.msg
		}
	} else if !d.Secret {
		in.Hint, in.IsDefault = d.Default, true
	}
	return in, nil
}

func (f *fakeSettings) Set(_ context.Context, k, v, _ string) (settings.Info, error) {
	d, _ := settings.Lookup(k)
	v, err := d.Check(v)
	if err != nil {
		return settings.Info{}, err
	}
	f.mu.Lock()
	f.values[k] = v
	f.checks = nil
	f.mu.Unlock()
	return f.Info(context.Background(), k)
}

func (f *fakeSettings) Clear(_ context.Context, k, _ string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.values, k)
	return nil
}

func (f *fakeSettings) RecordCheck(_ context.Context, _ string, ok bool, msg, _ string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.checks = append(f.checks, ok)
	f.msg = msg
	return nil
}

func (f *fakeSettings) CanSeal() bool { return true }

// fakeShortLinks answers like ShortLinks v0.4.0 with API tokens.
type fakeShortLinks struct {
	mu       sync.Mutex
	created  []LinkRequest
	fail     int // status to answer /api/links with
	readOnly bool
	baseURL  string
}

func (f *fakeShortLinks) handler(t *testing.T) http.Handler {
	mux := http.NewServeMux()
	auth := func(w http.ResponseWriter, r *http.Request) bool {
		if r.Header.Get("Authorization") != "Bearer "+token {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = io.WriteString(w, `{"error":"unauthenticated"}`)
			return false
		}
		return true
	}
	mux.HandleFunc("GET /api/me", func(w http.ResponseWriter, r *http.Request) {
		if !auth(w, r) {
			return
		}
		fmt.Fprintf(w, `{"id":1,"email":"b@example.com","is_admin":false,"base_url":%q,"api_token":{"id":3,"name":"wooddust-test","read_only":%v}}`, f.baseURL, f.readOnly)
	})
	mux.HandleFunc("GET /api/campaigns/{slug}", func(w http.ResponseWriter, r *http.Request) {
		if !auth(w, r) {
			return
		}
		if r.PathValue("slug") != "posters" {
			w.WriteHeader(http.StatusNotFound)
			_, _ = io.WriteString(w, `{"error":"not found"}`)
			return
		}
		_, _ = io.WriteString(w, `{"slug":"posters"}`)
	})
	mux.HandleFunc("POST /api/links", func(w http.ResponseWriter, r *http.Request) {
		if !auth(w, r) {
			return
		}
		if f.fail != 0 {
			w.Header().Set("Retry-After", "7")
			w.WriteHeader(f.fail)
			_, _ = io.WriteString(w, `{"error":"rate_limit_exceeded"}`)
			return
		}
		var lr LinkRequest
		if err := json.NewDecoder(r.Body).Decode(&lr); err != nil {
			t.Errorf("decode: %v", err)
		}
		f.mu.Lock()
		f.created = append(f.created, lr)
		f.mu.Unlock()
		w.WriteHeader(http.StatusCreated)
		fmt.Fprintf(w, `{"id":9,"key":%q,"destination_url":%q}`, lr.Key, lr.DestinationURL)
	})
	mux.HandleFunc("GET /api/links/{key}/qr.svg", func(w http.ResponseWriter, r *http.Request) {
		if !auth(w, r) {
			return
		}
		w.Header().Set("Content-Type", "image/svg+xml")
		_, _ = io.WriteString(w, `<svg xmlns="http://www.w3.org/2000/svg" data-key="`+r.PathValue("key")+`"></svg>`)
	})
	return mux
}

type env struct {
	t     *testing.T
	svc   *Service
	mux   *http.ServeMux
	sl    *fakeShortLinks
	st    *fakeSettings
	tok   *contrib.FormTokens
	logs  *bytes.Buffer
	clock time.Time
}

func newEnv(t *testing.T) *env {
	sl := &fakeShortLinks{baseURL: "https://go.dustalmanac.com"}
	ts := httptest.NewServer(sl.handler(t))
	t.Cleanup(ts.Close)
	st := &fakeSettings{values: map[string]string{
		settings.ShortLinksAPIKey: token,
		settings.ShortLinksAPIURL: ts.URL,
	}}
	tok := contrib.NewFormTokens(key32, 0, time.Hour)
	logs := &bytes.Buffer{}
	svc := &Service{
		Settings: st, SiteBase: "https://www.dustalmanac.com", Tokens: tok, IPs: contrib.NewIPHasher(key32),
		Limiter:  contrib.NewSubmitLimiter(LinksPerIPHour, LinksPerIPDay, LinksGlobalDay),
		ClientIP: func(r *http.Request) string { return strings.Split(r.RemoteAddr, ":")[0] },
		Auth:     &contrib.Auth{Origin: "https://www.dustalmanac.com"},
		Stock:    fstest.MapFS{"qr/shave.svg": {Data: []byte("<svg>stock-shave</svg>")}},
		Log:      slog.New(slog.NewTextHandler(logs, nil)),
	}
	mux := http.NewServeMux()
	svc.Register(mux)
	return &env{t: t, svc: svc, mux: mux, sl: sl, st: st, tok: tok, logs: logs}
}

func (e *env) post(ip string, body map[string]string) (*httptest.ResponseRecorder, map[string]any) {
	e.t.Helper()
	if _, ok := body["form_token"]; !ok {
		body["form_token"] = e.tok.Issue()
	}
	b, _ := json.Marshal(body)
	r := httptest.NewRequest(http.MethodPost, "/api/v1/posters/links", bytes.NewReader(b))
	r.RemoteAddr = ip + ":1"
	w := httptest.NewRecorder()
	e.mux.ServeHTTP(w, r)
	var out map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	return w, out
}

func TestMintCustomLink(t *testing.T) {
	e := newEnv(t)
	w, out := e.post("10.0.0.1", map[string]string{"poster": "shave", "shop": "  Brennan's\tShop\n  of   Wonders, a very long name indeed  "})
	if w.Code != http.StatusOK || out["fallback"] != nil {
		t.Fatalf("got %d %v", w.Code, out)
	}
	if len(e.sl.created) != 1 {
		t.Fatalf("created %d links", len(e.sl.created))
	}
	lr := e.sl.created[0]
	if lr.DestinationURL != "https://www.dustalmanac.com/protection/respirators" || lr.CampaignSlug != "posters" ||
		lr.UTMSource != "flyer" || lr.UTMMedium != "print" || lr.UTMCampaign != "poster-shave" || lr.UTMContent != "custom" {
		t.Errorf("link request %+v", lr)
	}
	if len(lr.Key) != 10 {
		t.Errorf("custom key %q", lr.Key)
	}
	if lr.Placement != "Brennan's Shop of Wonders, a very long n" || len([]rune(lr.Placement)) != 40 {
		t.Errorf("placement %q", lr.Placement)
	}
	if out["short_url"] != "https://go.dustalmanac.com/u/"+lr.Key || !strings.Contains(out["qr_svg"].(string), lr.Key) {
		t.Errorf("answer %v", out)
	}
	if strings.Contains(e.logs.String(), token) || strings.Contains(e.logs.String(), "Brennan") {
		t.Error("token or shop name logged")
	}
}

func TestShopOnlyWhenSent(t *testing.T) {
	e := newEnv(t)
	e.post("10.0.0.1", map[string]string{"poster": "sunbeam"})
	if len(e.sl.created) != 1 || e.sl.created[0].Placement != "" || e.sl.created[0].UTMCampaign != "poster-sunbeam" {
		t.Fatalf("%+v", e.sl.created)
	}
}

func TestFallbacks(t *testing.T) {
	t.Run("no key", func(t *testing.T) {
		e := newEnv(t)
		delete(e.st.values, settings.ShortLinksAPIKey)
		w, out := e.post("10.0.0.1", map[string]string{"poster": "shave"})
		if w.Code != 200 || out["fallback"] != true || out["short_url"] != "https://go.dustalmanac.com/u/shave" || out["qr_svg"] != "<svg>stock-shave</svg>" {
			t.Fatalf("%d %v", w.Code, out)
		}
	})
	t.Run("no database", func(t *testing.T) {
		e := newEnv(t)
		e.svc.Settings = nil
		w, out := e.post("10.0.0.1", map[string]string{"poster": "sunbeam"})
		if w.Code != 200 || out["fallback"] != true || !strings.Contains(out["qr_svg"].(string), "<svg") {
			t.Fatalf("%d %v", w.Code, out)
		}
	})
	t.Run("shortlinks fails", func(t *testing.T) {
		e := newEnv(t)
		e.sl.fail = http.StatusTooManyRequests
		w, out := e.post("10.0.0.1", map[string]string{"poster": "shave"})
		if w.Code != 200 || out["fallback"] != true {
			t.Fatalf("%d %v", w.Code, out)
		}
		if len(e.st.checks) != 1 || e.st.checks[0] || !strings.Contains(e.st.msg, "7 s") {
			t.Errorf("failure not recorded: %v %q", e.st.checks, e.st.msg)
		}
	})
	t.Run("bad token", func(t *testing.T) {
		e := newEnv(t)
		e.st.values[settings.ShortLinksAPIKey] = strings.Replace(token, "A", "B", 1)
		_, out := e.post("10.0.0.1", map[string]string{"poster": "shave"})
		if out["fallback"] != true || !strings.Contains(e.st.msg, "refused") {
			t.Fatalf("%v %q", out, e.st.msg)
		}
	})
	t.Run("unreachable", func(t *testing.T) {
		e := newEnv(t)
		e.st.values[settings.ShortLinksAPIURL] = "http://127.0.0.1:1"
		w, out := e.post("10.0.0.1", map[string]string{"poster": "shave"})
		if w.Code != 200 || out["fallback"] != true {
			t.Fatalf("%d %v", w.Code, out)
		}
	})
}

func TestAbuseControls(t *testing.T) {
	e := newEnv(t)
	if w, _ := e.post("10.0.0.1", map[string]string{"poster": "tall-tale"}); w.Code != http.StatusUnprocessableEntity {
		t.Errorf("unpublished poster: %d", w.Code)
	}
	if w, _ := e.post("10.0.0.1", map[string]string{"poster": "shave", "form_token": "forged.x"}); w.Code != http.StatusUnprocessableEntity {
		t.Errorf("forged token: %d", w.Code)
	}
	// Honeypot: a normal-looking answer, nothing minted.
	w, out := e.post("10.0.0.2", map[string]string{"poster": "shave", "website": "x"})
	if w.Code != 200 || out["fallback"] != nil || out["short_url"] != "https://go.dustalmanac.com/u/shave" || len(e.sl.created) != 0 {
		t.Errorf("honeypot: %d %v created=%d", w.Code, out, len(e.sl.created))
	}
	// 3 per hour per IP (the honeypot counted one for 10.0.0.2).
	for i := 0; i < 2; i++ {
		if w, _ := e.post("10.0.0.2", map[string]string{"poster": "shave"}); w.Code != 200 {
			t.Fatalf("request %d: %d", i, w.Code)
		}
	}
	w, _ = e.post("10.0.0.2", map[string]string{"poster": "shave"})
	if w.Code != http.StatusTooManyRequests || w.Header().Get("Retry-After") == "" {
		t.Errorf("4th in an hour: %d", w.Code)
	}
	// Body cap.
	r := httptest.NewRequest(http.MethodPost, "/api/v1/posters/links", strings.NewReader(`{"poster":"shave","shop":"`+strings.Repeat("x", 8<<10)+`"}`))
	r.RemoteAddr = "10.0.0.3:1"
	rec := httptest.NewRecorder()
	e.mux.ServeHTTP(rec, r)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("oversized body: %d", rec.Code)
	}
}

func TestTooFastIsSpam(t *testing.T) {
	e := newEnv(t)
	e.svc.Tokens = contrib.NewFormTokens(key32, time.Hour, 2*time.Hour)
	w, out := e.post("10.0.0.1", map[string]string{"poster": "shave", "form_token": e.svc.Tokens.Issue()})
	if w.Code != 200 || out["fallback"] != nil || len(e.sl.created) != 0 {
		t.Fatalf("%d %v", w.Code, out)
	}
}

func TestCleanShop(t *testing.T) {
	for in, want := range map[string]string{"": "", "  ": "", "A\u2028B": "A B", "Ünïcødé  shop": "Ünïcødé shop"} {
		if got := CleanShop(in); got != want {
			t.Errorf("CleanShop(%q) = %q", in, got)
		}
	}
}

func TestTokenRedacted(t *testing.T) {
	var buf bytes.Buffer
	l := slog.New(slog.NewJSONHandler(&buf, nil))
	c := Client{BaseURL: "http://x", Token: Token(token)}
	l.Info("call", "token", c.Token, "client", c)
	s := fmt.Sprintf("%v %+v %#v %s", c, c, c, c.Token)
	if strings.Contains(buf.String(), "slk_") || strings.Contains(s, "slk_") {
		t.Fatalf("token leaked: %s %s", buf.String(), s)
	}
}

func TestQRSVG(t *testing.T) {
	s, err := QRSVG("https://go.dustalmanac.com/u/shave")
	if err != nil || !strings.HasPrefix(s, "<svg") || strings.Contains(s, "style") {
		t.Fatalf("%v %q", err, s[:40])
	}
}

func TestTemplatesAreCSPSafe(t *testing.T) {
	for _, p := range Published() {
		b, err := os.ReadFile("../../posters/src/" + p.ID + ".svg")
		if err != nil {
			t.Fatalf("%s: %v", p.ID, err)
		}
		s := reComment.ReplaceAllString(string(b), "") // the header comment names the rules
		for _, bad := range []string{"style=", "<style", "<script", "on" + "load="} {
			if strings.Contains(s, bad) {
				t.Errorf("%s.svg contains %q (strict CSP: presentation attributes only)", p.ID, bad)
			}
		}
		for _, slot := range []string{`data-slot="art" href="art.png"`, `data-slot="qr" href="qr.svg"`, `data-slot="plaque"`, `data-slot="plaque-name"`, `data-slot="slogan"`, `data-bg="#`} {
			if !strings.Contains(s, slot) {
				t.Errorf("%s.svg lacks %s", p.ID, slot)
			}
		}
		c := Compose(s, nil, "<svg/>")
		if strings.Contains(c, `href="art.png"`) || strings.Contains(c, `href="qr.svg"`) {
			t.Errorf("%s: slots not filled", p.ID)
		}
		page := Page(Compose(s, []byte{1}, "<svg/>"), Background(s), Sheets[0], true)
		if !strings.Contains(page, `width="215.9mm"`) || strings.Contains(page, "#2b211b") {
			t.Errorf("%s: page wrapper/grey wrong", p.ID)
		}
	}
}

func TestCheck(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	check := func() string {
		msg, configured, err := e.svc.Check(ctx)
		if err != nil || !configured {
			t.Fatalf("configured=%v err=%v", configured, err)
		}
		return msg
	}
	if got := check(); got != "" {
		t.Fatalf("healthy: %q", got)
	}
	e.sl.readOnly = true
	if !strings.Contains(check(), "read-only") {
		t.Error("read-only token not caught")
	}
	e.sl.readOnly, e.sl.baseURL = false, "https://go.sstools.co"
	if !strings.Contains(check(), "go.sstools.co") {
		t.Error("base_url mismatch not caught")
	}
	e.sl.baseURL = "https://go.dustalmanac.com"
	e.st.values[settings.ShortLinksCampaign] = "flyers"
	if !strings.Contains(check(), "no campaign") {
		t.Error("missing campaign not caught")
	}
	e.st.values[settings.ShortLinksAPIKey] = strings.Replace(token, "A", "B", 1)
	if !strings.Contains(check(), "refused") {
		t.Error("bad token not caught")
	}
	delete(e.st.values, settings.ShortLinksAPIKey)
	if _, configured, _ := e.svc.Check(ctx); configured {
		t.Error("no key should be not configured")
	}
}

func TestAdminNeedsSession(t *testing.T) {
	e := newEnv(t)
	for _, m := range []string{"GET /api/v1/admin/settings", "POST /api/v1/admin/settings/shortlinks/test", "PUT /api/v1/admin/settings/shortlinks_api_key", "DELETE /api/v1/admin/settings/shortlinks_api_key"} {
		method, path, _ := strings.Cut(m, " ")
		r := httptest.NewRequest(method, path, strings.NewReader(`{"value":"x"}`))
		w := httptest.NewRecorder()
		e.mux.ServeHTTP(w, r)
		if w.Code < 400 || w.Code == http.StatusOK {
			t.Errorf("%s: %d without a session", m, w.Code)
		}
		if strings.Contains(w.Body.String(), "slk_") {
			t.Errorf("%s leaked the key", m)
		}
	}
}
