package site

import (
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/brennanMKE/WoodDust/internal/memstore"
	posterfiles "github.com/brennanMKE/WoodDust/posters"
)

func TestPostersPage(t *testing.T) {
	e := newEnv(t, nil)
	rec := get(t, e.h, "/posters")
	if rec.Code != http.StatusOK || rec.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("/posters: %d %q", rec.Code, rec.Header().Get("Cache-Control"))
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Paul shaves for a reason.") || !strings.Contains(body, "breathing.") || !strings.Contains(body, "No brooms.") {
		t.Error("all five published posters, with slogans")
	}
	init := initial(t, body)
	if init["route"] != "posters" {
		t.Fatalf("route %v", init["route"])
	}
	data := init["data"].(map[string]any)
	ps := data["posters"].([]any)
	if len(ps) != 5 {
		t.Fatalf("%d posters", len(ps))
	}
	shave := ps[0].(map[string]any)
	art := shave["art"].(map[string]any)
	if art["available"] != false || shave["stock"].(map[string]any)["short_url"] != "https://go.dustalmanac.com/u/shave" {
		t.Errorf("shave: %v", shave)
	}
	tmpl := shave["template"].(string)
	if !strings.HasPrefix(tmpl, "<svg") || strings.Contains(tmpl, "<!--") || strings.Contains(tmpl, "style=") {
		t.Errorf("template not cleaned: %.80s", tmpl)
	}
	if len(shave["downloads"].([]any)) != 0 || data["custom"] != false {
		t.Error("no downloads or custom links without files and a link service")
	}

	// Print files appear once scripts/posters.sh has written them.
	dir := filepath.Join(e.images, "posters")
	_ = os.MkdirAll(dir, 0o755)
	_ = os.WriteFile(filepath.Join(dir, "shave-letter.pdf"), []byte("%PDF"), 0o644)
	e = newEnvWithImages(t, e.images)
	v := decode[postersResponse](t, get(t, e.h, "/api/v1/posters"))
	if len(v.Posters[0].Downloads) != 1 || !strings.HasPrefix(v.Posters[0].Downloads[0].URL, "/images/posters/shave-letter.pdf?v=") {
		t.Errorf("downloads %+v", v.Posters[0].Downloads)
	}
}

func TestPosterPrintAndAPI(t *testing.T) {
	e := newEnv(t, nil)
	rec := get(t, e.h, "/posters/sunbeam/print")
	if rec.Code != http.StatusOK || rec.Header().Get("X-Robots-Tag") != "noindex" || initial(t, rec.Body.String())["route"] != "poster_print" {
		t.Fatalf("print: %d", rec.Code)
	}
	for _, p := range []string{"/posters/fire/print", "/posters/nope/print"} {
		if rec := get(t, e.h, p); rec.Code != http.StatusNotFound {
			t.Errorf("%s: %d", p, rec.Code)
		}
	}
	if rec := get(t, e.h, "/api/v1/posters/fire"); rec.Code != http.StatusNotFound {
		t.Errorf("unknown poster API: %d", rec.Code)
	}
	v := decode[posterResponse](t, get(t, e.h, "/api/v1/posters/shave"))
	if v.Poster.Destination != "/protection/respirators" || !strings.Contains(v.Poster.Stock.QRSVG, "<svg") {
		t.Errorf("%+v", v.Poster.Poster)
	}
	if !strings.Contains(get(t, e.h, "/sitemap.xml").Body.String(), "/posters</loc>") {
		t.Error("sitemap lacks /posters")
	}
}

// newEnvWithImages rebuilds the server over an existing images dir (image
// hashes are computed at startup).
func newEnvWithImages(t *testing.T, images string) testEnv {
	t.Helper()
	srv := New(Options{
		Catalog: memstore.New(fixtureData()), SiteName: "Dust Almanac", BaseURL: "https://example.test",
		ImagesDir: images, Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), PosterFS: posterfiles.FS,
	})
	return testEnv{srv: srv, h: srv.Routes(), images: images}
}
