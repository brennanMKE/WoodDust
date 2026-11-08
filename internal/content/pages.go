// Package content renders the education pages in content/pages/*.md
// (YAML front matter + Markdown) to HTML with goldmark.
//
// Source citations are written as footnote-style markers, [^KEY], where KEY
// is a data/sources.json key. They render as
// <sup class="cite"><a href="#src-KEY">n</a></sup>, numbered in order of
// first use (contract §3, GET /pages/{path}).
package content

import (
	"bytes"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	goldhtml "github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/text"
	"gopkg.in/yaml.v3"
)

// Page is one rendered education page.
type Page struct {
	// Slug is the URL path without the leading slash, e.g.
	// "protection/respirators".
	Slug         string
	Title        string
	Description  string
	LastReviewed string
	ImageAlt     string
	ImagePrompt  string
	HTML         string
	TOC          []TOCEntry
	// IDs holds every heading id on the page (all levels).
	IDs map[string]bool
	// Keys are the cited source keys in order of first use, followed by any
	// front-matter `sources` not cited inline.
	Keys []string
	// Cited is how many of Keys are numbered inline markers.
	Cited int
	File  string
}

// Key is the slug with "/" replaced by "-": the file-name and image-name form.
func (p Page) Key() string { return SlugKey(p.Slug) }

// SlugKey converts "protection/respirators" to "protection-respirators".
func SlugKey(slug string) string { return strings.ReplaceAll(strings.Trim(slug, "/"), "/", "-") }

// TOCEntry is one h2/h3 heading.
type TOCEntry struct {
	ID    string `json:"id"`
	Text  string `json:"text"`
	Level int    `json:"level"`
}

type frontMatter struct {
	Slug         string `yaml:"slug"`
	Title        string `yaml:"title"`
	Description  string `yaml:"description"`
	LastReviewed string `yaml:"last_reviewed"`
	ImageAlt     string `yaml:"image_alt"`
	OGImageAlt   string `yaml:"og_image_alt"`
	ImagePrompt  string `yaml:"image_prompt"`
	// Sources lists further keys the page relies on without an inline
	// marker; they are listed after the cited ones.
	Sources []string `yaml:"sources"`
}

// Set is the loaded pages, keyed by SlugKey.
type Set struct {
	pages map[string]*Page
}

// Get looks a page up by URL path ("protection/respirators") or key.
func (s *Set) Get(path string) (*Page, bool) {
	if s == nil {
		return nil, false
	}
	p, ok := s.pages[SlugKey(path)]
	return p, ok
}

// All returns the pages sorted by slug.
func (s *Set) All() []*Page {
	if s == nil {
		return nil
	}
	out := make([]*Page, 0, len(s.pages))
	for _, p := range s.pages {
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Slug < out[j].Slug })
	return out
}

// Load renders every *.md under dir (recursively). A missing dir is not an
// error: it yields an empty set, so the site runs before content exists.
//
// The slug is the front matter `slug` if present; otherwise it is derived
// from the path: "protection/respirators.md" -> "protection/respirators".
// A flat "protection-respirators.md" without a `slug` is found by the
// dashed key, so /protection/respirators resolves either way.
func Load(dir string) (*Set, error) {
	set := &Set{pages: map[string]*Page{}}
	info, err := os.Stat(dir)
	if os.IsNotExist(err) {
		return set, nil
	}
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%s is not a directory", dir)
	}
	err = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(d.Name(), ".md") || strings.HasPrefix(d.Name(), "_") {
			return nil
		}
		if strings.EqualFold(d.Name(), "README.md") {
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(dir, path)
		p, err := Render(b, strings.TrimSuffix(filepath.ToSlash(rel), ".md"))
		if err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		p.File = path
		if prev, dup := set.pages[p.Key()]; dup {
			return fmt.Errorf("%s: page %q already defined by %s", path, p.Slug, prev.File)
		}
		set.pages[p.Key()] = p
		return nil
	})
	if err != nil {
		return nil, err
	}
	return set, nil
}

// citeRE matches [^KEY] but not a footnote definition "[^KEY]:".
var citeRE = regexp.MustCompile(`\[\^([A-Za-z0-9][A-Za-z0-9-]*)\](:?)`)

