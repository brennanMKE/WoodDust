// Package posterfiles embeds the shop-poster sources (docs/posters.md): the
// hand-made SVG templates in src/ and the stock QR codes in qr/, which
// cmd/posterbuild generates (scripts/posters.sh runs it). The server hands both
// to the /posters page; scripts/posters.sh renders the print files from them.
package posterfiles

import "embed"

// FS holds src/<id>.svg and qr/<id>.svg.
//
//go:embed src/*.svg qr/*.svg
var FS embed.FS
