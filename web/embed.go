// Package web embeds the built Svelte SPA (web/dist) into the Go binary.
//
// //go:embed cannot reach outside its own directory, so the embed lives here
// rather than in cmd/. dist/ is build output and git-ignored except for
// dist/.gitkeep, which is tracked so `go build ./...` compiles from a clean
// checkout before `npm run build` has ever run (an embed pattern must match at
// least one file). Vite's emptyOutDir deletes .gitkeep on build; `make build`
// and scripts/check.sh put it back. When dist/index.html is absent the server
// falls back to a minimal built-in shell (internal/site/shell.go), so
// server-rendered pages still work without the SPA.
package web

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var distFS embed.FS

// DistFS returns the built SPA rooted at dist/ (index.html, assets/...).
func DistFS() fs.FS {
	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		panic("web: dist missing from embedded FS: " + err.Error())
	}
	return sub
}