var md = goldmark.New(
	goldmark.WithExtensions(extension.GFM, extension.Typographer),
	goldmark.WithParserOptions(parser.WithAutoHeadingID()),
	// Content is first-party, reviewed Markdown from git; raw HTML is needed
	// for the citation markers we inject.
	goldmark.WithRendererOptions(goldhtml.WithUnsafe()),
)

// Render parses one page. defaultSlug is used when the front matter has none.
func Render(src []byte, defaultSlug string) (*Page, error) {
	fm, body, err := splitFrontMatter(src)
	if err != nil {
		return nil, err
	}
	p := &Page{
		Slug:         strings.Trim(defaultSlug, "/"),
		Title:        fm.Title,
		Description:  fm.Description,
		LastReviewed: fm.LastReviewed,
		ImageAlt:     firstNonEmpty(fm.ImageAlt, fm.OGImageAlt),
		ImagePrompt:  fm.ImagePrompt,
	}
	if fm.Slug != "" {
		p.Slug = strings.Trim(fm.Slug, "/")
	}
	if p.Title == "" {
		return nil, fmt.Errorf("front matter: title is required")
	}

	// Number citations in order of first use.
	numbers := map[string]int{}
	body = citeRE.ReplaceAllFunc(body, func(m []byte) []byte {
		sub := citeRE.FindSubmatch(m)
		if len(sub[2]) > 0 { // a definition line; leave it alone
			return m
		}
		key := string(sub[1])
		n, ok := numbers[key]
		if !ok {
			n = len(numbers) + 1
			numbers[key] = n
			p.Keys = append(p.Keys, key)
		}
		k := html.EscapeString(key)
		return []byte(`<sup class="cite"><a href="#src-` + k + `" aria-label="Source ` + strconv.Itoa(n) + `">` + strconv.Itoa(n) + `</a></sup>`)
	})

	p.Cited = len(p.Keys)
	for _, k := range fm.Sources {
		if _, ok := numbers[k]; !ok {
			numbers[k] = 0
			p.Keys = append(p.Keys, k)
		}
	}

	ctx := parser.NewContext(parser.WithIDs(newHeadingIDs()))
	doc := md.Parser().Parse(text.NewReader(body), parser.WithContext(ctx))
	dropLeadingH1(doc)
	p.TOC = collectTOC(doc, body)
	p.IDs = map[string]bool{}
	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if h, ok := n.(*ast.Heading); ok && entering {
			if id, ok := h.AttributeString("id"); ok {
				if b, ok := id.([]byte); ok {
					p.IDs[string(b)] = true
				}
			}
		}
		return ast.WalkContinue, nil
	})
	var buf bytes.Buffer
	if err := md.Renderer().Render(&buf, body, doc); err != nil {
		return nil, err
	}
	p.HTML = buf.String()
	if p.TOC == nil {
		p.TOC = []TOCEntry{}
	}
	return p, nil
}

func firstNonEmpty(v ...string) string {
	for _, s := range v {
		if s != "" {
			return s
		}
	}
	return ""
}

// dropLeadingH1 removes a level-1 heading that opens the document: the page
// title is rendered from front matter, so keeping it would give two <h1>s.
func dropLeadingH1(doc ast.Node) {
	if h, ok := doc.FirstChild().(*ast.Heading); ok && h.Level == 1 {
		doc.RemoveChild(doc, h)
	}
}

var (
	citeHTMLRE = regexp.MustCompile(`<sup class="cite">.*?</sup>`)
	tagRE      = regexp.MustCompile(`<[^>]*>`)
	nonAlnumRE = regexp.MustCompile(`[^a-z0-9]+`)
)

// HeadingID is the id rule for headings: lowercase, every run of
// non-alphanumerics becomes "-", leading/trailing dashes trimmed. The
// glossary relies on it for /glossary#<id> links.
func HeadingID(s string) string {
	s = citeHTMLRE.ReplaceAllString(s, "")
	s = tagRE.ReplaceAllString(s, "")
	s = html.UnescapeString(s)
	return strings.Trim(nonAlnumRE.ReplaceAllString(strings.ToLower(s), "-"), "-")
}

