package site

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/brennanMKE/WoodDust/internal/content"
	"github.com/brennanMKE/WoodDust/internal/data"
	"github.com/brennanMKE/WoodDust/internal/memstore"
	"github.com/brennanMKE/WoodDust/internal/model"
)

func ptr[T any](v T) *T { return &v }

func fixtureData() *data.Dataset {
	return &data.Dataset{
		Species: []model.Species{
			{ID: "thuja-plicata", Slug: "western-red-cedar", BotanicalName: "Thuja plicata", Family: "Cupressaceae",
				CommonNames: []string{"western red cedar", "WRC", "red cedar"}, WoodType: "softwood", Region: "PNW",
				Description: "A light softwood.", ImagePrompt: "secret prompt", NotesInternal: "secret notes",
				ImageAlt: "Cedar board", FeaturedRank: ptr(2), LastReviewed: "2026-11-01",
				Hazards: []model.Hazard{
					{Type: "carcinogen", Effects: []string{"nasal cancer"}, Route: []string{"inhalation"}, Summary: "All wood dust is a carcinogen.", EvidenceLevel: 0, Sources: []string{"IARC-62"}},
					{Type: "respiratory_sensitizer", Effects: []string{"asthma"}, Route: []string{"inhalation"}, Summary: "Plicatic acid causes asthma <b>bold</b>.", EvidenceLevel: 1, Sources: []string{"ChanYeung-1973"}},
				}},
			{ID: "quercus-spp", Slug: "oak", BotanicalName: "Quercus spp.", Family: "Fagaceae",
				CommonNames: []string{"oak", "red oak"}, WoodType: "hardwood", Region: "N. Hemisphere",
				FeaturedRank: ptr(1), LastReviewed: "2026-11-07",
				Hazards:        []model.Hazard{{Type: "carcinogen", Effects: []string{"cancer"}, Route: []string{"inhalation"}, Summary: "All wood dust is a carcinogen.", EvidenceLevel: 0, Sources: []string{"IARC-62"}}},
				ExposureLimits: []model.ExposureLimit{{Body: "ACGIH TLV", ValueMgM3: 1, Source: "OSHA-CSI"}}},
			{ID: "handroanthus-spp", Slug: "ipe", BotanicalName: "Handroanthus spp.", Family: "Bignoniaceae",
				CommonNames: []string{"ipê", "African teak"}, WoodType: "hardwood", Region: "S. America", LastReviewed: "2026-11-02",
				Hazards: []model.Hazard{{Type: "skin_sensitizer", Effects: []string{"dermatitis"}, Route: []string{"skin"}, Summary: "Reported dermatitis from ipe.", EvidenceLevel: 3, Sources: []string{"OSHA-CSI"}}}},
			{ID: "milicia-excelsa", Slug: "iroko", BotanicalName: "Milicia excelsa", Family: "Moraceae",
				CommonNames: []string{"iroko", "African teak"}, WoodType: "hardwood", Region: "Africa", LastReviewed: "2026-11-02",
				Hazards: []model.Hazard{{Type: "respiratory_sensitizer", Effects: []string{"asthma"}, Route: []string{"inhalation"}, Summary: "Agency lists asthma for iroko.", EvidenceLevel: 2, Sources: []string{"OSHA-CSI"}}}},
		},
		Sources: map[string]model.Source{
			"IARC-62":        {Tier: "A", Citation: "IARC Monograph 62", URL: "https://publications.iarc.who.int/80", Verified: true},
			"ChanYeung-1973": {Tier: "A", Citation: "Chan-Yeung 1973", PMID: "4746581", Verified: true},
			"OSHA-CSI":       {Tier: "A", Citation: "OSHA wood dust", URL: "https://www.osha.gov/chemicaldata/798", Verified: true},
		},
		Summary: model.RegulatorySummary{EUHardwoodMgM3: ptr(2.0), UKHardwoodMgM3: ptr(3.0), UKSoftwoodMgM3: ptr(5.0), NIOSHRELMgM3: ptr(1.0)},
	}
}

