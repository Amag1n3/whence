#!/usr/bin/env bash
#
# The staleness check for the committed rasters, and the one piece shared
# between generating them and verifying them.
#
#   bash scripts/icons-lock.sh write    -> refresh public/icons.lock
#   bash scripts/icons-lock.sh check    -> exit 1 if a source moved since
#
# WHY HASH THE SOURCES RATHER THAN THE PNGs
#
# The obvious check is to re-render in CI and diff the images. That check
# would be red on every pull request: this repo's icons are generated on
# macOS and CI runs Linux, and the two rasterise text differently — font
# hinting and antialiasing alone guarantee og-image.png never matches
# byte-for-byte. A gate that fires when nothing is wrong gets switched off
# within a week, which is the same argument /docs makes for why `whence
# check` fails on damage rather than on proximity.
#
# Hashing the sources catches the failure that actually happened — someone
# changed the palette, updated the SVGs, and never re-ran the renderer — and
# it needs no browser in CI at all.
#
# What it cannot catch: someone editing a PNG by hand, or a render that
# silently used a fallback font. Those need eyes, and the renderer says so
# when it finishes.

set -euo pipefail

cd "$(dirname "$0")/.."

LOCK="public/icons.lock"

# The vector sources every committed raster is derived from.
SOURCES=(
  public/favicon.svg
  apple-touch-icon.svg
  og-card.html
)

# macOS ships shasum, most Linux images ship sha256sum, CI could be either.
hash_sources() {
  if command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "${SOURCES[@]}"
  else
    sha256sum "${SOURCES[@]}"
  fi
}

case "${1:-check}" in
  write)
    {
      echo "# sha256 of the sources public/*.png and og-image.png were rendered from."
      echo "# Regenerate with: pnpm icons"
      hash_sources
    } > "$LOCK"
    echo "wrote $LOCK"
    ;;

  check)
    if [ ! -f "$LOCK" ]; then
      echo "error: $LOCK is missing. Run: pnpm icons" >&2
      exit 1
    fi
    if diff -u <(grep -v '^#' "$LOCK") <(hash_sources) >/dev/null; then
      echo "icons are in sync with their sources"
    else
      echo "error: icon sources changed but the rasters were not regenerated." >&2
      echo >&2
      diff -u <(grep -v '^#' "$LOCK") <(hash_sources) >&2 || true
      echo >&2
      echo "Fix with:  cd web && pnpm icons   (then commit public/)" >&2
      exit 1
    fi
    ;;

  *)
    echo "usage: $0 [write|check]" >&2
    exit 2
    ;;
esac
