package site

import (
	"bytes"
	"embed"
	"encoding/json"
	"html/template"
	"io/fs"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/brennanMKE/WoodDust/internal/content"
	"github.com/brennanMKE/WoodDust/internal/model"
)

//go:embed templates/*.html
var templateFS embed.FS

var tmpl = template.Must(template.New("").Funcs(template.FuncMap{
	"join":          strings.Join,
	"evidenceLabel": func(l int) string { return model.EvidenceLabels[l].Label },
	"num":           func(f float64) string { return strconv.FormatFloat(f, 'f', -1, 64) },
}).ParseFS(templateFS, "templates/*.html"))

// ---- shell -------------------------------------------------------------------

const (
	markerHead = "<!--ssr-head-->"
	markerBody = "<!--ssr-body-->"
	markerData = "<!--ssr-data-->"
)

// fallbackShell is used when web/dist/index.html is absent (no SPA build).
const fallbackShell = `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<!--ssr-head-->
</head>
<body>
<div id="app"><!--ssr-body--></div>
<!--ssr-data-->
</body>
</html>
`

// shell is index.html split at the three markers.
type shell struct {
	parts [4]string
}

func splitShell(html string) (shell, bool) {
	var sh shell
	rest := html
	for i, m := range []string{markerHead, markerBody, markerData} {
		before, after, ok := strings.Cut(rest, m)
		if !ok {
			return sh, false
		}
		sh.parts[i] = before
		rest = after
	}
	sh.parts[3] = rest
	return sh, true
}

// loadShell reads dist/index.html; usingFallback is true when it cannot.
func loadShell(dist fs.FS) (sh shell, usingFallback bool) {
	if dist != nil {
		if b, err := fs.ReadFile(dist, "index.html"); err == nil {
			if sh, ok := splitShell(string(b)); ok {
				return sh, false
			}
		}
	}
	sh, _ = splitShell(fallbackShell)
	return sh, true
}

// ---- head ------------------------------------------------------------------

type ogImage struct {
	URL    string // absolute
	Alt    string
	Width  int
	Height int
}

type headData struct {
	Title       string
	OGTitle     string
	Description string
	Canonical   string // absolute; empty for 404
	SiteName    string
	OGType      string
	Image       ogImage
	NoIndex     bool
	JSONLD      []template.JS
	Favicon     string
}

type crumb struct {
	Name string
	URL  string // site path; empty for the current page
}

// view is everything one page render needs.
type view struct {
	Status      int
	Route       string // contract route id
	Head        headData
	Template    string // body_<route>
	Breadcrumbs []crumb
	Body        any
	Data        any // initial data (the API response for the route)
}

type siteInfo struct {
	Name    string `json:"name"`
	BaseURL string `json:"base_url"`
}

type initialData struct {
	Route string   `json:"route"`
	Data  any      `json:"data"`
	Site  siteInfo `json:"site"`
}

// truncate shortens to at most n runes at a word boundary, adding "…".
func truncate(s string, n int) string {
	s = strings.Join(strings.Fields(s), " ")
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	r := []rune(s)[:n-1]
	cut := string(r)
	if i := strings.LastIndex(cut, " "); i > n/2 {
		cut = cut[:i]
	}
	return strings.TrimRight(cut, " ,;:.") + "…"
}

func jsonLD(v any) template.JS {
	b, _ := json.Marshal(v) // escapes <, >, & so </script> cannot appear
	return template.JS(b)
}

// ogImageFor returns the share image: the page/species image when it exists,
// else images/og-default.jpg when it exists, else the placeholder.
func (s *Server) ogImageFor(img imageView) ogImage {
	url := img.URL
	if url == placeholderPath && s.imageExists("og-default.jpg") {
		url = "/images/og-default.jpg"
	}
	alt := img.Alt
	if alt == "" {
		alt = s.siteName
	}
	return ogImage{URL: s.absURL(url), Alt: alt, Width: imageWidth, Height: imageHeight}
}

func (s *Server) defaultImage() imageView {
	return imageView{URL: placeholderPath, Alt: s.siteName, Width: imageWidth, Height: imageHeight}
}

func (s *Server) breadcrumbLD(crumbs []crumb, canonical string) template.JS {
	items := make([]map[string]any, 0, len(crumbs))
	for i, c := range crumbs {
		item := s.absURL(c.URL)
		if c.URL == "" {
			item = canonical
		}
		items = append(items, map[string]any{"@type": "ListItem", "position": i + 1, "name": c.Name, "item": item})
	}
	return jsonLD(map[string]any{"@context": "https://schema.org", "@type": "BreadcrumbList", "itemListElement": items})
}