const testPage = `---
title: Wood dust risks
slug: risks
description: Why all wood dust is hazardous.
last_reviewed: 2026-11-07
og_image_alt: Dust in a sunbeam.
---
# Wood dust risks

## Cancer

Wood dust causes cancer.[^IARC-62]
`

type testEnv struct {
	srv    *Server
	h      http.Handler
	images string
}

func newEnv(t *testing.T, dist fstest.MapFS) testEnv {
	t.Helper()
	dir := t.TempDir()
	pagesDir := filepath.Join(dir, "pages")
	images := filepath.Join(dir, "images")
	for _, d := range []string{pagesDir, filepath.Join(images, "species")} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(pagesDir, "risks.md"), []byte(testPage), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(images, "species", "oak.jpg"), []byte("jpg"), 0o644); err != nil {
		t.Fatal(err)
	}
	pages, err := content.Load(pagesDir)
	if err != nil {
		t.Fatal(err)
	}
	var distFS = dist
	srv := New(Options{
		Catalog: memstore.New(fixtureData()), Pages: pages, SiteName: "Dust Almanac",
		BaseURL: "https://example.test", ImagesDir: images, Dist: distFS,
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	return testEnv{srv: srv, h: srv.Routes(), images: images}
}

func get(t *testing.T, h http.Handler, path string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	return rec
}

func decode[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	var v T
	if err := json.Unmarshal(rec.Body.Bytes(), &v); err != nil {
		t.Fatalf("decode %s: %v", rec.Body.String(), err)
	}
	return v
}

var initialRE = regexp.MustCompile(`<script type="application/json" id="initial">(.*?)</script>`)

func initial(t *testing.T, body string) map[string]any {
	t.Helper()
	m := initialRE.FindStringSubmatch(body)
	if m == nil {
		t.Fatalf("no #initial block in:\n%s", body)
	}
	var v map[string]any
	if err := json.Unmarshal([]byte(m[1]), &v); err != nil {
		t.Fatal(err)
	}
	return v
}

func TestAPISpeciesList(t *testing.T) {
	env := newEnv(t, nil)
	rec := get(t, env.h, "/api/v1/species")
	if rec.Code != 200 || rec.Header().Get("Content-Type") != "application/json; charset=utf-8" {
		t.Fatalf("status %d type %q", rec.Code, rec.Header().Get("Content-Type"))
	}
	list := decode[speciesList](t, rec)
	var slugs []string
	for _, s := range list.Species {
		slugs = append(slugs, s.Slug)
	}
	if got := strings.Join(slugs, ","); got != "oak,western-red-cedar,ipe,iroko" {
		t.Errorf("order = %s", got)
	}
	oak, cedar := list.Species[0], list.Species[1]
	if oak.MaxEvidenceLevel != nil {
		t.Error("oak has only level 0: max_evidence_level must be null")
	}
	if cedar.MaxEvidenceLevel == nil || *cedar.MaxEvidenceLevel != 1 || cedar.Name != "Western red cedar" {
		t.Errorf("cedar summary %+v", cedar)
	}
	if !strings.HasPrefix(oak.Image.URL, "/images/species/oak.jpg?v=") || cedar.Image.URL != placeholderPath || cedar.Image.Width != 1200 {
		t.Errorf("images: %+v %+v", oak.Image, cedar.Image)
	}

	for q, want := range map[string]int{
		"?featured=true": 2, "?wood_type=softwood": 1, "?hazard_type=skin_sensitizer": 1, "?evidence_level=0": 2,
	} {
		if n := len(decode[speciesList](t, get(t, env.h, "/api/v1/species"+q)).Species); n != want {
			t.Errorf("%s: %d results, want %d", q, n, want)
		}
	}
	if rec := get(t, env.h, "/api/v1/species?wood_type=plastic"); rec.Code != 400 {
		t.Errorf("bad filter status %d", rec.Code)
	}
}

func TestAPISpeciesDetail(t *testing.T) {
	env := newEnv(t, nil)
	rec := get(t, env.h, "/api/v1/species/western-red-cedar")
	if rec.Code != 200 {
		t.Fatal(rec.Code)
	}
	body := rec.Body.String()
	for _, banned := range []string{"notes_internal", "image_prompt", "secret"} {
		if strings.Contains(body, banned) {
			t.Errorf("detail leaks %q", banned)
		}
	}
	var d map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &d)
	srcs := d["sources"].(map[string]any)
	if len(srcs) != 2 || srcs["IARC-62"].(map[string]any)["key"] != "IARC-62" {
		t.Errorf("sources = %v", srcs)
	}
	if d["name"] != "Western red cedar" || d["url"] != "/wood/western-red-cedar" {
		t.Errorf("name/url %v %v", d["name"], d["url"])
	}
	if reg := d["regulatory"].(map[string]any); reg["niosh_rel_mg_m3"] != 1.0 || reg["eu_hardwood_mg_m3"] != 2.0 {
		t.Errorf("regulatory = %v", reg)
	}
	rec = get(t, env.h, "/api/v1/species/wrc")
	if rec.Code != 404 {
		t.Errorf("API alias should 404, got %d", rec.Code)
	}
	e := decode[apiError](t, rec)
	if e.Error.Code != "not_found" {
		t.Errorf("error body %s", rec.Body.String())
	}
}

