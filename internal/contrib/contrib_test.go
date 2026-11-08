package contrib

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Kolonnade/Portiko/rp/session"

	"github.com/brennanMKE/WoodDust/internal/model"
)

var testKey = []byte("0123456789abcdef0123456789abcdef")

func TestFormTokens(t *testing.T) {
	ft := NewFormTokens(testKey, 4*time.Second, 24*time.Hour)
	now := time.Unix(1_800_000_000, 0)
	ft.now = func() time.Time { return now }
	tok := ft.Issue()
	if err := ft.Check(tok); !errors.Is(err, ErrTokenTooFast) {
		t.Errorf("immediate: %v", err)
	}
	now = now.Add(5 * time.Second)
	if err := ft.Check(tok); err != nil {
		t.Errorf("after 5s: %v", err)
	}
	now = now.Add(25 * time.Hour)
	if err := ft.Check(tok); !errors.Is(err, ErrTokenExpired) {
		t.Errorf("after 25h: %v", err)
	}
	for _, bad := range []string{"", "x", tok + "x", strings.Replace(tok, ".", "A.", 1)} {
		if err := ft.Check(bad); !errors.Is(err, ErrTokenInvalid) {
			t.Errorf("%q: %v", bad, err)
		}
	}
	other := NewFormTokens([]byte("another key another key another k"), 0, time.Hour)
	other.now = ft.now
	if err := other.Check(tok); !errors.Is(err, ErrTokenInvalid) {
		t.Error("token from another key accepted")
	}
}

func TestIPHasherRotatesDaily(t *testing.T) {
	h := NewIPHasher(testKey)
	day := time.Date(2026, 11, 8, 10, 0, 0, 0, time.UTC)
	h.now = func() time.Time { return day }
	a, b := h.Hash("203.0.113.9"), h.Hash("203.0.113.9")
	if a != b || len(a) != 32 || strings.Contains(a, "203") {
		t.Fatalf("hash %q %q", a, b)
	}
	if h.Hash("203.0.113.10") == a {
		t.Error("different IPs collide")
	}
	day = day.Add(24 * time.Hour)
	if h.Hash("203.0.113.9") == a {
		t.Error("salt did not rotate")
	}
}

func TestAPIKeys(t *testing.T) {
	key, prefix, hash := NewAPIKey()
	if !strings.HasPrefix(key, "da_"+prefix+"_") || len(prefix) != 8 {
		t.Fatalf("key %q prefix %q", key, prefix)
	}
	if p, ok := ParseAPIKey(key); !ok || p != prefix {
		t.Errorf("parse: %q %v", p, ok)
	}
	if !KeyMatches(key, hash) || KeyMatches(key+"x", hash) {
		t.Error("KeyMatches")
	}
	for _, bad := range []string{"", "da_", "da_short_x", "xx_" + key[3:], "da_" + prefix + "_bad!chars-here-123456789"} {
		if _, ok := ParseAPIKey(bad); ok {
			t.Errorf("%q parsed", bad)
		}
	}
}

func TestValidate(t *testing.T) {
	exists := func(s string) bool { return s == "oak" }
	good := Request{Kind: "correction", TargetSlug: "oak", HazardType: "skin_sensitizer",
		Details:   "Oak sawdust caused contact dermatitis in a published case report.",
		Citations: []Citation{{DOI: "https://doi.org/10.1016/j.x.2020.01.001"}, {}, {URL: "https://www.wood-database.com/oak"}}}
	fields, warnings := Validate(&good, exists)
	if len(fields) != 0 {
		t.Fatalf("fields: %v", fields)
	}
	if len(good.Citations) != 2 || good.Citations[0].DOI != "10.1016/j.x.2020.01.001" {
		t.Errorf("citations not normalized: %+v", good.Citations)
	}
	if len(warnings) != 1 || !strings.Contains(warnings[0], "wood-database.com") {
		t.Errorf("warnings: %v", warnings)
	}

	cases := []struct {
		req   Request
		field string
	}{
		{Request{Kind: "bogus", Details: strings.Repeat("x", 30)}, "kind"},
		{Request{Kind: "new_species", Details: "short"}, "details"},
		{Request{Kind: "new_species", SpeciesName: "Bubinga", Details: strings.Repeat("x", 30)}, "citations"},
		{Request{Kind: "correction", Details: strings.Repeat("x", 30), Citations: []Citation{{PMID: "123"}}}, "target_slug"},
		{Request{Kind: "correction", TargetSlug: "nope", Details: strings.Repeat("x", 30), Citations: []Citation{{PMID: "1"}}}, "target_slug"},
		{Request{Kind: "new_source", TargetSlug: "oak", Details: strings.Repeat("x", 30), Citations: []Citation{{DOI: "11.1/x"}}}, "citations"},
		{Request{Kind: "new_source", TargetSlug: "oak", Details: strings.Repeat("x", 30), Citations: []Citation{{PMID: "12a"}}}, "citations"},
		{Request{Kind: "new_source", TargetSlug: "oak", Details: strings.Repeat("x", 30), Citations: []Citation{{URL: "ftp://x.org/a"}}}, "citations"},
		{Request{Kind: "new_source", TargetSlug: "oak", Details: strings.Repeat("x", 30), Citations: []Citation{{Text: "a book"}}}, "citations"},
		{Request{Kind: "reaction_report", Details: strings.Repeat("x", 30)}, "species_name"},
		{Request{Kind: "reaction_report", TargetSlug: "oak", Details: strings.Repeat("x", 30), Email: "nope"}, "email"},
		{Request{Kind: "reaction_report", TargetSlug: "oak", Details: strings.Repeat("x", 30), HazardType: "zombie"}, "hazard_type"},
		{Request{Kind: "reaction_report", TargetSlug: "oak", Details: strings.Repeat("see https://a.b ", 6)}, "details"},
	}
	for i, c := range cases {
		fields, _ := Validate(&c.req, exists)
		if _, ok := fields[c.field]; !ok {
			t.Errorf("case %d: want error on %s, got %v", i, c.field, fields)
		}
	}
	ok := Request{Kind: "reaction_report", TargetSlug: "oak", Details: "My hands itched after sanding oak all day."}
	if fields, _ := Validate(&ok, exists); len(fields) != 0 {
		t.Errorf("reaction report needs no citation: %v", fields)
	}
}