// headingIDs implements parser.IDs with HeadingID, de-duplicating with
// "-1", "-2", ... within one page.
type headingIDs struct{ used map[string]bool }

func newHeadingIDs() *headingIDs { return &headingIDs{used: map[string]bool{}} }

func (h *headingIDs) Generate(value []byte, _ ast.NodeKind) []byte {
	base := HeadingID(string(value))
	if base == "" {
		base = "section"
	}
	id := base
	for i := 1; h.used[id]; i++ {
		id = base + "-" + strconv.Itoa(i)
	}
	h.used[id] = true
	return []byte(id)
}

func (h *headingIDs) Put(value []byte) { h.used[string(value)] = true }

func splitFrontMatter(src []byte) (frontMatter, []byte, error) {
	var fm frontMatter
	src = bytes.TrimPrefix(src, []byte("\ufeff"))
	s := strings.ReplaceAll(string(src), "\r\n", "\n")
	if !strings.HasPrefix(s, "---\n") {
		return fm, []byte(s), nil
	}
	rest := s[4:]
	end := strings.Index(rest, "\n---")
	if end < 0 {
		return fm, nil, fmt.Errorf("front matter: missing closing ---")
	}
	if err := yaml.Unmarshal([]byte(rest[:end]), &fm); err != nil {
		return fm, nil, fmt.Errorf("front matter: %w", err)
	}
	body := rest[end+4:]
	body = strings.TrimPrefix(body, "\n")
	return fm, []byte(body), nil
}

func collectTOC(doc ast.Node, src []byte) []TOCEntry {
	var out []TOCEntry
	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		h, ok := n.(*ast.Heading)
		if !ok || h.Level < 2 || h.Level > 3 {
			return ast.WalkContinue, nil
		}
		id, _ := h.AttributeString("id")
		idb, _ := id.([]byte)
		out = append(out, TOCEntry{ID: string(idb), Text: strings.TrimSpace(nodeText(h, src)), Level: h.Level})
		return ast.WalkSkipChildren, nil
	})
	return out
}

// nodeText concatenates the text segments under n, skipping raw HTML (the
// citation markers).
func nodeText(n ast.Node, src []byte) string {
	var b strings.Builder
	inCite := false
	_ = ast.Walk(n, func(c ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch t := c.(type) {
		case *ast.RawHTML:
			var raw strings.Builder
			for i := 0; i < t.Segments.Len(); i++ {
				seg := t.Segments.At(i)
				raw.Write(seg.Value(src))
			}
			if strings.Contains(raw.String(), `class="cite"`) {
				inCite = true
			} else if strings.Contains(raw.String(), "</sup>") {
				inCite = false
			}
			return ast.WalkSkipChildren, nil
		case *ast.Text:
			if inCite {
				return ast.WalkContinue, nil
			}
			b.Write(t.Segment.Value(src))
			if t.SoftLineBreak() {
				b.WriteByte(' ')
			}
		case *ast.String:
			b.Write(t.Value)
		}
		return ast.WalkContinue, nil
	})
	return b.String()
}

var anchorRE = regexp.MustCompile(`href="/([a-z0-9/-]*)#([^"]+)"`)

// BrokenAnchors lists same-site links of the form /<page>#<id> (e.g.
// /glossary#patch-test) whose page exists in the set but has no heading with
// that id. Links to pages that are not in the set are not checked.
func (s *Set) BrokenAnchors() []string {
	var out []string
	for _, p := range s.All() {
		for _, m := range anchorRE.FindAllStringSubmatch(p.HTML, -1) {
			target, ok := s.Get(m[1])
			if !ok {
				continue
			}
			if !target.IDs[html.UnescapeString(m[2])] {
				out = append(out, fmt.Sprintf("%s links to /%s#%s, which has no such heading", p.File, m[1], m[2]))
			}
		}
	}
	return out
}

// CitedKeys returns every source key used by any page, for integrity checks.
func (s *Set) CitedKeys() map[string][]string {
	out := map[string][]string{}
	for _, p := range s.All() {
		out[p.File] = p.Keys
	}
	return out
}
