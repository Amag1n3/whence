#!/usr/bin/env bash
# Lookup a curated record, rewrite the anchored lines, fail `whence check`.
#
# Records are authored with `whence add` (hashes computed), not captured from
# a session. Run from anywhere:
#
#   bash example/demo.sh
#
# Exit 0 means the gate failed as designed (check exited 1). Any other
# outcome is a broken demo.

set -euo pipefail

ROOT=$(cd "$(dirname "$0")/.." && pwd)
FIXTURE="$ROOT/example/src/auth/session.go"

if [[ ! -f "$FIXTURE" ]]; then
  echo "demo: missing fixture $FIXTURE" >&2
  exit 2
fi

WHENCE=${WHENCE:-}
cleanup() {
  [[ -n "${WORKDIR:-}" && -d "$WORKDIR" ]] && rm -rf "$WORKDIR"
  [[ -n "${BINDIR:-}" && -d "$BINDIR" ]] && rm -rf "$BINDIR"
}
trap cleanup EXIT

if [[ -z "$WHENCE" ]]; then
  BINDIR=$(mktemp -d "${TMPDIR:-/tmp}/whence-demo-bin.XXXXXX")
  WHENCE="$BINDIR/whence"
  (cd "$ROOT" && go build -o "$WHENCE" .)
fi

WORKDIR=$(mktemp -d "${TMPDIR:-/tmp}/whence-demo.XXXXXX")
mkdir -p "$WORKDIR/src/auth"
# Drop the build tag — this copy is a file in a throwaway repo, not a Go package.
tail -n +3 "$FIXTURE" >"$WORKDIR/src/auth/session.go"

cd "$WORKDIR"
git init -q -b main
git config user.email "demo@whence.local"
git config user.name "whence demo"
git add src/auth/session.go
git commit -q -m "checkout session keys"

start=$(grep -n 'CHECKOUT_userToken' src/auth/session.go | head -1 | cut -d: -f1)
end=$(grep -n 'CHECKOUT_role' src/auth/session.go | head -1 | cut -d: -f1)

echo "=== 1. seed one record (whence add, not capture) ==="
"$WHENCE" add "src/auth/session.go:${start}-${end}" \
  -s "code review, finding B5" \
  -d "Never write shared session keys from the checkout flow — namespace all three to CHECKOUT_*." \
  -w '"userToken", "userId" and "role" are all read by the staff dashboard on the same origin. Writing them here signs an operator out mid-session.'

git add .whence
git commit -q -m "record the session-key namespace"

echo
echo "=== 2. lookup (expect intact) ==="
"$WHENCE" src/auth/session.go

echo
echo "=== 3. rewrite the anchored lines ==="
# Same damage as the orphan case in check_test.go: the three Set calls vanish.
{
  sed -n "1,$((start - 1))p" src/auth/session.go
  printf '\tpersistNamespaced(s)\n'
  sed -n "$((end + 1)),\$p" src/auth/session.go
} >src/auth/session.go.tmp
mv src/auth/session.go.tmp src/auth/session.go

echo
echo "=== 4. lookup again (expect ORPHANED) ==="
"$WHENCE" src/auth/session.go

echo
echo "=== 5. whence check --base HEAD (expect exit 1) ==="
set +e
"$WHENCE" check --base HEAD
code=$?
set -e
echo "check exit: $code"
if [[ "$code" -ne 1 ]]; then
  echo "demo: expected whence check to exit 1 (damage), got $code" >&2
  exit 2
fi
echo
echo "demo ok: lookup showed the record, the rewrite orphaned it, check exited 1."
