// Package site holds the HTTP handlers: the public JSON API under /api/v1
// and the server-rendered page routes that fill the SPA shell's
// <!--ssr-head-->, <!--ssr-body--> and <!--ssr-data--> markers
// (docs/contract.md). Handlers depend only on the narrow Catalog interface,
// so tests run against internal/memstore with no database.
//
// Submissions, the admin API and API keys come from internal/contrib
// (Options.Contrib); Portiko's identity routes are mounted on the same mux
// by internal/identity.
package site

import (
	"context"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/brennanMKE/WoodDust/internal/content"
	"github.com/brennanMKE/WoodDust/internal/contrib"
	"github.com/brennanMKE/WoodDust/internal/model"
)

// Catalog is the read-only published data the handlers need.
type Catalog interface {
	ListSpecies(ctx context.Context) ([]model.Species, error)
	GetSpecies(ctx context.Context, slug string) (model.Species, bool, error)
	// ResolveName returns every species slug that has this name (normalized).
	ResolveName(ctx context.Context, name string) ([]string, error)
	Search(ctx context.Context, q string, limit int) ([]model.SearchHit, error)
	Sources(ctx context.Context) (map[string]model.Source, error)
	Regulatory(ctx context.Context) (model.RegulatorySummary, error)
	Ping(ctx context.Context) error
}

// Options configures a Server.
type Options struct {
	Catalog   Catalog
	Pages     *content.Set
	SiteName  string
	BaseURL   string // origin, no trailing slash
	ImagesDir string
	// Dist is the built SPA (web.DistFS()). If it has no usable index.html
	// the built-in shell is used.
	Dist   fs.FS
	Logger *slog.Logger
	// Contrib serves submissions, admin and API keys (contract §6–7). Nil
	// leaves those routes unregistered (tests of the public site).
	Contrib *contrib.API
}

// Server serves the site.
type Server struct {
	cat       Catalog
	pages     *content.Set
	siteName  string
	baseURL   string
	imagesDir string
	images    fs.FS
	dist      fs.FS
	shell     shell
	log       *slog.Logger
	contrib   *contrib.API
}

// New builds a Server.
func New(o Options) *Server {
	if o.Logger == nil {
		o.Logger = slog.Default()
	}
	s := &Server{
		cat:       o.Catalog,
		pages:     o.Pages,
		siteName:  o.SiteName,
		baseURL:   strings.TrimRight(o.BaseURL, "/"),
		imagesDir: o.ImagesDir,
		dist:      o.Dist,
		log:       o.Logger,
		contrib:   o.Contrib,
	}
	if s.siteName == "" {
		s.siteName = DefaultSiteName
	}
	if o.ImagesDir != "" {
		s.images = os.DirFS(o.ImagesDir)
	}
	var usingFallback bool
	s.shell, usingFallback = loadShell(o.Dist)
	if usingFallback {
		s.log.Warn("web/dist/index.html missing or lacks SSR markers; using built-in shell (no SPA)")
	}
	return s
}

// DefaultSiteName is used when SITE_NAME is unset.
const DefaultSiteName = "Dust Almanac"

// pageRoutes are the education page paths (contract §2, route id "page").
// /protection/{sub} and /print/{sub} (the shop poster) are pages too.
var pageRoutes = []string{"/risks", "/protection", "/symptoms", "/limits", "/glossary", "/methodology"}

// Routes returns the mux with every route.
func (s *Server) Routes() *http.ServeMux {
	mux := http.NewServeMux()

	// API (contract §3).
	mux.HandleFunc("GET /api/v1/species", s.apiSpeciesList)
	mux.HandleFunc("GET /api/v1/species/{slug}", s.apiSpecies)
	mux.HandleFunc("GET /api/v1/search", s.apiSearch)
	mux.HandleFunc("GET /api/v1/sources", s.apiSources)
	mux.HandleFunc("GET /api/v1/sources/{key}", s.apiSource)
	mux.HandleFunc("GET /api/v1/pages/{path...}", s.apiPage)
	mux.HandleFunc("GET /api/v1/health", s.apiHealth)
	mux.HandleFunc("GET /health", s.apiHealth)

	// Pages (contract §2).
	mux.HandleFunc("GET /{$}", s.pageHome)
	mux.HandleFunc("GET /wood/{slug}", s.pageSpecies)
	mux.HandleFunc("GET /browse", s.pageBrowse)
	mux.HandleFunc("GET /sources", s.pageSources)
	for _, p := range pageRoutes {
		mux.HandleFunc("GET "+p, s.pageContent)
	}
	mux.HandleFunc("GET /protection/{sub}", s.pageContent)
	mux.HandleFunc("GET /print/{sub}", s.pageContent)
	// Contributions (contract §2, §6) and the admin console (§7).
	mux.HandleFunc("GET /submit", s.pageSubmit)
	mux.HandleFunc("GET /submit/{slug}", s.pageSubmit)
	mux.HandleFunc("GET /submit/status/{token}", s.pageSubmissionStatus)
	mux.HandleFunc("GET /admin", s.pageAdmin)
	mux.HandleFunc("GET /admin/submissions", s.pageAdmin)
	mux.HandleFunc("GET /admin/submissions/{id}", s.pageAdmin)
	mux.HandleFunc("GET /admin/api-keys", s.pageAdmin)
	if s.contrib != nil {
		s.contrib.Register(mux)
	}

	// SEO and static.
	mux.HandleFunc("GET /sitemap.xml", s.sitemap)
	mux.HandleFunc("GET /robots.txt", s.robots)
	mux.HandleFunc("GET /images/{path...}", s.serveImage)
	mux.HandleFunc("GET /assets/{path...}", s.serveAsset)
	mux.HandleFunc("GET /favicon.ico", s.favicon)
	// Catch-all for every method: unknown /api/ paths get a JSON 404,
	// everything else the HTML 404 (or a dist file).
	mux.HandleFunc("/", s.fallback)
	return mux
}

// fallback serves other files from dist (favicon etc.), redirects trailing
// slashes, and otherwise renders the 404 page.
func (s *Server) fallback(w http.ResponseWriter, r *http.Request) {
	p := r.URL.Path
	if strings.HasPrefix(p, "/api/") {
		s.apiNotFound(w, r)
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if len(p) > 1 && strings.HasSuffix(p, "/") {
		http.Redirect(w, r, strings.TrimRight(p, "/"), http.StatusMovedPermanently)
		return
	}
	name := strings.TrimPrefix(path.Clean(p), "/")
	if s.dist != nil && name != "" && name != "index.html" && fs.ValidPath(name) {
		if st, err := fs.Stat(s.dist, name); err == nil && !st.IsDir() {
			w.Header().Set("Cache-Control", "public, max-age=3600")
			http.ServeFileFS(w, r, s.dist, name)
			return
		}
	}
	s.renderNotFound(w, r, "", nil)
}

// absURL turns a site path into an absolute URL.
func (s *Server) absURL(p string) string { return s.baseURL + p }