func TestCheckURL(t *testing.T) {
	if ok, w := CheckURL("https://www.cdc.gov/niosh/x"); !ok || len(w) != 0 {
		t.Errorf("cdc: %v %v", ok, w)
	}
	if ok, w := CheckURL("https://sub.haz-map.com/x"); !ok || len(w) != 1 {
		t.Errorf("haz-map: %v %v", ok, w)
	}
	if ok, _ := CheckURL("javascript:alert(1)"); ok {
		t.Error("javascript URL accepted")
	}
	if _, w := CheckURL("https://notwood-database.com/"); len(w) != 0 {
		t.Error("suffix match must be on a label boundary")
	}
}

func TestSubmitLimiter(t *testing.T) {
	l := NewSubmitLimiter(2, 3, 5)
	now := time.Unix(1_800_000_000, 0)
	l.now = func() time.Time { return now }
	take := func(k string) bool {
		ok, _ := l.Check(k)
		if ok {
			l.Record(k)
		}
		return ok
	}
	if !take("a") || !take("a") {
		t.Fatal("first two per hour")
	}
	if ok, wait := l.Check("a"); ok || wait <= 0 || wait > time.Hour {
		t.Fatalf("third in the hour: %v %v", ok, wait)
	}
	now = now.Add(61 * time.Minute)
	if !take("a") {
		t.Fatal("after an hour")
	}
	if take("a") {
		t.Fatal("per-day ceiling of 3")
	}
	if !take("b") || !take("b") {
		t.Fatal("other clients are independent")
	}
	if take("c") {
		t.Fatal("global ceiling of 5")
	}
	now = now.Add(25 * time.Hour)
	if !take("c") {
		t.Fatal("windows slide")
	}
}

func TestBuildExport(t *testing.T) {
	str := func(s string) *string { return &s }
	s := Submission{ID: 7, Reference: "DA-0007", Kind: "new_species", SpeciesName: str("Bubinga"),
		BotanicalName: str("Guibourtia spp."), HazardType: str("skin_sensitizer"),
		Citations: []Citation{{DOI: "10.1/abc", Text: "Someone 2020"}}}
	ex, err := BuildExport(s, nil)
	if err != nil || !ex.Draft || ex.Action != "create" || ex.File != "data/species/bubinga.json" {
		t.Fatalf("%+v %v", ex, err)
	}
	if _, ok := ex.ProposedSources["TODO-DA-0007-1"]; !ok {
		t.Errorf("sources: %v", ex.ProposedSources)
	}
	sp := &model.Species{Slug: "oak", Hazards: []model.Hazard{{Type: "skin_sensitizer", EvidenceLevel: 3, Sources: []string{"A"}}}}
	s.Kind, s.TargetSlug = "new_source", str("oak")
	ex, err = BuildExport(s, sp)
	if err != nil || ex.Action != "patch" || len(ex.Changes) == 0 {
		t.Fatalf("%+v %v", ex, err)
	}
	if len(sp.Hazards[0].Sources) != 1 {
		t.Error("export mutated the published record")
	}
	s.Kind = "reaction_report"
	if _, err := BuildExport(s, sp); err == nil {
		t.Error("reaction reports must not export")
	}
}

