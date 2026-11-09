package site

import (
	"encoding/json"
	"html/template"
	"io/fs"
	"net/http"
	"regexp"
	"strings"

	"github.com/brennanMKE/WoodDust/internal/posters"
)

// Shop posters (docs/posters.md, contract §2 and §3).

type posterArt struct {
	URL       string `json:"url"`
	Available bool   `json:"available"` // false: the template's placeholder block shows
	Width     int    `json:"width"`
	Height    int    `json:"height"`
}

type posterDownload struct {
	Label  string `json:"label"`
	Format string `json:"format"` // pdf, png, svg
	URL    string `json:"url"`
}

type posterView struct {
	posters.Poster
	Art       posterArt        `json:"art"`
	Template  string           `json:"template"` // posters/src/<id>.svg, comments stripped
	Stock     posters.Link     `json:"stock"`
	Downloads []posterDownload `json:"downloads"`
}

type postersResponse struct {
	Posters []posterView `json:"posters"`
	// FormToken is the min-time token POST /api/v1/posters/links needs.
	FormToken string `json:"form_token"`
	// Custom: the plaque builder can ask for its own link (otherwise it
	// uses the stock link).
	Custom bool `json:"custom"`
}

type posterResponse struct {
	Poster posterView `json:"poster"`
}

var (
	reSVGComment = regexp.MustCompile(`(?s)<!--.*?-->\s*`)
	reSVGDecl    = regexp.MustCompile(`^\s*<\?xml[^>]*\?>\s*`)
)

// posterDownloads lists the print files scripts/posters.sh wrote for p.
var posterDownloads = []struct{ suffix, label, format string }{
	{"-letter.pdf", "Letter PDF", "pdf"},
	{"-a4.pdf", "A4 PDF", "pdf"},
	{"-grey.pdf", "Greyscale PDF (Letter)", "pdf"},
	{".png", "PNG, 300 dpi (Letter)", "png"},
	{".svg", "SVG source", "svg"},
}

func (s *Server) posterView(p posters.Poster) posterView {
	v := posterView{Poster: p, Downloads: []posterDownload{}, Art: posterArt{Width: 1024, Height: 1536}}
	rel := "art/posters/" + p.ID + ".png"
	v.Art.URL = s.imageURL(rel)
	v.Art.Available = s.imageExists(rel)
	if s.posterFS != nil {
		if b, err := fs.ReadFile(s.posterFS, "src/"+p.ID+".svg"); err == nil {
			v.Template = strings.TrimSpace(reSVGComment.ReplaceAllString(reSVGDecl.ReplaceAllString(string(b), ""), ""))
		}
	}
	v.Stock = posters.StockLink(s.posterFS, p)
	for _, d := range posterDownloads {
		if rel := "posters/" + p.ID + d.suffix; s.imageExists(rel) {
			v.Downloads = append(v.Downloads, posterDownload{Label: d.label, Format: d.format, URL: s.imageURL(rel)})
		}
	}
	return v
}

func (s *Server) postersData() postersResponse {
	out := postersResponse{Posters: []posterView{}}
	for _, p := range posters.Published() {
		out.Posters = append(out.Posters, s.posterView(p))
	}
	if s.posterLinks != nil && s.posterLinks.Tokens != nil {
		out.FormToken = s.posterLinks.Tokens.Issue()
		out.Custom = true
	}
	return out
}

func publishedPoster(id string) (posters.Poster, bool) {
	p, ok := posters.Get(id)
	return p, ok && p.Published
}

func (s *Server) apiPosters(w http.ResponseWriter, r *http.Request) {
	writeJSONNoCache(w, s.postersData()) // carries a fresh form token
}

func (s *Server) apiPoster(w http.ResponseWriter, r *http.Request) {
	p, ok := publishedPoster(r.PathValue("id"))
	if !ok {
		writeError(w, http.StatusNotFound, "not_found", "No such poster.")
		return
	}
	writeJSON(w, http.StatusOK, posterResponse{Poster: s.posterView(p)})
}

func writeJSONNoCache(w http.ResponseWriter, v any) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(v)
}

type postersBody struct {
	Posters []posterView
	Custom  bool
}

func (s *Server) pagePosters(w http.ResponseWriter, r *http.Request) {
	data := s.postersData()
	canonical := s.absURL("/posters")
	crumbs := []crumb{{"Home", "/"}, {"Shop posters", ""}}
	head := headData{
		Title:       "Shop posters | " + s.siteName,
		OGTitle:     "Free wood dust posters for your shop",
		Description: "Free printable posters for wood shops: Paul Bunyan and Babe on dust and respirators, with a QR code to the facts. Letter and A4 PDFs.",
		Canonical:   canonical,
		JSONLD:      []template.JS{s.breadcrumbLD(crumbs, canonical)},
	}
	w.Header().Set("Cache-Control", "no-store") // the form token
	s.render(w, r, view{Route: "posters", Head: head, Template: "body_posters", Breadcrumbs: crumbs,
		Body: postersBody{Posters: data.Posters, Custom: data.Custom}, Data: data})
}

func (s *Server) pagePosterPrint(w http.ResponseWriter, r *http.Request) {
	p, ok := publishedPoster(r.PathValue("id"))
	if !ok {
		s.renderNotFound(w, r, "", nil)
		return
	}
	v := s.posterView(p)
	head := headData{
		Title:       "Print: " + p.Title + " poster | " + s.siteName,
		Description: "A print layout for the " + p.Title + " shop poster.",
		Canonical:   s.absURL("/posters"),
		NoIndex:     true,
	}
	s.render(w, r, view{Route: "poster_print", Head: head, Template: "body_poster_print",
		Body: v, Data: posterResponse{Poster: v}})
}
