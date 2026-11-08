package contrib

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/brennanMKE/WoodDust/internal/db"
	"github.com/brennanMKE/WoodDust/internal/model"
)

// These tests need TEST_DATABASE_URL (a disposable database; it is migrated
// down and up).
type apiEnv struct {
	t     *testing.T
	st    *Store
	api   *API
	h     http.Handler
	clock *time.Time
}

func newAPIEnv(t *testing.T) *apiEnv {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	if err := db.MigrateDown(url, 0); err != nil {
		t.Fatalf("migrate down: %v", err)
	}
	if err := db.MigrateUp(url); err != nil {
		t.Fatalf("migrate up: %v", err)
	}
	pool, err := db.Open(context.Background(), url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	st := NewStore(pool)
	now := time.Now()
	clock := &now
	tokens := NewFormTokens(testKey, 4*time.Second, 24*time.Hour)
	tokens.now = func() time.Time { return *clock }
	oak := model.Species{Slug: "oak", BotanicalName: "Quercus spp.", CommonNames: []string{"oak"},
		Hazards: []model.Hazard{{Type: "skin_sensitizer", EvidenceLevel: 3, Sources: []string{"X"}}}}
	api := &API{
		Store: st, Auth: &Auth{Users: st, Origin: "https://example.test"}, Tokens: tokens,
		IPs: NewIPHasher(testKey), Limiter: NewSubmitLimiter(5, 20, 200),
		Species: func(_ context.Context, slug string) (model.Species, bool, error) {
			return oak, slug == "oak", nil
		},
		ClientIP: func(r *http.Request) string { return strings.Split(r.RemoteAddr, ":")[0] },
		Log:      slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	mux := http.NewServeMux()
	api.Register(mux)
	return &apiEnv{t: t, st: st, api: api, h: mux, clock: clock}
}

func (e *apiEnv) do(method, path, ip string, body any, hdr map[string]string) (*httptest.ResponseRecorder, map[string]any) {
	e.t.Helper()
	var rd io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rd = bytes.NewReader(b)
	}
	r := httptest.NewRequest(method, path, rd)
	r.RemoteAddr = ip + ":1234"
	for k, v := range hdr {
		r.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	e.h.ServeHTTP(rec, r)
	var out map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	return rec, out
}

// aged returns a form token issued 10 seconds ago.
func (e *apiEnv) aged() string {
	real := *e.clock
	*e.clock = real.Add(-10 * time.Second)
	tok := e.api.Tokens.Issue()
	*e.clock = real
	return tok
}

func TestSubmissionFlow(t *testing.T) {
	e := newAPIEnv(t)
	ctx := context.Background()
	valid := map[string]any{
		"kind": "correction", "target_slug": "oak", "hazard_type": "skin_sensitizer",
		"details":   "A 2020 case report describes allergic contact dermatitis from oak dust.",
		"citations": []map[string]string{{"doi": "10.1111/cod.12345"}, {"url": "https://www.wood-database.com/oak/"}},
		"email":     "me@example.com", "form_token": e.aged(), "website": "",
	}
	rec, out := e.do("POST", "/api/v1/submissions", "198.51.100.1", valid, nil)
	if rec.Code != 201 || out["reference"] == nil || !strings.HasPrefix(out["status_url"].(string), "/submit/status/") {
		t.Fatalf("create: %d %s", rec.Code, rec.Body)
	}
	if w := out["warnings"].([]any); len(w) != 1 {
		t.Errorf("warnings: %v", w)
	}
	id := int64(out["id"].(float64))
	token := strings.TrimPrefix(out["status_url"].(string), "/submit/status/")

	// Stored privately: hashed IP and token, never the raw values.
	var ipHash string
	var tokHash []byte
	_ = e.st.pool.QueryRow(ctx, `SELECT ip_hash, status_token_hash FROM submissions WHERE id = $1`, id).Scan(&ipHash, &tokHash)
	if strings.Contains(ipHash, "198.51") || len(ipHash) != 32 || bytes.Contains(tokHash, []byte(token)) {
		t.Errorf("ip_hash %q", ipHash)
	}

	rec, st := e.do("GET", "/api/v1/submissions/status/"+token, "1.1.1.1", nil, nil)
	if rec.Code != 200 || st["status"] != "new" || st["reference"] != out["reference"] {
		t.Fatalf("status: %d %s", rec.Code, rec.Body)
	}
	if rec, _ := e.do("GET", "/api/v1/submissions/status/"+strings.Repeat("A", 43), "1.1.1.1", nil, nil); rec.Code != 404 {
		t.Errorf("unknown token: %d", rec.Code)
	}

	// Invalid: 422 with fields.
	bad := map[string]any{"kind": "new_species", "details": "short", "form_token": e.aged()}
	if rec, out := e.do("POST", "/api/v1/submissions", "198.51.100.2", bad, nil); rec.Code != 422 ||
		out["error"].(map[string]any)["fields"] == nil {
		t.Errorf("invalid: %d %s", rec.Code, rec.Body)
	}

	// Honeypot and too-fast: a normal-looking 201, stored as spam.
	hp := map[string]any{"kind": "new_species", "species_name": "x", "details": "buy cheap stuff at my site please",
		"form_token": e.aged(), "website": "http://spam.example", "email": "bot@example.com"}
	rec, sp1 := e.do("POST", "/api/v1/submissions", "198.51.100.3", hp, nil)
	fast := map[string]any{"kind": "new_species", "species_name": "x", "details": "submitted instantly by a script",
		"form_token": e.api.Tokens.Issue()}
	rec2, sp2 := e.do("POST", "/api/v1/submissions", "198.51.100.3", fast, nil)
	if rec.Code != 201 || rec2.Code != 201 || sp1["reference"] == nil || sp2["status_url"] == nil {
		t.Fatalf("spam: %d %d", rec.Code, rec2.Code)
	}
	for _, sp := range []map[string]any{sp1, sp2} {
		var status string
		var email *string
		_ = e.st.pool.QueryRow(ctx, `SELECT status, email FROM submissions WHERE id = $1`, int64(sp["id"].(float64))).Scan(&status, &email)
		if status != "spam" || email != nil {
			t.Errorf("spam stored as %q email %v", status, email)
		}
		tok := strings.TrimPrefix(sp["status_url"].(string), "/submit/status/")
		if _, v := e.do("GET", "/api/v1/submissions/status/"+tok, "1.1.1.1", nil, nil); v["status"] != "new" {
			t.Errorf("spam must show as new: %v", v)
		}
	}

	// Expired token: 422, not spam.
	*e.clock = e.clock.Add(-25 * time.Hour)
	old := e.api.Tokens.Issue()
	*e.clock = e.clock.Add(25 * time.Hour)
	valid["form_token"] = old
	if rec, _ := e.do("POST", "/api/v1/submissions", "198.51.100.4", valid, nil); rec.Code != 422 {
		t.Errorf("expired token: %d", rec.Code)
	}

	// Per-IP limit: 5 per hour, then 429 with Retry-After.
	valid["form_token"] = e.aged()
	for i := range 5 {
		if rec, _ := e.do("POST", "/api/v1/submissions", "203.0.113.50", valid, nil); rec.Code != 201 {
			t.Fatalf("post %d: %d", i, rec.Code)
		}
	}
	rec, _ = e.do("POST", "/api/v1/submissions", "203.0.113.50", valid, nil)
	if rec.Code != 429 || rec.Header().Get("Retry-After") == "" {
		t.Errorf("limit: %d %v", rec.Code, rec.Header())
	}

	// Admin without credentials: 401.
	if rec, _ := e.do("GET", "/api/v1/submissions", "1.1.1.1", nil, nil); rec.Code != 401 {
		t.Errorf("list without key: %d", rec.Code)
	}

	// Keys: read-only, then read-write.
	_, readKey, err := e.st.CreateAPIKey(ctx, "ha", []string{ScopeSubmissionsRead, ScopeStatsRead}, nil, nil, "test")
	if err != nil {
		t.Fatal(err)
	}
	_, writeKey, _ := e.st.CreateAPIKey(ctx, "rw", []string{ScopeSubmissionsRead, ScopeSubmissionsWrite}, nil, nil, "test")
	auth := func(k string) map[string]string { return map[string]string{"Authorization": "Bearer " + k} }

	rec, list := e.do("GET", "/api/v1/submissions?limit=2", "1.1.1.1", nil, auth(readKey))
	if rec.Code != 200 || len(list["submissions"].([]any)) != 2 || list["next_cursor"] == nil {
		t.Fatalf("list: %d %s", rec.Code, rec.Body)
	}
	rec, list = e.do("GET", "/api/v1/submissions?status=spam", "1.1.1.1", nil, auth(readKey))
	if len(list["submissions"].([]any)) != 2 {
		t.Errorf("spam filter: %s", rec.Body)
	}

	rec, _ = e.do("PATCH", "/api/v1/submissions/"+itoa(id), "1.1.1.1", map[string]any{"status": "accepted"}, auth(readKey))
	if rec.Code != 403 {
		t.Errorf("patch with read key: %d", rec.Code)
	}
	patch := map[string]any{"status": "accepted", "note": map[string]any{"text": "Thanks, we will add it.", "visible_to_submitter": true}}
	rec, got := e.do("PATCH", "/api/v1/submissions/"+itoa(id), "1.1.1.1", patch, auth(writeKey))
	if rec.Code != 200 || got["status"] != "accepted" || got["email"] == nil {
		t.Fatalf("patch: %d %s", rec.Code, rec.Body)
	}
	rec, det := e.do("GET", "/api/v1/submissions/"+itoa(id), "1.1.1.1", nil, auth(readKey))
	if rec.Code != 200 || det["kind"] != "correction" || len(det["events"].([]any)) != 3 {
		t.Fatalf("detail: %d %s", rec.Code, rec.Body)
	}
	if ev := det["events"].([]any)[0].(map[string]any); ev["type"] != "created" {
		t.Errorf("event shape: %v", ev)
	}
	_, st = e.do("GET", "/api/v1/submissions/status/"+token, "1.1.1.1", nil, nil)
	if st["status"] != "accepted" || len(st["notes"].([]any)) != 1 {
		t.Errorf("status after patch: %v", st)
	}

	rec, ex := e.do("GET", "/api/v1/submissions/"+itoa(id)+"/export", "1.1.1.1", nil, auth(readKey))
	if rec.Code != 200 || ex["draft"] != true || ex["action"] != "patch" {
		t.Errorf("export: %d %s", rec.Code, rec.Body)
	}

	// Closing deletes the email.
	_, got = e.do("PATCH", "/api/v1/submissions/"+itoa(id), "1.1.1.1", map[string]any{"status": "merged"}, auth(writeKey))
	if got["email"] != nil {
		t.Errorf("email kept after merge: %v", got["email"])
	}
	var audits int
	_ = e.st.pool.QueryRow(ctx, `SELECT count(*) FROM audit_log WHERE action = 'submission.update' AND actor LIKE 'key:%'`).Scan(&audits)
	if audits != 2 {
		t.Errorf("audit rows: %d", audits)
	}

	// Stats.
	if rec, _ := e.do("GET", "/api/v1/stats", "1.1.1.1", nil, auth(writeKey)); rec.Code != 403 {
		t.Errorf("stats without scope: %d", rec.Code)
	}
	rec, stats := e.do("GET", "/api/v1/stats", "1.1.1.1", nil, auth(readKey))
	if rec.Code != 200 || stats["open"].(float64) != 5 || stats["spam_last_24h"].(float64) != 2 ||
		stats["new_last_24h"].(float64) != 6 || stats["newest_at"] == nil {
		t.Errorf("stats: %d %s", rec.Code, rec.Body)
	}
	var lastUsed *time.Time
	_ = e.st.pool.QueryRow(ctx, `SELECT last_used_at FROM api_keys WHERE name = 'ha'`).Scan(&lastUsed)
	if lastUsed == nil {
		t.Error("last_used_at not set")
	}

	// Revoked keys stop working.
	p, _ := ParseAPIKey(readKey)
	if err := e.st.RevokeAPIKey(ctx, p, "test"); err != nil {
		t.Fatal(err)
	}
	if rec, _ := e.do("GET", "/api/v1/stats", "1.1.1.1", nil, auth(readKey)); rec.Code != 401 {
		t.Errorf("revoked key: %d", rec.Code)
	}

	// check-url and form-token are public.
	if _, cu := e.do("GET", "/api/v1/submissions/check-url?url=https://haz-map.com/x", "1.1.1.1", nil, nil); cu["leads_only"] != true {
		t.Errorf("check-url: %v", cu)
	}
	if _, ft := e.do("GET", "/api/v1/submissions/form-token", "1.1.1.1", nil, nil); ft["form_token"] == "" {
		t.Error("form-token")
	}
}

func TestFirstSignInBecomesAdmin(t *testing.T) {
	e := newAPIEnv(t)
	ctx := context.Background()
	a, err := e.st.SignedIn(ctx, "usr_owner", "Owner")
	if err != nil || !a.IsAdmin {
		t.Fatalf("first: %+v %v", a, err)
	}
	b, _ := e.st.SignedIn(ctx, "usr_other", "")
	if b.IsAdmin {
		t.Error("second user became admin")
	}
	a, _ = e.st.SignedIn(ctx, "usr_owner", "")
	if !a.IsAdmin || a.DisplayName != "Owner" {
		t.Errorf("re-sign-in: %+v", a)
	}
}

func itoa(n int64) string { return strconv.FormatInt(n, 10) }
