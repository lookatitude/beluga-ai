#!/usr/bin/env bash
# Run every Go native fuzz target in the module, one package at a time.
#
# `go test -fuzz` accepts exactly one package, so `go test ./... -fuzz=X`
# fails with "cannot use -fuzz flag with multiple packages". This script
# lists the targets per package and fuzzes each one in its own package.
#
# Usage: scripts/fuzz.sh [fuzztime] [extra go test flags...]
#   fuzztime defaults to $FUZZ_TIME or 30s, and must look like 30s, 5m or 1h.
# Exit status: 0 when every target passes (or there are none), 1 otherwise.
set -euo pipefail

GO="${GO:-go}"
FUZZ_TIME="${1:-${FUZZ_TIME:-30s}}"
if [ "$#" -gt 0 ]; then shift; fi

if ! [[ "$FUZZ_TIME" =~ ^[0-9]+[smh]$ ]]; then
  echo "fuzz.sh: fuzztime must look like 30s, 5m or 1h (got '${FUZZ_TIME}')" >&2
  exit 2
fi

# `go test -list` prints the matching names for a package, then a line
# "ok  <pkg>  <time>". Pair each name with the package that follows it.
PAIRS=$("$GO" test -run='^$' -list='^Fuzz' ./... 2>/dev/null \
  | awk '/^Fuzz/ {t[n++]=$1; next} /^ok/ {for (i = 0; i < n; i++) print $2, t[i]; n = 0; next} {n = 0}')

if [ -z "$PAIRS" ]; then
  echo "No fuzz targets found."
  exit 0
fi

FAILED=0
while read -r pkg target; do
  echo "=== Fuzzing ${pkg} ${target} for ${FUZZ_TIME} ==="
  if ! "$GO" test "$pkg" -run='^$' -fuzz="^${target}\$" -fuzztime="$FUZZ_TIME" "$@"; then
    FAILED=$((FAILED + 1))
    echo "FAILED: ${pkg} ${target}"
  fi
done <<< "$PAIRS"

if [ -n "${GITHUB_OUTPUT:-}" ]; then
  echo "fuzz_failures=${FAILED}" >> "$GITHUB_OUTPUT"
fi
if [ "$FAILED" -gt 0 ]; then
  echo "${FAILED} fuzz target(s) failed."
  exit 1
fi
