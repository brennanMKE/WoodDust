package posters

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"regexp"
	"strconv"
	"strings"
)

// Print-file composition for scripts/posters.sh (via cmd/posterbuild). The
// browser fills the same template slots itself for the plaque builder.

var (
	reXMLDecl  = regexp.MustCompile(`^\s*<\?xml[^>]*\?>\s*`)
	reComment  = regexp.MustCompile(`(?s)<!--.*?-->\s*`)
	reArtImage = regexp.MustCompile(`<image data-slot="art"[^>]*/>`)
	reRootSize = regexp.MustCompile(`<svg ([^>]*?)width="1024" height="1536"`)
	reBG       = regexp.MustCompile(`data-bg="(#[0-9a-fA-F]{3,6})"`)
	reHex      = regexp.MustCompile(`#[0-9a-fA-F]{6}\b`)
)

// greyHex maps a #rrggbb colour to its luma grey (Rec. 601).
func greyHex(c string) string {
	if len(c) != 7 {
		return c
	}
	v, err := strconv.ParseUint(c[1:], 16, 32)
	if err != nil {
		return c
	}
	r, g, b := float64(v>>16&0xff), float64(v>>8&0xff), float64(v&0xff)
	y := int(0.299*r + 0.587*g + 0.114*b + 0.5)
	return fmt.Sprintf("#%02x%02x%02x", y, y, y)
}

// Compose fills a template: the art inlined as a data URI (or its slot
// removed so the placeholder shows, when art is nil) and the QR code inlined.
func Compose(template string, art []byte, qrSVG string) string {
	s := reComment.ReplaceAllString(reXMLDecl.ReplaceAllString(template, ""), "")
	if art == nil {
		s = reArtImage.ReplaceAllString(s, "<!-- art missing: placeholder shown -->")
	} else {
		s = strings.Replace(s, `data-slot="art" href="art.png"`, `data-slot="art" href="data:image/png;base64,`+base64.StdEncoding.EncodeToString(art)+`"`, 1)
	}
	return strings.Replace(s, `data-slot="qr" href="qr.svg"`, `data-slot="qr" href="data:image/svg+xml;base64,`+base64.StdEncoding.EncodeToString([]byte(qrSVG))+`"`, 1)
}

// Sheet is a paper size in millimetres.
type Sheet struct {
	Name string
	W, H float64
}

// Sheets are the Letter and A4 print files, from one layout.
var Sheets = []Sheet{{"letter", 215.9, 279.4}, {"a4", 210, 297}}

// Background returns the template's page colour (data-bg), for the margins
// a 2:3 sheet leaves on Letter and A4.
func Background(template string) string {
	if m := reBG.FindStringSubmatch(template); m != nil {
		return m[1]
	}
	return "#ffffff"
}

// Greyscale converts PNG art to 8-bit grey, for the low-toner version.
func Greyscale(pngData []byte) ([]byte, error) {
	src, err := png.Decode(bytes.NewReader(pngData))
	if err != nil {
		return nil, err
	}
	g := image.NewGray(src.Bounds())
	draw.Draw(g, g.Bounds(), src, src.Bounds().Min, draw.Src)
	var out bytes.Buffer
	if err := png.Encode(&out, g); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

// Page wraps a composed poster on a sheet: full-bleed background, the
// 1024 x 1536 layout scaled to fit (it fills the height; the 2:3 art leaves
// even margins at the sides, in the page colour). grey also desaturates the
// vector parts (slogan, plaque, background) for low-toner shop printers;
// pass art already made grey by Greyscale.
func Page(composed, bg string, sh Sheet, grey bool) string {
	if grey {
		bg = greyHex(bg)
		composed = reHex.ReplaceAllStringFunc(composed, greyHex)
	}
	inner := reRootSize.ReplaceAllString(composed, fmt.Sprintf(`<svg ${1}width="%g" height="%g" preserveAspectRatio="xMidYMid meet"`, sh.W, sh.H))
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>`+"\n"+
		`<svg xmlns="http://www.w3.org/2000/svg" width="%gmm" height="%gmm" viewBox="0 0 %g %g">`+"\n"+
		"<rect width=\"%g\" height=\"%g\" fill=\"%s\"/>\n%s\n</svg>\n", sh.W, sh.H, sh.W, sh.H, sh.W, sh.H, bg, inner)
}