func TestAPISearch(t *testing.T) {
	env := newEnv(t, nil)
	res := decode[searchResponse](t, get(t, env.h, "/api/v1/search?q=wrc"))
	if len(res.Results) == 0 || res.Results[0].Slug != "western-red-cedar" || res.Results[0].MatchedName != "WRC" || res.Results[0].Score != 1 {
		t.Errorf("wrc: %+v", res)
	}
	res = decode[searchResponse](t, get(t, env.h, "/api/v1/search?q=CED"))
	if len(res.Results) != 1 || res.Results[0].Slug != "western-red-cedar" {
		t.Errorf("ced: %+v", res)
	}
	res = decode[searchResponse](t, get(t, env.h, "/api/v1/search?q=IPE"))
	if len(res.Results) == 0 || res.Results[0].Slug != "ipe" || res.Results[0].MatchKind != "common" {
		t.Errorf("diacritics: %+v", res)
	}
	res = decode[searchResponse](t, get(t, env.h, "/api/v1/search?q=african+teak&limit=1"))
	if len(res.Results) != 1 {
		t.Errorf("limit: %+v", res)
	}
	rec := get(t, env.h, "/api/v1/search?q=")
	if !strings.Contains(rec.Body.String(), `"results":[]`) {
		t.Errorf("empty q: %s", rec.Body.String())
	}
}

func TestAPISourcesPagesHealth(t *testing.T) {
	env := newEnv(t, nil)
	if s := decode[sourcesResponse](t, get(t, env.h, "/api/v1/sources")); len(s.Sources) != 3 {
		t.Errorf("sources: %d", len(s.Sources))
	}
	if s := decode[model.Source](t, get(t, env.h, "/api/v1/sources/OSHA-CSI")); s.Key != "OSHA-CSI" {
		t.Errorf("source: %+v", s)
	}
	if rec := get(t, env.h, "/api/v1/sources/NOPE"); rec.Code != 404 {
		t.Errorf("missing source %d", rec.Code)
	}
	p := decode[pageResponse](t, get(t, env.h, "/api/v1/pages/risks"))
	if p.Slug != "risks" || p.Title != "Wood dust risks" || len(p.TOC) != 1 || p.Sources["IARC-62"].Key != "IARC-62" ||
		!strings.Contains(p.HTML, `href="#src-IARC-62"`) || p.Image.Alt != "Dust in a sunbeam." {
		t.Errorf("page: %+v", p)
	}
	if rec := get(t, env.h, "/api/v1/pages/protection/respirators"); rec.Code != 404 {
		t.Errorf("missing page %d", rec.Code)
	}
	h := decode[healthResponse](t, get(t, env.h, "/api/v1/health"))
	if h.Status != "ok" || h.Species != 4 || h.Sources != 3 {
		t.Errorf("health %+v", h)
	}
	if rec := get(t, env.h, "/api/v1/nothing"); rec.Code != 404 || !strings.Contains(rec.Body.String(), "not_found") {
		t.Errorf("unknown api %d %s", rec.Code, rec.Body.String())
	}
}

