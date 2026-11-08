package site

import (
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"path"
	"strings"
)

// Image cache busting.
//
// Apache serves /images/ with a long cache. A browser that once fetched an
// image before it existed (or before it was redrawn) would keep the stale
// response, so every image URL the site hands out carries ?v=<content hash>.
// The hashes are computed once at startup from IMAGES_DIR; a release that
// changes a file changes its URL.

// hashImages returns rel path -> first 10 hex chars of the file's SHA-256 for
// every published file under fsys (not prompts/ or files starting with "_").
func hashImages(fsys fs.FS) map[string]string {
	out := map[string]string{}
	if fsys == nil {
		return out
	}
	_ = fs.WalkDir(fsys, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if p == "prompts" {
				return fs.SkipDir
			}
			return nil
		}
		if strings.HasPrefix(path.Base(p), "_") {
			return nil
		}
		b, err := fs.ReadFile(fsys, p)
		if err != nil {
			return nil
		}
		sum := sha256.Sum256(b)
		out[p] = hex.EncodeToString(sum[:])[:10]
		return nil
	})
	return out
}

// imageURL returns /images/<rel> with its version query, or without one when
// the file is unknown.
func (s *Server) imageURL(rel string) string {
	if v, ok := s.imageVersions[rel]; ok {
		return "/images/" + rel + "?v=" + v
	}
	return "/images/" + rel
}

// spaAssetVersions is the subset the SPA references by fixed path
// (illustrations, mascots and the default share image).
func (s *Server) spaAssetVersions() map[string]string {
	out := map[string]string{}
	for k, v := range s.imageVersions {
		if strings.HasPrefix(k, "art/") || strings.HasPrefix(k, "mascots/") || k == "og-default.jpg" ||
			k == "pages/protection-respirators.jpg" || k == "pages/methodology.jpg" {
			out[k] = v
		}
	}
	return out
}
