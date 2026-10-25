#!/usr/bin/env bash
# Tests for scripts/release-version.sh against throwaway git repositories.
set -euo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPT="${HERE}/release-version.sh"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT
fail=0

# new_repo: a repo whose HEAD is tagged v2.13.6
new_repo() {
  rm -rf "$TMP/r"; mkdir -p "$TMP/r"; cd "$TMP/r"
  git init -q -b main
  git config user.email t@example.com; git config user.name t
  git config commit.gpgsign false; git config tag.gpgsign false
  echo base > core.go; git add .; git commit -qm "feat: base"
  git tag -a v2.13.6 -m v2.13.6
}
commit() { # path message
  mkdir -p "$(dirname "$1")"; echo "$RANDOM" >> "$1"; git add -A; git commit -qm "$2"
}
expect() { # name want
  local got
  got=$("$SCRIPT" 2>/dev/null | paste -sd' ' -)
  if [ "$got" = "$2" ]; then echo "ok   $1"; else echo "FAIL $1: want '$2', got '$got'"; fail=1; fi
}

new_repo; expect "HEAD already tagged" "skip=true"

new_repo; commit .github/workflows/ci.yml "ci: tweak workflow"
expect "workflow-only change does not release" "skip=true"

new_repo; commit docs/guide.md "docs: guide"; commit .guild/wiki/x.md "docs: wiki"; commit README.md "docs: readme"
expect "docs/Guild/Markdown-only change does not release" "skip=true"

new_repo; commit Makefile "chore: tidy Makefile"; commit .golangci.yml "ci: lint config"
expect "chore/ci-only code change does not release" "skip=true"

new_repo; commit llm/x.go "fix(llm): handle nil"
expect "fix -> patch" "tag=v2.13.7 skip=false"

new_repo; commit llm/x.go "fix(llm): handle nil"; commit agent/a.go "feat(agent): add hook"
expect "feat -> minor" "tag=v2.14.0 skip=false"

new_repo; commit go.mod "chore(deps): bump the go-minor-patch group"
expect "dependency bump -> patch" "tag=v2.13.7 skip=false"

new_repo; commit .github/workflows/ci.yml "fix(ci): repair workflow"
expect "fix that only touches workflows does not release" "skip=true"

new_repo; commit core/c.go "feat(core)!: drop old API"
expect "breaking subject skips" "skip=true"

new_repo; mkdir -p x; echo a > x/a.go; git add -A; git commit -qm "fix: thing" -m "BREAKING CHANGE: renamed"
expect "breaking body skips" "skip=true"

rm -rf "$TMP/r"; mkdir -p "$TMP/r"; cd "$TMP/r"; git init -q -b main
git config user.email t@example.com; git config user.name t; git config commit.gpgsign false
echo a > a.go; git add .; git commit -qm "feat: first"
expect "no tag yet" "tag=v0.0.1 skip=false"

exit "$fail"
