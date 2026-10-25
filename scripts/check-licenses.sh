#!/usr/bin/env bash
# License compliance gate, shared by CI (_security-checks.yml) and `make licenses`.
#
# The exit code of `go-licenses check` is the verdict. Its stderr also carries
# glog `W…` warnings and lists of .s/.c files, so the violation count is taken
# from the real violation lines only. stderr is kept in $LICENSE_ERRORS_FILE
# (default license-errors.txt).
#
# --ignore entries were reviewed by hand (2026-10-25):
#   github.com/segmentio/asm — MIT No Attribution (MIT-0), which go-licenses'
#     classifier does not recognise.
#   github.com/asg017/sqlite-vec-go-bindings — the Go module ships no LICENSE
#     file; upstream sqlite-vec is MIT OR Apache-2.0.
#   github.com/nexus-rpc/nexus-proto-annotations — the Go module ships no
#     LICENSE file (pulled in transitively by the Temporal SDK).
# Re-review these when their versions change.
set -uo pipefail

ERRORS="${LICENSE_ERRORS_FILE:-license-errors.txt}"
VIOLATION_RE='^(Did not find license|Forbidden|Not allowed)|license.*not allowed'

go-licenses check ./... \
  --allowed_licenses=MIT,Apache-2.0,BSD-2-Clause,BSD-3-Clause,ISC,MPL-2.0 \
  --ignore github.com/segmentio/asm \
  --ignore github.com/asg017/sqlite-vec-go-bindings \
  --ignore github.com/nexus-rpc/nexus-proto-annotations \
  2> "$ERRORS"
rc=$?

violations=$(grep -cE "$VIOLATION_RE" "$ERRORS" || true)
grep -E "$VIOLATION_RE" "$ERRORS" || true

if [ -n "${GITHUB_OUTPUT:-}" ]; then
  echo "rc=${rc}" >> "$GITHUB_OUTPUT"
  echo "violations=${violations:-0}" >> "$GITHUB_OUTPUT"
fi
if [ "$rc" -ne 0 ]; then
  echo "go-licenses check failed (exit ${rc}, ${violations:-0} violation line(s)); full output in ${ERRORS}" >&2
fi
exit "$rc"
