#!/usr/bin/env bash
# Decide whether HEAD should be auto-released and compute the next version.
# Used by the release-tag job in .github/workflows/main.yml.
#
# Prints "tag=<vX.Y.Z>" and "skip=<true|false>" lines to stdout and, when
# $GITHUB_OUTPUT is set, appends them there too. Explanations go to stderr.
#
# Rules, in order:
#   1. No v*.*.* tag yet          -> v0.0.1
#   2. HEAD already tagged        -> skip
#   3. Only workflows (.github/), agent/Guild config (.claude/, .guild/),
#      docs/ or Markdown changed since the last tag -> skip. This is the
#      release filter; main.yml itself runs CI on every push.
#   4. Breaking change (type!: or "BREAKING CHANGE" in a body) -> skip;
#      majors are released manually through release.yml.
#   5. No feat/fix/perf/revert commit and no "<type>(deps):" commit since
#      the last tag -> skip. chore/ci/docs/test/refactor-only ranges ship
#      with the next releasable commit.
#   6. Any feat -> minor bump; otherwise patch bump.
set -euo pipefail

emit() {
  echo "$1"
  if [ -n "${GITHUB_OUTPUT:-}" ]; then echo "$1" >> "$GITHUB_OUTPUT"; fi
}
log() { echo "$*" >&2; }

LATEST=$(git tag -l 'v*.*.*' --sort=-v:refname | head -1 || true)
if [ -z "$LATEST" ]; then
  log "No existing tag found; starting at v0.0.1"
  emit "tag=v0.0.1"
  emit "skip=false"
  exit 0
fi
log "Latest tag: ${LATEST}"

if git describe --exact-match --tags HEAD >/dev/null 2>&1; then
  log "HEAD is already tagged; skipping auto-release."
  emit "skip=true"
  exit 0
fi

RELEASE_RELEVANT=$(git diff --name-only "${LATEST}..HEAD" \
  | grep -Ev '^(\.github/|\.claude/|\.guild/|docs/)|\.md$' || true)
if [ -z "$RELEASE_RELEVANT" ]; then
  log "Only workflows/docs/Guild/Markdown changed since ${LATEST}; skipping release."
  emit "skip=true"
  exit 0
fi
log "Release-relevant files changed since ${LATEST}:"
log "$RELEASE_RELEVANT"

V="${LATEST#v}"
MAJOR="${V%%.*}"
REST="${V#*.}"
MINOR="${REST%%.*}"
PATCH="${REST#*.}"

HAS_BREAKING=false
HAS_FEAT=false
HAS_RELEASABLE=false

# NUL-delimited records so multi-line bodies are safe: subject<NUL>body<NUL>
while IFS= read -r -d '' SUBJECT && IFS= read -r -d '' BODY; do
  if echo "$SUBJECT" | grep -qE '^[a-z]+(\([^)]*\))?!:'; then
    log "  BREAKING (subject): ${SUBJECT}"
    HAS_BREAKING=true
  fi
  if echo "$BODY" | grep -qi 'BREAKING CHANGE'; then
    log "  BREAKING (body): ${SUBJECT}"
    HAS_BREAKING=true
  fi
  if echo "$SUBJECT" | grep -qE '^feat(\([^)]*\))?:'; then
    log "  feat: ${SUBJECT}"
    HAS_FEAT=true
  fi
  if echo "$SUBJECT" | grep -qE '^(feat|fix|perf|revert)(\([^)]*\))?:|^[a-z]+\(deps\):'; then
    HAS_RELEASABLE=true
  fi
done < <(git log "${LATEST}..HEAD" --pretty=format:"%s%x00%b%x00")

if [ "$HAS_BREAKING" = true ]; then
  log "Breaking changes detected; skipping auto-release. Publish majors via the Release workflow."
  emit "skip=true"
  exit 0
fi

if [ "$HAS_RELEASABLE" != true ]; then
  log "No feat/fix/perf/revert or (deps) commits since ${LATEST}; skipping release."
  emit "skip=true"
  exit 0
fi

if [ "$HAS_FEAT" = true ]; then
  NEW_TAG="v${MAJOR}.$((MINOR + 1)).0"
  log "feat detected: minor bump ${LATEST} -> ${NEW_TAG}"
else
  NEW_TAG="v${MAJOR}.${MINOR}.$((PATCH + 1))"
  log "patch bump ${LATEST} -> ${NEW_TAG}"
fi
emit "tag=${NEW_TAG}"
emit "skip=false"
