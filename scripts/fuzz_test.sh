#!/usr/bin/env bash
# Regression test for scripts/fuzz.sh. Builds throwaway modules and checks:
#   - targets in two packages are both fuzzed (the old `make fuzz` failed
#     with "cannot use -fuzz flag with multiple packages"),
#   - a module with no fuzz targets exits 0,
#   - a crashing target exits 1,
#   - an invalid fuzztime is rejected with exit 2.
set -euo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
FUZZ="${HERE}/fuzz.sh"
GOVER="$(go env GOVERSION | sed -E 's/^go([0-9]+\.[0-9]+).*/\1/')"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

mkmod() { mkdir -p "$1"; printf 'module %s\n\ngo %s\n' "$2" "$GOVER" > "$1/go.mod"; }

fail=0
check() { # name expected actual
  if [ "$2" = "$3" ]; then echo "ok   $1"; else echo "FAIL $1: want exit $2, got $3"; fail=1; fi
}

# Two packages, one target each.
mkmod "$TMP/multi" multi
mkdir -p "$TMP/multi/a" "$TMP/multi/b"
printf 'package a\n\nimport "testing"\n\nfunc FuzzA(f *testing.F) { f.Fuzz(func(t *testing.T, s string) {}) }\n' > "$TMP/multi/a/a_test.go"
printf 'package b\n\nimport "testing"\n\nfunc FuzzB(f *testing.F) { f.Fuzz(func(t *testing.T, n int) {}) }\n' > "$TMP/multi/b/b_test.go"
set +e
out=$(cd "$TMP/multi" && "$FUZZ" 1s 2>&1); rc=$?
set -e
check "multi-package targets" 0 "$rc"
if ! grep -q 'Fuzzing multi/a FuzzA' <<< "$out" || ! grep -q 'Fuzzing multi/b FuzzB' <<< "$out"; then
  echo "FAIL multi-package targets: not every target ran"; echo "$out"; fail=1
fi

# No targets.
mkmod "$TMP/empty" empty
printf 'package empty\n' > "$TMP/empty/e.go"
set +e; (cd "$TMP/empty" && "$FUZZ" 1s >/dev/null 2>&1); rc=$?; set -e
check "no targets" 0 "$rc"

# Crashing target.
mkmod "$TMP/crash" crash
printf 'package crash\n\nimport "testing"\n\nfunc FuzzCrash(f *testing.F) {\n\tf.Add(0)\n\tf.Fuzz(func(t *testing.T, n int) { t.Fatal("boom") })\n}\n' > "$TMP/crash/c_test.go"
set +e; (cd "$TMP/crash" && "$FUZZ" 1s >/dev/null 2>&1); rc=$?; set -e
check "crashing target" 1 "$rc"

# Invalid fuzztime.
set +e; (cd "$TMP/empty" && "$FUZZ" '1s;id' >/dev/null 2>&1); rc=$?; set -e
check "invalid fuzztime" 2 "$rc"

exit "$fail"
