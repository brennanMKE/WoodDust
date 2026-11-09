// Command posterbuild prepares the shop-poster print files (docs/posters.md).
// scripts/posters.sh runs it, then renders its SVGs with rsvg-convert.
//
//	go run ./cmd/posterbuild qr                       write posters/qr/<id>.svg (stock QR codes; embedded by the server)
//	go run ./cmd/posterbuild compose -out DIR [-proof] [id...]
//
// compose writes, per published poster (or each id given):
//
//	DIR/<id>.svg              the filled template (art and QR inlined): the SVG source download
//	DIR/<id>-letter.svg       Letter sheet
//	DIR/<id>-a4.svg           A4 sheet
//	DIR/<id>-grey.svg         Letter sheet, greyscale
//
// Art comes from images/art/posters/<id>.png. A poster without art is
// skipped unless -proof, which uses the template's placeholder block.
// It prints one line per poster: "<id> <art|placeholder>".
package main

import (
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	posterfiles "github.com/brennanMKE/WoodDust/posters"

	"github.com/brennanMKE/WoodDust/internal/posters"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: posterbuild qr | compose -out DIR [-proof] [id...]")
		os.Exit(2)
	}
	var err error
	switch os.Args[1] {
	case "qr":
		err = writeQR("posters/qr")
	case "compose":
		err = compose(os.Args[2:])
	default:
		err = fmt.Errorf("unknown command %q", os.Args[1])
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "posterbuild:", err)
		os.Exit(1)
	}
}

func writeQR(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	for _, p := range posters.Catalog {
		svg, err := posters.QRSVG(p.StockURL())
		if err != nil {
			return fmt.Errorf("%s: %w", p.ID, err)
		}
		if err := os.WriteFile(filepath.Join(dir, p.ID+".svg"), []byte(svg), 0o644); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "qr %-10s %s\n", p.ID, p.StockURL())
	}
	return nil
}

func compose(args []string) error {
	fset := flag.NewFlagSet("compose", flag.ContinueOnError)
	out := fset.String("out", "", "output directory")
	proof := fset.Bool("proof", false, "use the placeholder block for missing art")
	art := fset.String("art", "images/art/posters", "directory of <id>.png art")
	if err := fset.Parse(args); err != nil {
		return err
	}
	if *out == "" {
		return errors.New("-out is required")
	}
	var list []posters.Poster
	if fset.NArg() == 0 {
		list = posters.Published()
	}
	for _, id := range fset.Args() {
		p, ok := posters.Get(id)
		if !ok {
			return fmt.Errorf("unknown poster %q", id)
		}
		list = append(list, p)
	}
	if err := os.MkdirAll(*out, 0o755); err != nil {
		return err
	}
	for _, p := range list {
		tmpl, err := fs.ReadFile(posterfiles.FS, "src/"+p.ID+".svg")
		if err != nil {
			fmt.Fprintf(os.Stderr, "skip %s: no template posters/src/%s.svg\n", p.ID, p.ID)
			continue
		}
		png, err := os.ReadFile(filepath.Join(*art, p.ID+".png"))
		kind := "art"
		if err != nil {
			if !*proof {
				fmt.Fprintf(os.Stderr, "skip %s: no art at %s/%s.png (use -proof for a placeholder proof)\n", p.ID, *art, p.ID)
				continue
			}
			png, kind = nil, "placeholder"
		}
		composed := posters.Compose(string(tmpl), png, posters.StockLink(posterfiles.FS, p).QRSVG)
		bg := posters.Background(string(tmpl))
		files := map[string]string{p.ID + ".svg": composed}
		for _, sh := range posters.Sheets {
			files[p.ID+"-"+sh.Name+".svg"] = posters.Page(composed, bg, sh, false)
		}
		greyArt := png
		if png != nil {
			if greyArt, err = posters.Greyscale(png); err != nil {
				return fmt.Errorf("%s: %w", p.ID, err)
			}
		}
		files[p.ID+"-grey.svg"] = posters.Page(posters.Compose(string(tmpl), greyArt, posters.StockLink(posterfiles.FS, p).QRSVG), bg, posters.Sheets[0], true)
		for name, body := range files {
			if err := os.WriteFile(filepath.Join(*out, name), []byte(body), 0o644); err != nil {
				return err
			}
		}
		fmt.Printf("%s %s\n", p.ID, kind)
	}
	return nil
}
