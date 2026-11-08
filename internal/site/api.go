package site

import (
	"encoding/json"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/brennanMKE/WoodDust/internal/content"
	"github.com/brennanMKE/WoodDust/internal/model"
)

// ---- views (contract §3) -------------------------------------------------

type imageView struct {
	URL    string `json:"url"`
	Alt    string `json:"alt"`
	Width  int    `json:"width,omitempty"`
	Height int    `json:"height,omitempty"`
}

type speciesSummary struct {
	Slug             string    `json:"slug"`
	Name             string    `json:"name"`
	CommonNames      []string  `json:"common_names"`
	BotanicalName    string    `json:"botanical_name"`
	WoodType         string    `json:"wood_type"`
	Region           string    `json:"region"`
	FeaturedRank     *int      `json:"featured_rank"`
	HazardTypes      []string  `json:"hazard_types"`
	MaxEvidenceLevel *int      `json:"max_evidence_level"`
	Image            imageView `json:"image"`
	URL              string    `json:"url"`
}

type speciesList struct {
	Species []speciesSummary `json:"species"`
}

type speciesDetail struct {
	model.Species
	Name       string                  `json:"name"`
	URL        string                  `json:"url"`
	Image      imageView               `json:"image"`
	Sources    map[string]model.Source `json:"sources"`
	Regulatory model.RegulatorySummary `json:"regulatory"`
}

type searchResult struct {
	Slug          string  `json:"slug"`
	Name          string  `json:"name"`
	BotanicalName string  `json:"botanical_name"`
	MatchedName   string  `json:"matched_name"`
	MatchKind     string  `json:"match_kind"`
	Score         float64 `json:"score"`
	URL           string  `json:"url"`
}

type searchResponse struct {
	Query   string         `json:"query"`
	Results []searchResult `json:"results"`
}

type sourcesResponse struct {
	Sources map[string]model.Source `json:"sources"`
}

type pageResponse struct {
	Slug         string                  `json:"slug"`
	Title        string                  `json:"title"`
	Description  string                  `json:"description"`
	LastReviewed string                  `json:"last_reviewed"`
	HTML         string                  `json:"html"`
	TOC          []content.TOCEntry      `json:"toc"`
	Sources      map[string]model.Source `json:"sources"`
	Image        imageView               `json:"image"`
}

type healthResponse struct {
	Status  string `json:"status"`
	Species int    `json:"species"`
	Sources int    `json:"sources"`
}

type apiError struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// ---- helpers ---------------------------------------------------------------

const (
	imageWidth      = 1200
	imageHeight     = 630
	placeholderPath = "/images/placeholder.svg"
)

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if status == http.StatusOK {
		w.Header().Set("Cache-Control", "public, max-age=300")
	} else {
		w.Header().Set("Cache-Control", "no-store")
	}
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	_ = enc.Encode(v)
}

func writeError(w http.ResponseWriter, status int, code, msg string) {
	var e apiError
	e.Error.Code = code
	e.Error.Message = msg
	writeJSON(w, status, e)
}

func (s *Server) internalError(w http.ResponseWriter, r *http.Request, err error) {
	s.log.Error("handler error", "path", r.URL.Path, "err", err)
	writeError(w, http.StatusInternalServerError, "internal", "Something went wrong.")
}

// imageExists reports whether IMAGES_DIR has the file (rel has no leading /).
func (s *Server) imageExists(rel string) bool {
	if s.images == nil {
		return false
	}
	st, err := statFS(s.images, rel)
	return err == nil && !st.IsDir()
}

func (s *Server) speciesImage(sp model.Species) imageView {
	img := imageView{URL: placeholderPath, Alt: sp.ImageAlt, Width: imageWidth, Height: imageHeight}
	if s.imageExists("species/" + sp.Slug + ".jpg") {
		img.URL = "/images/species/" + sp.Slug + ".jpg"
	}
	return img
}

func (s *Server) pageImage(p *content.Page) imageView {
	img := imageView{URL: placeholderPath, Alt: p.ImageAlt, Width: imageWidth, Height: imageHeight}
	if name := "pages/" + p.Key() + ".jpg"; s.imageExists(name) {
		img.URL = "/images/" + name
	}
	return img
}