func TestHomePage(t *testing.T) {
	env := newEnv(t, nil)
	rec := get(t, env.h, "/")
	body := rec.Body.String()
	if rec.Code != 200 {
		t.Fatal(rec.Code)
	}
	for _, want := range []string{
		"<title>Dust Almanac: wood dust hazards by species</title>",
		`<link rel="canonical" href="https://example.test/">`,
		`"@type":"WebSite"`, `"@type":"SearchAction"`, `search_term_string`,
		`<meta property="og:image" content="https://example.test/images/placeholder.svg">`,
		`<meta property="og:image:width" content="1200">`,
		"<h1>All wood dust is hazardous</h1>",
		`<div id="app">`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("home missing %q", want)
		}
	}
	if strings.Contains(body, "<!--ssr-") || strings.Contains(body, "noindex") {
		t.Error("markers left or unexpected noindex")
	}
	init := initial(t, body)
	if init["route"] != "home" || init["site"].(map[string]any)["name"] != "Dust Almanac" {
		t.Errorf("initial = %v", init)
	}
	if n := len(init["data"].(map[string]any)["species"].([]any)); n != 2 {
		t.Errorf("home data has %d species, want featured only", n)
	}

	rec = get(t, env.h, "/?q=cedar")
	if !strings.Contains(rec.Body.String(), `<meta name="robots" content="noindex">`) || !strings.Contains(rec.Body.String(), `href="/wood/western-red-cedar"`) {
		t.Error("search page should be noindex and list results")
	}
}

func TestSpeciesPage(t *testing.T) {
	env := newEnv(t, nil)
	rec := get(t, env.h, "/wood/western-red-cedar")
	body := rec.Body.String()
	if rec.Code != 200 {
		t.Fatal(rec.Code)
	}
	for _, want := range []string{
		"<title>Western red cedar wood dust hazards | Dust Almanac</title>",
		`<meta property="og:type" content="article">`,
		`<meta property="og:image:alt" content="Cedar board">`,
		`<meta name="twitter:card" content="summary_large_image">`,
		`"@type":"Article"`, `"@type":"BreadcrumbList"`, `"dateModified":"2026-11-01"`, `"citation":[`,
		`<h1 id="species-title">Western red cedar</h1>`,
		"Evidence: <strong>Confirmed in clinical testing</strong>",
		"Evidence: <strong>Applies to all wood dust</strong>",
		"<h3>Asthma / respiratory allergy</h3>",
		`<li id="src-ChanYeung-1973">`, `https://pubmed.ncbi.nlm.nih.gov/4746581/`,
		"asthma &lt;b&gt;bold&lt;/b&gt;", // data is escaped
		"Also known as: WRC, red cedar",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("species page missing %q", want)
		}
	}
	if strings.Contains(body, "secret") {
		t.Error("species page leaks internal fields")
	}
	init := initial(t, body)
	if init["route"] != "species" || init["data"].(map[string]any)["slug"] != "western-red-cedar" {
		t.Errorf("initial = %v", init["route"])
	}
	// The data block must not be breakable by a </script> inside data.
	if strings.Count(body, "</script>") != strings.Count(body, "<script") {
		t.Error("unbalanced script tags")
	}

	// og:image uses the species image when it exists.
	if !strings.Contains(get(t, env.h, "/wood/oak").Body.String(), `content="https://example.test/images/species/oak.jpg?v=`) {
		t.Error("oak og:image should be its photo")
	}
}

