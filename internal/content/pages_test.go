package content

import (
	"strings"
	"testing"
)

const sample = `---
title: Respirators
slug: protection/respirators
description: How to choose one.
last_reviewed: 2026-11-07
og_image_alt: A respirator.
sources:
  - EXTRA
  - NIOSH-FFR
---

# Respirators

Masks are not respirators.[^NIOSH-FFR] Fit matters.[^OSHA-1910-134] Again.[^NIOSH-FFR]

## Fit testing & seal checks[^OSHA-1910-134]

See the [glossary](/glossary#assigned-protection-factor).

## Fit testing & seal checks
`

func TestRender(t *testing.T) {
	p, err := Render([]byte(sample), "ignored")
	if err != nil {
		t.Fatal(err)
	}
	if p.Slug != "protection/respirators" || p.Key() != "protection-respirators" {
		t.Errorf("slug %q key %q", p.Slug, p.Key())
	}
	if p.LastReviewed != "2026-11-07" || p.ImageAlt != "A respirator." {
		t.Errorf("front matter: %+v", p)
	}
	if strings.Contains(p.HTML, "<h1") {
		t.Error("leading h1 should be dropped (title comes from front matter)")
	}
	if !strings.Contains(p.HTML, `<sup class="cite"><a href="#src-NIOSH-FFR" aria-label="Source 1">1</a></sup>`) ||
		!strings.Contains(p.HTML, `href="#src-OSHA-1910-134" aria-label="Source 2">2</a>`) {
		t.Errorf("citations not numbered by first use:\n%s", p.HTML)
	}
	if strings.Contains(p.HTML, "[^") {
		t.Error("unrendered marker")
	}
	if got := strings.Join(p.Keys, ","); got != "NIOSH-FFR,OSHA-1910-134,EXTRA" || p.Cited != 2 {
		t.Errorf("keys = %s cited = %d", got, p.Cited)
	}
	if len(p.TOC) != 2 || p.TOC[0].ID != "fit-testing-seal-checks" || p.TOC[0].Text != "Fit testing & seal checks" ||
		p.TOC[1].ID != "fit-testing-seal-checks-1" {
		t.Errorf("toc = %+v", p.TOC)
	}
}

func TestHeadingID(t *testing.T) {
	for in, want := range map[string]string{
		"IARC Group 1":              "iarc-group-1",
		"  Inhalable dust  ":        "inhalable-dust",
		"mg/m³ (milligrams)":        "mg-m-milligrams",
		"PAPR — powered respirator": "papr-powered-respirator",
	} {
		if got := HeadingID(in); got != want {
			t.Errorf("HeadingID(%q) = %q, want %q", in, got, want)
		}
	}
}

// Every /glossary#id (and any /<page>#id) link in the real content must hit
// a generated heading id.
func TestRepoContentAnchors(t *testing.T) {
	set, err := Load("../../content/pages")
	if err != nil {
		t.Fatal(err)
	}
	if len(set.All()) == 0 {
		t.Skip("no content pages yet")
	}
	g, ok := set.Get("glossary")
	if !ok {
		t.Fatal("glossary page missing")
	}
	links := 0
	for _, p := range set.All() {
		links += strings.Count(p.HTML, `href="/glossary#`)
	}
	if links == 0 || len(g.IDs) == 0 {
		t.Fatalf("expected glossary links and ids, got %d links, %d ids", links, len(g.IDs))
	}
	for _, msg := range set.BrokenAnchors() {
		t.Error(msg)
	}
	if _, ok := set.Get("print/shop-poster"); !ok {
		t.Error("print/shop-poster not loaded")
	}
	if _, ok := set.Get("protection/respirators"); !ok {
		t.Error("protection/respirators not found from flat file name")
	}
}

func TestBrokenAnchorDetected(t *testing.T) {
	a, _ := Render([]byte("---\ntitle: G\nslug: glossary\n---\n## Real term\n"), "")
	b, _ := Render([]byte("---\ntitle: R\nslug: risks\n---\nSee [x](/glossary#real-term) and [y](/glossary#missing-term).\n"), "")
	set := &Set{pages: map[string]*Page{a.Key(): a, b.Key(): b}}
	got := set.BrokenAnchors()
	if len(got) != 1 || !strings.Contains(got[0], "missing-term") {
		t.Errorf("BrokenAnchors = %v", got)
	}
}