func (s *Server) summary(sp model.Species) speciesSummary {
	names := sp.CommonNames
	if names == nil {
		names = []string{}
	}
	return speciesSummary{
		Slug: sp.Slug, Name: sp.DisplayName(), CommonNames: names, BotanicalName: sp.BotanicalName,
		WoodType: sp.WoodType, Region: sp.Region, FeaturedRank: sp.FeaturedRank,
		HazardTypes: sp.HazardTypes(), MaxEvidenceLevel: sp.MaxEvidenceLevel(),
		Image: s.speciesImage(sp), URL: "/wood/" + sp.Slug,
	}
}

// speciesFilter holds the GET /species query filters.
type speciesFilter struct {
	featured      bool
	woodType      string
	hazardType    string
	evidenceLevel *int
}

func parseSpeciesFilter(r *http.Request) (speciesFilter, string) {
	q := r.URL.Query()
	var f speciesFilter
	switch strings.ToLower(q.Get("featured")) {
	case "", "false", "0":
	case "true", "1":
		f.featured = true
	default:
		return f, "featured must be true or false"
	}
	f.woodType = q.Get("wood_type")
	if f.woodType != "" && f.woodType != "hardwood" && f.woodType != "softwood" {
		return f, "wood_type must be hardwood or softwood"
	}
	f.hazardType = q.Get("hazard_type")
	if f.hazardType != "" {
		if _, ok := model.HazardTypeNames[f.hazardType]; !ok {
			return f, "unknown hazard_type"
		}
	}
	if v := q.Get("evidence_level"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 || n > 3 {
			return f, "evidence_level must be 0, 1, 2 or 3"
		}
		f.evidenceLevel = &n
	}
	return f, ""
}

func (f speciesFilter) match(sp model.Species) bool {
	if f.featured && sp.FeaturedRank == nil {
		return false
	}
	if f.woodType != "" && sp.WoodType != f.woodType {
		return false
	}
	if f.hazardType != "" && !sp.HasHazardType(f.hazardType) {
		return false
	}
	if f.evidenceLevel != nil && !sp.HasEvidenceLevel(*f.evidenceLevel) {
		return false
	}
	return true
}

func (s *Server) listSpecies(r *http.Request, f speciesFilter) (speciesList, error) {
	all, err := s.cat.ListSpecies(r.Context())
	if err != nil {
		return speciesList{}, err
	}
	out := speciesList{Species: []speciesSummary{}}
	for _, sp := range all {
		if f.match(sp) {
			out.Species = append(out.Species, s.summary(sp))
		}
	}
	return out, nil
}

func (s *Server) speciesDetail(r *http.Request, sp model.Species) (speciesDetail, error) {
	srcs, err := s.cat.Sources(r.Context())
	if err != nil {
		return speciesDetail{}, err
	}
	reg, err := s.cat.Regulatory(r.Context())
	if err != nil {
		return speciesDetail{}, err
	}
	return speciesDetail{
		Species:    sp.Public(),
		Name:       sp.DisplayName(),
		URL:        "/wood/" + sp.Slug,
		Image:      s.speciesImage(sp),
		Sources:    pick(srcs, sp.CitedKeys()),
		Regulatory: reg,
	}, nil
}

// pick returns the registry entries for keys (unknown keys are skipped;
// seed's integrity check makes that impossible for published data).
func pick(all map[string]model.Source, keys []string) map[string]model.Source {
	out := make(map[string]model.Source, len(keys))
	for _, k := range keys {
		if src, ok := all[k]; ok {
			src.Key = k
			out[k] = src
		}
	}
	return out
}

func (s *Server) pageData(r *http.Request, p *content.Page) (pageResponse, error) {
	srcs, err := s.cat.Sources(r.Context())
	if err != nil {
		return pageResponse{}, err
	}
	return pageResponse{
		Slug: p.Slug, Title: p.Title, Description: p.Description, LastReviewed: p.LastReviewed,
		HTML: p.HTML, TOC: p.TOC, Sources: pick(srcs, p.Keys), Image: s.pageImage(p),
	}, nil
}

const (
	searchDefaultLimit = 8
	searchMaxLimit     = 25
)

