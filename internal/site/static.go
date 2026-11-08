package site

import (
	"encoding/xml"
	"io/fs"
	"net/http"
	"path"
	"strings"
)

func statFS(fsys fs.FS, name string) (fs.FileInfo, error) {
	if !fs.ValidPath(name) {
		return nil, fs.ErrInvalid
	}
	return fs.Stat(fsys, name)
}

// serveFile serves name from fsys, 404 for directories and missing files.
func serveFile(w http.ResponseWriter, r *http.Request, fsys fs.FS, name, cache string) bool {
	if fsys == nil {
		return false
	}
	name = path.Clean(name)
	st, err := statFS(fsys, name)
	if err != nil || st.IsDir() {
		return false
	}
	w.Header().Set("Cache-Control", cache)
	http.ServeFileFS(w, r, fsys, name)
	return true
}

// serveImage serves IMAGES_DIR in development; in production Apache serves
// /images/ directly (deploy/apache). Missing files are a plain 404: the API
// only ever points at files that exist or at /images/placeholder.svg.
func (s *Server) serveImage(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("path")
	if strings.HasPrefix(path.Base(name), "_") || strings.HasPrefix(name, "prompts/") {
		http.NotFound(w, r) // reference art and prompt notes are not published
		return
	}
	// Images never run script, even an SVG opened directly.
	w.Header().Set("Content-Security-Policy", "default-src 'none'; img-src 'self'; style-src 'unsafe-inline'; sandbox")
	if !serveFile(w, r, s.images, name, "public, max-age=86400") {
		w.Header().Del("Content-Security-Policy")
		http.NotFound(w, r)
	}
}

// serveAsset serves Vite's hashed build output with an immutable cache.
func (s *Server) serveAsset(w http.ResponseWriter, r *http.Request) {
	if !serveFile(w, r, s.dist, "assets/"+r.PathValue("path"), "public, max-age=31536000, immutable") {
		http.NotFound(w, r)
	}
}

// favicon serves images/art/favicon.png when present, else dist/favicon.ico,
// else 404.
func (s *Server) favicon(w http.ResponseWriter, r *http.Request) {
	if serveFile(w, r, s.images, "art/favicon.png", "public, max-age=86400") {
		return
	}
	if serveFile(w, r, s.dist, "favicon.ico", "public, max-age=86400") {
		return
	}
	http.NotFound(w, r)
}

// ---- robots and sitemap -------------------------------------------------------

func (s *Server) robots(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	_, _ = w.Write([]byte("User-agent: *\n" +
		"Disallow: /submit\n" +
		"Disallow: /account\n" +
		"Disallow: /admin\n" +
		"Disallow: /auth/\n" +
		"Disallow: /api/\n" +
		"Disallow: /*?q=\n" +
		"\nSitemap: " + s.absURL("/sitemap.xml") + "\n"))
}

type sitemapURL struct {
	Loc     string `xml:"loc"`
	LastMod string `xml:"lastmod,omitempty"`
}

type urlset struct {
	XMLName xml.Name     `xml:"urlset"`
	XMLNS   string       `xml:"xmlns,attr"`
	URLs    []sitemapURL `xml:"url"`
}

func (s *Server) sitemap(w http.ResponseWriter, r *http.Request) {
	all, err := s.cat.ListSpecies(r.Context())
	if err != nil {
		s.log.Error("sitemap", "err", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	set := urlset{XMLNS: "http://www.sitemaps.org/schemas/sitemap/0.9"}
	latest := ""
	for _, sp := range all {
		if sp.LastReviewed > latest {
			latest = sp.LastReviewed
		}
	}
	set.URLs = append(set.URLs,
		sitemapURL{Loc: s.absURL("/"), LastMod: latest},
		sitemapURL{Loc: s.absURL("/browse"), LastMod: latest},
		sitemapURL{Loc: s.absURL("/sources")},
	)
	for _, p := range s.pages.All() {
		if _, ok := s.lookupPage(p.Slug); ok {
			set.URLs = append(set.URLs, sitemapURL{Loc: s.absURL("/" + p.Slug), LastMod: p.LastReviewed})
		}
	}
	for _, sp := range all {
		set.URLs = append(set.URLs, sitemapURL{Loc: s.absURL("/wood/" + sp.Slug), LastMod: sp.LastReviewed})
	}
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	_, _ = w.Write([]byte(xml.Header))
	enc := xml.NewEncoder(w)
	enc.Indent("", "  ")
	_ = enc.Encode(set)
	_, _ = w.Write([]byte("\n"))
}
