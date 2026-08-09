#!/usr/bin/env bash
#
# Regenerate the raster icons and the social card from their sources.
#
#   public/favicon-96.png       <- public/favicon.svg
#   public/favicon-192.png      <- public/favicon.svg
#   public/apple-touch-icon.png <- apple-touch-icon.svg
#   public/og-image.png         <- og-card.html
#
# The four PNGs are build output that happens to be committed, and nothing
# regenerated them when the palette changed — so they sat two repaints behind
# the site (warm sepia, then amber-on-neutral) while every vector source was
# already correct. This script exists so the next repaint is one command
# instead of an archaeology exercise.
#
# Uses headless Chrome rather than rsvg/ImageMagick because Chrome is the only
# one of them already on this machine, and because og-card.html is a real HTML
# document with webfonts — a pure SVG rasteriser could not render it at all.
#
# Usage:  pnpm icons        (from web/)
#    or:  bash scripts/make-icons.sh

set -euo pipefail

cd "$(dirname "$0")/.."

CHROME="/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"
if [ ! -x "$CHROME" ]; then
  echo "error: Google Chrome not found at:" >&2
  echo "  $CHROME" >&2
  echo "Install Chrome, or point CHROME at another Chromium build." >&2
  exit 1
fi

TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

# $1 = file:// target, $2 = width, $3 = height, $4 = output path
shoot() {
  "$CHROME" \
    --headless \
    --disable-gpu \
    --hide-scrollbars \
    --force-device-scale-factor=1 \
    --default-background-color=00000000 \
    --screenshot="$4" \
    --window-size="$2,$3" \
    "$1" >/dev/null 2>&1
  echo "  $(basename "$4")  ${2}x${3}"
}

# Chrome screenshots a viewport, not a file, so an SVG has to be dropped into
# a page that pins it to exactly the viewport box. Sizing via 100vw/100vh
# rather than the SVG's own width/height means the source can declare any
# intrinsic size without changing the output.
wrap_svg() {
  local svg="$1" out="$2"
  {
    printf '<!doctype html><meta charset="utf-8"><style>'
    printf 'html,body{margin:0;padding:0;background:transparent}'
    printf 'svg{display:block;width:100vw;height:100vh}'
    printf '</style>'
    cat "$svg"
  } > "$out"
}

echo "Rendering icons:"

wrap_svg public/favicon.svg "$TMP/favicon.html"
shoot "file://$TMP/favicon.html" 96 96 "$PWD/public/favicon-96.png"
shoot "file://$TMP/favicon.html" 192 192 "$PWD/public/favicon-192.png"

wrap_svg apple-touch-icon.svg "$TMP/apple.html"
shoot "file://$TMP/apple.html" 180 180 "$PWD/public/apple-touch-icon.png"

# og-card.html declares 1200x630 on .card itself; the window matches so there
# is nothing to crop.
shoot "file://$PWD/og-card.html" 1200 630 "$PWD/public/og-image.png"

echo
echo "Done. Check public/og-image.png before pushing — it carries webfonts,"
echo "and a font that failed to load renders as a fallback without erroring."
