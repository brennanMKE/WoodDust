package model

import (
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// foldExtra covers letters that Postgres unaccent rewrites but that have no
// combining-mark decomposition.
var foldExtra = map[rune]string{
	'ø': "o", 'æ': "ae", 'œ': "oe", 'ß': "ss", 'ð': "d", 'þ': "th", 'ł': "l", 'đ': "d", 'ı': "i",
}

// Normalize lowercases, strips diacritics, and collapses every run of
// characters outside [a-z0-9] to one space. It must agree with the SQL
// function wd_normalize (migrations/000001) — a DB-backed test checks parity
// over every name in data/.
//
// "Western Red-Cedar" -> "western red cedar"; "Ipê" -> "ipe";
// "thuja-plicata" -> "thuja plicata".
func Normalize(v string) string {
	var b strings.Builder
	space := false
	for _, r := range norm.NFKD.String(strings.ToLower(v)) {
		if unicode.Is(unicode.Mn, r) {
			continue
		}
		var piece string
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			piece = string(r)
		default:
			piece = foldExtra[r]
		}
		if piece == "" {
			space = true
			continue
		}
		if space && b.Len() > 0 {
			b.WriteByte(' ')
		}
		space = false
		b.WriteString(piece)
	}
	return b.String()
}

// trigrams mirrors pg_trgm's show_trgm for already-normalized text: each
// word is padded with two leading spaces and one trailing space.
func trigrams(v string) map[string]struct{} {
	out := map[string]struct{}{}
	for _, w := range strings.Fields(v) {
		p := "  " + w + " "
		rs := []rune(p)
		for i := 0; i+3 <= len(rs); i++ {
			out[string(rs[i:i+3])] = struct{}{}
		}
	}
	return out
}

// Similarity mirrors pg_trgm similarity() on normalized strings.
func Similarity(a, b string) float64 {
	ta, tb := trigrams(a), trigrams(b)
	if len(ta) == 0 || len(tb) == 0 {
		return 0
	}
	common := 0
	for t := range ta {
		if _, ok := tb[t]; ok {
			common++
		}
	}
	return float64(common) / float64(len(ta)+len(tb)-common)
}

// SimilarityThreshold matches pg_trgm.similarity_threshold's default.
const SimilarityThreshold = 0.3

// MatchScore ranks one normalized name against a normalized query, using the
// same formula as the SQL search in internal/store. Prefix matches always
// outrank trigram matches. ok is false when the name does not match.
func MatchScore(name, q string) (score float64, ok bool) {
	if q == "" || name == "" {
		return 0, false
	}
	ratio := float64(len(q)) / float64(len(name))
	switch {
	case name == q:
		return 1, true
	case strings.HasPrefix(name, q):
		return 0.9 + 0.09*ratio, true
	case strings.Contains(name, " "+q):
		return 0.8 + 0.09*ratio, true
	}
	if s := Similarity(name, q); s >= SimilarityThreshold {
		return s * 0.79, true
	}
	return 0, false
}