func TestAliasRedirectsAndNotFound(t *testing.T) {
	env := newEnv(t, nil)
	for path, want := range map[string]string{
		"/wood/wrc":           "/wood/western-red-cedar",
		"/wood/thuja-plicata": "/wood/western-red-cedar",
		"/wood/Red-Cedar":     "/wood/western-red-cedar",
		"/wood/quercus-spp":   "/wood/oak",
		"/wood/oak/":          "/wood/oak",
	} {
		rec := get(t, env.h, path)
		if rec.Code != http.StatusMovedPermanently || rec.Header().Get("Location") != want {
			t.Errorf("%s: %d -> %q, want 301 -> %s", path, rec.Code, rec.Header().Get("Location"), want)
		}
	}
	rec := get(t, env.h, "/wood/african-teak")
	if rec.Code != 404 || !strings.Contains(rec.Body.String(), `href="/wood/ipe"`) || !strings.Contains(rec.Body.String(), `href="/wood/iroko"`) {
		t.Errorf("ambiguous alias: %d", rec.Code)
	}
	for _, p := range []string{"/wood/nope", "/nothing", "/protection/respirators", "/print/missing"} {
		rec := get(t, env.h, p)
		if rec.Code != 404 {
			t.Errorf("%s: %d, want 404", p, rec.Code)
		}
		if !strings.Contains(rec.Body.String(), `<meta name="robots" content="noindex">`) || rec.Header().Get("X-Robots-Tag") != "noindex" {
			t.Errorf("%s: 404 must be noindex", p)
		}
		if initial(t, rec.Body.String())["route"] != "not_found" {
			t.Errorf("%s: route should be not_found", p)
		}
	}
}

func TestContentBrowseSourcesPages(t *testing.T) {
	env := newEnv(t, nil)
	rec := get(t, env.h, "/risks")
	body := rec.Body.String()
	for _, want := range []string{
		"<title>Wood dust risks | Dust Almanac</title>", `"@type":"Article"`, `"@type":"BreadcrumbList"`,
		`<h2 id="cancer">Cancer</h2>`, `<li id="src-IARC-62">`, `<link rel="canonical" href="https://example.test/risks">`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("risks missing %q", want)
		}
	}
	if strings.Count(body, "<h1") != 1 {
		t.Error("risks should have exactly one h1")
	}
	if initial(t, body)["route"] != "page" {
		t.Error("risks route id")
	}
	rec = get(t, env.h, "/browse")
	if rec.Code != 200 || initial(t, rec.Body.String())["route"] != "browse" || !strings.Contains(rec.Body.String(), "<caption>") {
		t.Errorf("browse %d", rec.Code)
	}
	rec = get(t, env.h, "/sources")
	if rec.Code != 200 || initial(t, rec.Body.String())["route"] != "sources" || !strings.Contains(rec.Body.String(), `id="src-OSHA-CSI"`) {
		t.Errorf("sources %d", rec.Code)
	}
}

func TestSitemapRobots(t *testing.T) {
	env := newEnv(t, nil)
	sm := get(t, env.h, "/sitemap.xml").Body.String()
	for _, want := range []string{
		"<loc>https://example.test/wood/oak</loc>", "<lastmod>2026-11-07</lastmod>",
		"<loc>https://example.test/risks</loc>", "<loc>https://example.test/browse</loc>",
	} {
		if !strings.Contains(sm, want) {
			t.Errorf("sitemap missing %q", want)
		}
	}
	rb := get(t, env.h, "/robots.txt").Body.String()
	for _, want := range []string{"Disallow: /submit", "Disallow: /admin", "Disallow: /account", "Sitemap: https://example.test/sitemap.xml"} {
		if !strings.Contains(rb, want) {
			t.Errorf("robots missing %q", want)
		}
	}
}

