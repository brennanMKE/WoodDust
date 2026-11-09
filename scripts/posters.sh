#!/usr/bin/env bash
#
# posters.sh — build the shop-poster print files (docs/posters.md).
#
#   scripts/posters.sh              stock QR codes, then print files for every published poster with art
#   scripts/posters.sh shave        just one poster
#   scripts/posters.sh --proof      also posters without art, using the placeholder block, into tmp/posters/
#   scripts/posters.sh --qr         only regenerate posters/qr/<id>.svg (embedded by the server)
#
# Sources: posters/src/<id>.svg (hand-made templates, presentation attributes
# only) and images/art/posters/<id>.png (generated art; it stays unpublished
# until Brennan approves a printed proof). Output, per poster, in
# images/posters/ (served at /images/posters/ and listed on /posters):
#
#   <id>-letter.pdf  <id>-a4.pdf  <id>-grey.pdf  <id>.png (Letter, 300 dpi)  <id>.svg (source)
#
# Fonts: Alfa Slab One and Source Sans 3 from web/public/fonts (OFL), through
# a private fontconfig file so nothing needs installing.
#
# Needs: go, rsvg-convert (brew install librsvg). Local only; it never
# touches a server. Commit the outputs only once the art is approved.

set -euo pipefail
REPO="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$REPO"

PROOF=0 QR_ONLY=0 IDS=()
for a in "$@"; do
  case "$a" in
    --proof) PROOF=1 ;;
    --qr) QR_ONLY=1 ;;
    -h|--help) sed -n '2,24p' "$0"; exit 0 ;;
    -*) echo "unknown option $a" >&2; exit 2 ;;
    *) IDS+=("$a") ;;
  esac
done

echo "== stock QR codes (posters/qr/)"
go run ./cmd/posterbuild qr
[ "$QR_ONLY" = 1 ] && exit 0

command -v rsvg-convert >/dev/null 2>&1 || { echo "rsvg-convert not found: brew install librsvg" >&2; exit 1; }

OUT=images/posters
[ "$PROOF" = 1 ] && OUT=tmp/posters
WORK="$(mktemp -d "${TMPDIR:-/tmp}/posters.XXXXXX")"
trap 'rm -rf "$WORK"' EXIT

cat >"$WORK/fonts.conf" <<CONF
<?xml version="1.0"?>
<!DOCTYPE fontconfig SYSTEM "fonts.dtd">
<fontconfig>
  <dir>$REPO/web/public/fonts</dir>
  <include ignore_missing="yes">/opt/homebrew/etc/fonts/fonts.conf</include>
  <include ignore_missing="yes">/etc/fonts/fonts.conf</include>
  <cachedir>$WORK/fc-cache</cachedir>
</fontconfig>
CONF
export FONTCONFIG_FILE="$WORK/fonts.conf"
export PANGOCAIRO_BACKEND=fc   # use fontconfig (and so web/public/fonts) even where pango defaults to CoreText

args=(compose -out "$WORK/svg")
[ "$PROOF" = 1 ] && args+=(-proof)
echo "== compose"
built="$(go run ./cmd/posterbuild "${args[@]}" ${IDS[@]+"${IDS[@]}"})"
[ -n "$built" ] || { echo "nothing to build (no approved art in images/art/posters/; try --proof)"; exit 0; }

mkdir -p "$OUT"
while read -r id kind; do
  echo "== $id ($kind)"
  S="$WORK/svg"
  rsvg-convert -f pdf -o "$OUT/$id-letter.pdf" "$S/$id-letter.svg"
  rsvg-convert -f pdf -o "$OUT/$id-a4.pdf" "$S/$id-a4.svg"
  rsvg-convert -f pdf -o "$OUT/$id-grey.pdf" "$S/$id-grey.svg"
  rsvg-convert -f png --dpi-x 300 --dpi-y 300 -o "$OUT/$id.png" "$S/$id-letter.svg"
  cp "$S/$id.svg" "$OUT/$id.svg"
  ls -lh "$OUT/$id"* | awk '{print "   " $5 "\t" $NF}'
done <<<"$built"
echo "done: $OUT/"