// ---- admin middleware with a fake session -------------------------------------

type fakeUsers struct {
	users map[string]User
	keys  map[string]APIKey
}

func (f fakeUsers) UserBySubject(_ context.Context, subject string) (User, error) {
	u, ok := f.users[subject]
	if !ok {
		return u, ErrNotFound
	}
	return u, nil
}

func (f fakeUsers) AuthenticateKey(_ context.Context, key string) (APIKey, error) {
	k, ok := f.keys[key]
	if !ok {
		return k, ErrNotFound
	}
	return k, nil
}

func fakeSession(r *http.Request) (*session.Session, bool) {
	if c, err := r.Cookie("fake"); err == nil {
		return &session.Session{Subject: c.Value}, true
	}
	return nil, false
}

func TestAuthorize(t *testing.T) {
	users := fakeUsers{
		users: map[string]User{"owner": {Subject: "owner", IsAdmin: true, Active: true}, "guest": {Subject: "guest", Active: true}},
		keys:  map[string]APIKey{"da_stats": {Prefix: "stats", Scopes: []string{ScopeStatsRead}}},
	}
	a := &Auth{Users: users, Resolve: fakeSession, Origin: "https://example.test"}
	req := func(method, who, origin, key string) *http.Request {
		r := httptest.NewRequest(method, "/api/v1/stats", nil)
		if who != "" {
			r.AddCookie(&http.Cookie{Name: "fake", Value: who})
		}
		if origin != "" {
			r.Header.Set("Origin", origin)
		}
		if key != "" {
			r.Header.Set("Authorization", "Bearer "+key)
		}
		return r
	}
	status := func(err error) int {
		var e *errAuth
		if err == nil {
			return 200
		}
		if errors.As(err, &e) {
			return e.status
		}
		return 500
	}
	cases := []struct {
		name              string
		method, who, o, k string
		scope             string
		want              int
	}{
		{"anonymous", "GET", "", "", "", ScopeStatsRead, 401},
		{"admin read", "GET", "owner", "", "", ScopeStatsRead, 200},
		{"non-admin", "GET", "guest", "", "", ScopeStatsRead, 403},
		{"unknown subject", "GET", "stranger", "", "", ScopeStatsRead, 403},
		{"admin write, no Origin", "PATCH", "owner", "", "", ScopeSubmissionsWrite, 403},
		{"admin write, foreign Origin", "PATCH", "owner", "https://evil.test", "", ScopeSubmissionsWrite, 403},
		{"admin write, same Origin", "PATCH", "owner", "https://example.test", "", ScopeSubmissionsWrite, 200},
		{"key with scope", "GET", "", "", "da_stats", ScopeStatsRead, 200},
		{"key without scope", "GET", "", "", "da_stats", ScopeSubmissionsRead, 403},
		{"key write needs no Origin but scope", "PATCH", "", "", "da_stats", ScopeSubmissionsWrite, 403},
		{"bad key", "GET", "owner", "", "da_nope", ScopeStatsRead, 401},
		{"key on session-only", "GET", "", "", "da_stats", "", 403},
	}
	for _, c := range cases {
		_, err := a.Authorize(req(c.method, c.who, c.o, c.k), c.scope)
		if got := status(err); got != c.want {
			t.Errorf("%s: %d, want %d (%v)", c.name, got, c.want, err)
		}
	}

	if _, res, _ := a.PageUser(req("GET", "", "", "")); res != PageSignIn {
		t.Errorf("page anonymous: %v", res)
	}
	if _, res, _ := a.PageUser(req("GET", "guest", "", "")); res != PageForbidden {
		t.Errorf("page guest: %v", res)
	}
	if u, res, _ := a.PageUser(req("GET", "owner", "", "")); res != PageOK || u.Subject != "owner" {
		t.Errorf("page owner: %v", res)
	}

	// Sign-in disabled: pages and session-only endpoints are 503; keys still work.
	off := &Auth{Users: users, Origin: "https://example.test"}
	if _, res, msg := off.PageUser(req("GET", "owner", "", "")); res != PageUnavailable || !strings.Contains(msg, "IDENTITY_DATABASE_URL") {
		t.Errorf("disabled page: %v %q", res, msg)
	}
	if _, err := off.Authorize(req("GET", "", "", ""), ""); status(err) != 503 {
		t.Errorf("disabled session-only: %v", err)
	}
	if _, err := off.Authorize(req("GET", "", "", ""), ScopeStatsRead); status(err) != 401 {
		t.Errorf("disabled, no key: %v", err)
	}
	if _, err := off.Authorize(req("GET", "", "", "da_stats"), ScopeStatsRead); err != nil {
		t.Errorf("disabled, key: %v", err)
	}
}