func TestShellAndStatic(t *testing.T) {
	dist := fstest.MapFS{
		"index.html":          {Data: []byte("<html><head><!--ssr-head--><script type=module src=/assets/app-1.js></script></head><body><div id=app><!--ssr-body--></div><!--ssr-data--></body></html>")},
		"assets/app-1.js":     {Data: []byte("console.log(1)")},
		"robots-not-used.txt": {Data: []byte("x")},
	}
	env := newEnv(t, dist)
	body := get(t, env.h, "/").Body.String()
	if !strings.Contains(body, "/assets/app-1.js") || !strings.Contains(body, "<title>") {
		t.Error("dist shell not used")
	}
	rec := get(t, env.h, "/assets/app-1.js")
	if rec.Code != 200 || !strings.Contains(rec.Header().Get("Cache-Control"), "immutable") {
		t.Errorf("asset %d %q", rec.Code, rec.Header().Get("Cache-Control"))
	}
	if rec := get(t, env.h, "/assets/missing.js"); rec.Code != 404 {
		t.Errorf("missing asset %d", rec.Code)
	}
	if rec := get(t, env.h, "/images/species/oak.jpg"); rec.Code != 200 {
		t.Errorf("image %d", rec.Code)
	}
	for _, p := range []string{"/images/species/", "/images/../go.mod", "/images/species/nope.jpg"} {
		if rec := get(t, env.h, p); rec.Code == 200 {
			t.Errorf("%s should not be served", p)
		}
	}
	if rec := get(t, env.h, "/favicon.ico"); rec.Code != 404 {
		t.Errorf("favicon without art: %d", rec.Code)
	}
	if err := os.MkdirAll(filepath.Join(env.images, "art"), 0o755); err != nil {
		t.Fatal(err)
	}
	_ = os.WriteFile(filepath.Join(env.images, "art", "favicon.png"), []byte("\x89PNG\r\n\x1a\n"), 0o644)
	_ = os.WriteFile(filepath.Join(env.images, "og-default.jpg"), []byte("jpg"), 0o644)
	if rec := get(t, env.h, "/favicon.ico"); rec.Code != 200 {
		t.Errorf("favicon with art: %d", rec.Code)
	}
	cedar := get(t, env.h, "/wood/western-red-cedar").Body.String()
	if !strings.Contains(cedar, `<meta property="og:image" content="https://example.test/images/og-default.jpg">`) {
		t.Error("og:image should fall back to og-default.jpg once it exists")
	}
	if !strings.Contains(cedar, `<link rel="icon" type="image/png" href="/images/art/favicon.png">`) {
		t.Error("favicon link missing")
	}
	// API image.url keeps the placeholder (og-default is for share cards only).
	if d := decode[speciesDetail](t, get(t, env.h, "/api/v1/species/western-red-cedar")); d.Image.URL != placeholderPath {
		t.Errorf("api image %q", d.Image.URL)
	}
}

func TestFallbackShellWithoutDist(t *testing.T) {
	env := newEnv(t, fstest.MapFS{".gitkeep": {Data: nil}})
	body := get(t, env.h, "/wood/oak").Body.String()
	if !strings.HasPrefix(body, "<!doctype html>") || !strings.Contains(body, `<div id="app"><a class="skip-link"`) {
		t.Errorf("fallback shell not used:\n%.300s", body)
	}
}

func TestMethodNotAllowed(t *testing.T) {
	env := newEnv(t, nil)
	rec := httptest.NewRecorder()
	env.h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/wood/oak", nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("POST: %d", rec.Code)
	}
}

func TestSubmitPages(t *testing.T) {
	env := newEnv(t, nil)
	for _, p := range []string{"/submit", "/submit/oak", "/submit?name=Bubinga"} {
		rec := get(t, env.h, p)
		body := rec.Body.String()
		if rec.Code != 200 || !strings.Contains(body, `<meta name="robots" content="noindex">`) {
			t.Errorf("%s: %d noindex=%v", p, rec.Code, strings.Contains(body, "noindex"))
		}
		init := initial(t, body)
		data, _ := init["data"].(map[string]any)
		if init["route"] != "submit" || data == nil || len(data["kinds"].([]any)) != 4 {
			t.Errorf("%s: initial %v", p, init)
		}
		if p == "/submit/oak" && data["target"].(map[string]any)["slug"] != "oak" {
			t.Errorf("target: %v", data["target"])
		}
		if p == "/submit?name=Bubinga" && data["prefill_name"] != "Bubinga" {
			t.Errorf("prefill: %v", data["prefill_name"])
		}
	}
	if rec := get(t, env.h, "/submit/unknown"); rec.Code != 404 {
		t.Errorf("/submit/unknown: %d", rec.Code)
	}
}