func (s *Server) articleLD(headline, desc, canonical, modified string, img ogImage, about any, citations []model.Source) template.JS {
	a := map[string]any{
		"@context":            "https://schema.org",
		"@type":               "Article",
		"headline":            headline,
		"description":         desc,
		"url":                 canonical,
		"mainEntityOfPage":    canonical,
		"image":               img.URL,
		"inLanguage":          "en",
		"isAccessibleForFree": true,
		"publisher":           map[string]any{"@type": "Organization", "name": s.siteName, "url": s.absURL("/")},
	}
	if modified != "" {
		a["dateModified"] = modified
	}
	if about != nil {
		a["about"] = about
	}
	if len(citations) > 0 {
		var cs []map[string]any
		for _, c := range citations {
			m := map[string]any{"@type": "CreativeWork", "name": c.Citation}
			if l := c.Link(); l != "" {
				m["url"] = l
			}
			cs = append(cs, m)
		}
		a["citation"] = cs
	}
	return jsonLD(a)
}

// ---- rendering -------------------------------------------------------------

func (s *Server) render(w http.ResponseWriter, r *http.Request, v view) {
	v.Head.SiteName = s.siteName
	if v.Head.OGTitle == "" {
		v.Head.OGTitle = v.Head.Title
	}
	if v.Head.OGType == "" {
		v.Head.OGType = "website"
	}
	if v.Head.Image.URL == "" {
		v.Head.Image = s.ogImageFor(s.defaultImage())
	}
	if s.imageExists("art/favicon.png") {
		v.Head.Favicon = "/images/art/favicon.png"
	}

	var head, body bytes.Buffer
	if err := tmpl.ExecuteTemplate(&head, "head", v.Head); err != nil {
		s.log.Error("render head", "err", err)
	}
	if v.Head.Favicon != "" {
		head.WriteString(`<link rel="icon" type="image/png" href="` + v.Head.Favicon + `">` + "\n")
	}
	bodyData := struct {
		Site        siteInfo
		Breadcrumbs []crumb
		Body        any
	}{siteInfo{s.siteName, s.baseURL}, v.Breadcrumbs, v.Body}
	if err := tmpl.ExecuteTemplate(&body, v.Template, bodyData); err != nil {
		s.log.Error("render body", "template", v.Template, "err", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	if v.Data == nil {
		v.Data = struct{}{}
	}
	init, err := json.Marshal(initialData{Route: v.Route, Data: v.Data, Site: siteInfo{s.siteName, s.baseURL}})
	if err != nil {
		s.log.Error("render data", "err", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	var out bytes.Buffer
	out.Grow(len(s.shell.parts[0]) + head.Len() + body.Len() + len(init) + 2048)
	out.WriteString(s.shell.parts[0])
	out.Write(head.Bytes())
	out.WriteString(s.shell.parts[1])
	out.Write(body.Bytes())
	out.WriteString(s.shell.parts[2])
	out.WriteString(`<script type="application/json" id="initial">`)
	out.Write(init)
	out.WriteString(`</script>`)
	out.WriteString(s.shell.parts[3])

	h := w.Header()
	h.Set("Content-Type", "text/html; charset=utf-8")
	h.Set("Cache-Control", "no-cache")
	if v.Head.NoIndex {
		h.Set("X-Robots-Tag", "noindex")
	}
	status := v.Status
	if status == 0 {
		status = http.StatusOK
	}
	w.WriteHeader(status)
	if r.Method != http.MethodHead {
		_, _ = w.Write(out.Bytes())
	}
}

func (s *Server) renderError(w http.ResponseWriter, r *http.Request, err error) {
	s.log.Error("page error", "path", r.URL.Path, "err", err)
	http.Error(w, "Something went wrong. Please try again.", http.StatusInternalServerError)
}

// ---- body view models --------------------------------------------------------

type speciesRow struct {
	URL, Name, BotanicalName, WoodType string
	HazardNames                        []string
	MaxLabel                           string
}

func rowFor(sp model.Species) speciesRow {
	row := speciesRow{URL: "/wood/" + sp.Slug, Name: sp.DisplayName(), BotanicalName: sp.BotanicalName, WoodType: sp.WoodType}
	for _, t := range sp.HazardTypes() {
		row.HazardNames = append(row.HazardNames, model.HazardTypeName(t))
	}
	if l := sp.MaxEvidenceLevel(); l != nil {
		row.MaxLabel = model.EvidenceLabels[*l].Label
	}
	return row
}

type refView struct {
	Key string
	N   int
}

type hazardView struct {
	TypeName string
	Level    int
	Label    string
	Short    string
	Summary  string
	Effects  []string
	Routes   []string
	Parts    []string
	Refs     []refView
}

type sourceView struct {
	Key, Citation, Link, LinkText, Tier string
	Verified, Missing                   bool
}

func sourceViewFor(key string, src model.Source, ok bool) sourceView {
	v := sourceView{Key: key, Missing: !ok}
	if !ok {
		v.Citation = key
		return v
	}
	v.Citation, v.Tier, v.Verified, v.Link = src.Citation, src.Tier, src.Verified, src.Link()
	switch {
	case src.DOI != "" && src.URL == "":
		v.LinkText = "doi:" + src.DOI
	case src.PMID != "" && src.URL == "" && src.DOI == "":
		v.LinkText = "PubMed " + src.PMID
	default:
		v.LinkText = "Link"
	}
	return v
}

type regRow struct {
	Body  string
	Value float64
	Note  string
}

type speciesBody struct {
	Name        string
	Species     model.Species
	AlsoKnownAs []string
	Image       *imageView
	Hazards     []hazardView
	Regulatory  []regRow
	Sources     []sourceView
}

type pageBody struct {
	Title, LastReviewed string
	HTML                template.HTML
	TOC                 []content.TOCEntry
	Sources             []sourceView
}

// ---- page handlers -----------------------------------------------------------

type homeBody struct {
	Query    string
	Results  []searchResult
	Featured []speciesRow
	Total    int
}

func (s *Server) pageHome(w http.ResponseWriter, r *http.Request) {
	all, err := s.cat.ListSpecies(r.Context())
	if err != nil {
		s.renderError(w, r, err)
		return
	}
	data := speciesList{Species: []speciesSummary{}}
	body := homeBody{Total: len(all)}
	for _, sp := range all {
		if sp.FeaturedRank != nil {
			data.Species = append(data.Species, s.summary(sp))
			body.Featured = append(body.Featured, rowFor(sp))
		}
	}
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if len(q) > 200 {
		q = q[:200]
	}
	body.Query = q
	if q != "" {
		res, err := s.search(r, q, searchDefaultLimit)
		if err != nil {
			s.renderError(w, r, err)
			return
		}
		body.Results = res.Results
	}
	canonical := s.absURL("/")
	desc := truncate("Which wood dusts cause asthma, skin allergies or cancer? Plain-language, sourced hazard information for "+
		strconv.Itoa(len(all))+" woods, and how to protect yourself in the shop.", 160)
	head := headData{
		Title:       s.siteName + ": wood dust hazards by species",
		Description: desc,
		Canonical:   canonical,
		NoIndex:     r.URL.Query().Has("q"),
		JSONLD: []template.JS{jsonLD(map[string]any{
			"@context": "https://schema.org",
			"@type":    "WebSite",
			"name":     s.siteName,
			"url":      canonical,
			"potentialAction": map[string]any{
				"@type":       "SearchAction",
				"target":      map[string]any{"@type": "EntryPoint", "urlTemplate": s.absURL("/?q={search_term_string}")},
				"query-input": "required name=search_term_string",
			},
		})},
	}
	if q != "" {
		head.Title = "Search: " + q + " | " + s.siteName
	}
	s.render(w, r, view{Route: "home", Head: head, Template: "body_home", Body: body, Data: data})
}

func (s *Server) pageSpecies(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	sp, ok, err := s.cat.GetSpecies(r.Context(), slug)
	if err != nil {
		s.renderError(w, r, err)
		return
	}
	if !ok {
		slugs, err := s.cat.ResolveName(r.Context(), slug)
		if err != nil {
			s.renderError(w, r, err)
			return
		}
		if len(slugs) == 1 {
			http.Redirect(w, r, "/wood/"+slugs[0], http.StatusMovedPermanently)
			return
		}
		s.renderNotFound(w, r, slug, slugs)
		return
	}
	detail, err := s.speciesDetail(r, sp)
	if err != nil {
		s.renderError(w, r, err)
		return
	}

	keys := sp.CitedKeys()
	num := map[string]int{}
	var srcViews []sourceView
	var citations []model.Source
	for i, k := range keys {
		num[k] = i + 1
		src, ok := detail.Sources[k]
		srcViews = append(srcViews, sourceViewFor(k, src, ok))
		if ok {
			citations = append(citations, src)
		}
	}
	name := sp.DisplayName()
	body := speciesBody{Name: name, Species: sp.Public(), Sources: srcViews}
	if len(sp.CommonNames) > 1 {
		body.AlsoKnownAs = sp.CommonNames[1:]
	}
	if detail.Image.URL != placeholderPath {
		img := detail.Image
		body.Image = &img
	}
	var hazardNames []string
	for _, h := range sp.Hazards {
		hv := hazardView{
			TypeName: model.HazardTypeName(h.Type), Level: h.EvidenceLevel,
			Label: model.EvidenceLabels[h.EvidenceLevel].Label, Short: model.EvidenceLabels[h.EvidenceLevel].Short,
			Summary: h.Summary, Effects: h.Effects, Routes: h.Route, Parts: h.SourcePart,
		}
		seen := map[string]bool{}
		for _, k := range h.Sources {
			if !seen[k] {
				seen[k] = true
				hv.Refs = append(hv.Refs, refView{Key: k, N: num[k]})
			}
		}
		body.Hazards = append(body.Hazards, hv)
		if h.EvidenceLevel > 0 {
			hazardNames = appendUniq(hazardNames, strings.ToLower(model.HazardTypeName(h.Type)))
		}
	}
	reg := detail.Regulatory
	addReg := func(label string, v *float64, note string) {
		if v != nil {
			body.Regulatory = append(body.Regulatory, regRow{Body: label, Value: *v, Note: note})
		}
	}
	if sp.WoodType == "hardwood" {
		addReg("EU binding limit (hardwood)", reg.EUHardwoodMgM3, "Applies to all hardwood dust")
		addReg("UK WEL (hardwood)", reg.UKHardwoodMgM3, "Applies to all hardwood dust")
	} else {
		addReg("UK WEL (softwood)", reg.UKSoftwoodMgM3, "Applies to all softwood dust")
	}
	addReg("NIOSH REL", reg.NIOSHRELMgM3, "Recommended limit for all wood dust")

	canonical := s.absURL("/wood/" + sp.Slug)
	descLead := name + " (" + sp.BotanicalName + ") wood dust"
	var desc string
	if len(hazardNames) > 0 {
		desc = descLead + ": " + strings.Join(hazardNames, ", ") + ". Evidence levels, exposure limits and sources."
	} else {
		desc = descLead + " hazards, evidence levels, exposure limits and sources. All wood dust can cause cancer."
	}
	desc = truncate(desc, 160)
	img := s.ogImageFor(detail.Image)
	crumbs := []crumb{{"Home", "/"}, {"All woods", "/browse"}, {name, ""}}
	about := map[string]any{"@type": "Thing", "name": name, "alternateName": append([]string{sp.BotanicalName}, body.AlsoKnownAs...)}
	head := headData{
		Title:       name + " wood dust hazards | " + s.siteName,
		OGTitle:     name + " wood dust hazards",
		Description: desc,
		Canonical:   canonical,
		OGType:      "article",
		Image:       img,
		JSONLD: []template.JS{
			s.articleLD(name+" wood dust hazards", desc, canonical, sp.LastReviewed, img, about, citations),
			s.breadcrumbLD(crumbs, canonical),
		},
	}
	s.render(w, r, view{Route: "species", Head: head, Template: "body_species", Breadcrumbs: crumbs, Body: body, Data: detail})
}

func appendUniq(list []string, v string) []string {
	for _, x := range list {
		if x == v {
			return list
		}
	}
	return append(list, v)
}

func (s *Server) pageBrowse(w http.ResponseWriter, r *http.Request) {
	data, err := s.listSpecies(r, speciesFilter{})
	if err != nil {
		s.renderError(w, r, err)
		return
	}
	all, err := s.cat.ListSpecies(r.Context())
	if err != nil {
		s.renderError(w, r, err)
		return
	}
	var rows []speciesRow
	for _, sp := range all {
		rows = append(rows, rowFor(sp))
	}
	canonical := s.absURL("/browse")
	crumbs := []crumb{{"Home", "/"}, {"All woods", ""}}
	head := headData{
		Title:       "All woods: dust hazards by species | " + s.siteName,
		Description: truncate("Browse "+strconv.Itoa(len(rows))+" woods and the health hazards reported for each: asthma, skin allergy, irritation and cancer, with evidence levels and sources.", 160),
		Canonical:   canonical,
		JSONLD:      []template.JS{s.breadcrumbLD(crumbs, canonical)},
	}
	s.render(w, r, view{Route: "browse", Head: head, Template: "body_browse", Breadcrumbs: crumbs,
		Body: struct{ Rows []speciesRow }{rows}, Data: data})
}

func (s *Server) pageSources(w http.ResponseWriter, r *http.Request) {
	srcs, err := s.cat.Sources(r.Context())
	if err != nil {
		s.renderError(w, r, err)
		return
	}
	var views []sourceView
	for _, src := range sortedSources(srcs) {
		views = append(views, sourceViewFor(src.Key, src, true))
	}
	canonical := s.absURL("/sources")
	crumbs := []crumb{{"Home", "/"}, {"Sources", ""}}
	// /sources shares the methodology page's art.
	img := imageView{URL: placeholderPath, Alt: "Sources and methodology", Width: imageWidth, Height: imageHeight}
	if s.imageExists("pages/methodology.jpg") {
		img.URL = "/images/pages/methodology.jpg"
	}
	if mp, ok := s.pages.Get("methodology"); ok && mp.ImageAlt != "" {
		img.Alt = mp.ImageAlt
	}
	head := headData{
		Title:       "Sources | " + s.siteName,
		Description: truncate("The "+strconv.Itoa(len(views))+" studies, agency documents and references behind every hazard on "+s.siteName+".", 160),
		Canonical:   canonical,
		Image:       s.ogImageFor(img),
		JSONLD:      []template.JS{s.breadcrumbLD(crumbs, canonical)},
	}
	s.render(w, r, view{Route: "sources", Head: head, Template: "body_sources", Breadcrumbs: crumbs,
		Body: struct{ Sources []sourceView }{views}, Data: sourcesResponse{Sources: srcs}})
}

func (s *Server) pageContent(w http.ResponseWriter, r *http.Request) {
	p, ok := s.lookupPage(r.URL.Path)
	if !ok {
		s.renderNotFound(w, r, "", nil)
		return
	}
	data, err := s.pageData(r, p)
	if err != nil {
		s.renderError(w, r, err)
		return
	}
	var srcViews []sourceView
	var citations []model.Source
	for _, k := range p.Keys {
		src, ok := data.Sources[k]
		srcViews = append(srcViews, sourceViewFor(k, src, ok))
		if ok {
			citations = append(citations, src)
		}
	}
	canonical := s.absURL("/" + p.Slug)
	crumbs := []crumb{{"Home", "/"}}
	if strings.HasPrefix(p.Slug, "protection/") {
		parent := "Protection"
		if pp, ok := s.pages.Get("protection"); ok {
			parent = pp.Title
		}
		crumbs = append(crumbs, crumb{parent, "/protection"})
	}
	crumbs = append(crumbs, crumb{p.Title, ""})
	desc := p.Description
	if desc == "" {
		desc = p.Title
	}
	desc = truncate(desc, 160)
	img := s.ogImageFor(data.Image)
	head := headData{
		Title:       p.Title + " | " + s.siteName,
		OGTitle:     p.Title,
		Description: desc,
		Canonical:   canonical,
		OGType:      "article",
		Image:       img,
		JSONLD: []template.JS{
			s.articleLD(p.Title, desc, canonical, p.LastReviewed, img, nil, citations),
			s.breadcrumbLD(crumbs, canonical),
		},
	}
	body := pageBody{Title: p.Title, LastReviewed: p.LastReviewed, HTML: template.HTML(p.HTML), TOC: p.TOC, Sources: srcViews}
	s.render(w, r, view{Route: "page", Head: head, Template: "body_page", Breadcrumbs: crumbs, Body: body, Data: data})
}

type notFoundBody struct {
	Name       string
	Candidates []speciesRow
}

// renderNotFound writes the 404 page. When name is ambiguous across several
// species (candidates), it links to each.
func (s *Server) renderNotFound(w http.ResponseWriter, r *http.Request, name string, candidates []string) {
	body := notFoundBody{Name: name}
	for _, slug := range candidates {
		if sp, ok, err := s.cat.GetSpecies(r.Context(), slug); err == nil && ok {
			body.Candidates = append(body.Candidates, rowFor(sp))
		}
	}
	head := headData{
		Title:       "Page not found | " + s.siteName,
		Description: "We could not find that page.",
		NoIndex:     true,
	}
	s.render(w, r, view{Status: http.StatusNotFound, Route: "not_found", Head: head, Template: "body_not_found", Body: body})
}

// pageSubmit is the contributions placeholder (contract §2, route "submit").
func (s *Server) pageSubmit(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	if slug != "" {
		if _, ok, err := s.cat.GetSpecies(r.Context(), slug); err != nil || !ok {
			slug = ""
		}
	}
	head := headData{
		Title:       "Suggest a wood or a correction | " + s.siteName,
		Description: "Suggest a missing wood, a correction, or a better source for " + s.siteName + ".",
		Canonical:   s.absURL("/submit"),
		NoIndex:     true,
	}
	s.render(w, r, view{Route: "submit", Head: head, Template: "body_submit", Body: struct{ Slug string }{slug}})
}