func (s *Server) search(r *http.Request, q string, limit int) (searchResponse, error) {
	resp := searchResponse{Query: q, Results: []searchResult{}}
	if strings.TrimSpace(q) == "" {
		return resp, nil
	}
	hits, err := s.cat.Search(r.Context(), q, limit)
	if err != nil {
		return resp, err
	}
	for _, h := range hits {
		resp.Results = append(resp.Results, searchResult{
			Slug: h.Slug, Name: h.Name, BotanicalName: h.BotanicalName, MatchedName: h.MatchedName,
			MatchKind: h.MatchKind, Score: math.Round(h.Score*100) / 100, URL: "/wood/" + h.Slug,
		})
	}
	return resp, nil
}

// ---- handlers --------------------------------------------------------------

func (s *Server) apiSpeciesList(w http.ResponseWriter, r *http.Request) {
	f, msg := parseSpeciesFilter(r)
	if msg != "" {
		writeError(w, http.StatusBadRequest, "bad_request", msg)
		return
	}
	out, err := s.listSpecies(r, f)
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) apiSpecies(w http.ResponseWriter, r *http.Request) {
	sp, ok, err := s.cat.GetSpecies(r.Context(), r.PathValue("slug"))
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	if !ok {
		writeError(w, http.StatusNotFound, "not_found", "No species with that slug.")
		return
	}
	out, err := s.speciesDetail(r, sp)
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) apiSearch(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	if len(q) > 200 {
		writeError(w, http.StatusBadRequest, "bad_request", "q is too long")
		return
	}
	limit := searchDefaultLimit
	if v := r.URL.Query().Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 {
			writeError(w, http.StatusBadRequest, "bad_request", "limit must be a positive integer")
			return
		}
		limit = min(n, searchMaxLimit)
	}
	out, err := s.search(r, q, limit)
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) apiSources(w http.ResponseWriter, r *http.Request) {
	srcs, err := s.cat.Sources(r.Context())
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, sourcesResponse{Sources: srcs})
}

func (s *Server) apiSource(w http.ResponseWriter, r *http.Request) {
	srcs, err := s.cat.Sources(r.Context())
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	src, ok := srcs[r.PathValue("key")]
	if !ok {
		writeError(w, http.StatusNotFound, "not_found", "No source with that key.")
		return
	}
	writeJSON(w, http.StatusOK, src)
}

func (s *Server) apiPage(w http.ResponseWriter, r *http.Request) {
	p, ok := s.lookupPage(r.PathValue("path"))
	if !ok {
		writeError(w, http.StatusNotFound, "not_found", "No page at that path.")
		return
	}
	out, err := s.pageData(r, p)
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

// lookupPage finds a content page for a URL path that is one of the page
// routes (contract §2). Other paths are not pages even if a file exists.
func (s *Server) lookupPage(p string) (*content.Page, bool) {
	p = strings.Trim(p, "/")
	allowed := false
	for _, prefix := range []string{"protection/", "print/"} {
		if rest, ok := strings.CutPrefix(p, prefix); ok && rest != "" && !strings.Contains(rest, "/") {
			allowed = true
		}
	}
	for _, r := range pageRoutes {
		if "/"+p == r {
			allowed = true
		}
	}
	if !allowed {
		return nil, false
	}
	pg, ok := s.pages.Get(p)
	if ok && pg.Slug != p {
		// Report the URL path as the slug regardless of file layout.
		cp := *pg
		cp.Slug = p
		pg = &cp
	}
	return pg, ok
}

func (s *Server) apiHealth(w http.ResponseWriter, r *http.Request) {
	if err := s.cat.Ping(r.Context()); err != nil {
		s.log.Error("health: ping failed", "err", err)
		writeJSON(w, http.StatusServiceUnavailable, healthResponse{Status: "error"})
		return
	}
	all, err := s.cat.ListSpecies(r.Context())
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, healthResponse{Status: "error"})
		return
	}
	srcs, err := s.cat.Sources(r.Context())
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, healthResponse{Status: "error"})
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, healthResponse{Status: "ok", Species: len(all), Sources: len(srcs)})
}

func (s *Server) apiNotFound(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotFound, "not_found", "No such API endpoint.")
}

// sortedSources returns registry entries ordered by key.
func sortedSources(m map[string]model.Source) []model.Source {
	out := make([]model.Source, 0, len(m))
	for _, v := range m {
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool { return strings.ToLower(out[i].Key) < strings.ToLower(out[j].Key) })
	return out
}
