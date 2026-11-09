package posters

import (
	"fmt"
	"strings"

	qrcode "github.com/skip2/go-qrcode"
)

// QRSVG renders text as a QR code: pure black modules on white, with the
// standard 4-module quiet zone, as one path. Presentation attributes only,
// so it is safe under the site's strict CSP when inlined.
func QRSVG(text string) (string, error) {
	q, err := qrcode.New(text, qrcode.Medium)
	if err != nil {
		return "", err
	}
	bm := q.Bitmap() // includes the quiet zone
	n := len(bm)
	var d strings.Builder
	for y, row := range bm {
		for x := 0; x < len(row); {
			if !row[x] {
				x++
				continue
			}
			start := x
			for x < len(row) && row[x] {
				x++
			}
			fmt.Fprintf(&d, "M%d %dh%dv1h-%dz", start, y, x-start, x-start)
		}
	}
	return fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %d %d" width="%d" height="%d" shape-rendering="crispEdges">`+
		`<rect width="%d" height="%d" fill="#ffffff"/><path fill="#000000" d="%s"/></svg>`+"\n",
		n, n, n*8, n*8, n, n, d.String()), nil
}
