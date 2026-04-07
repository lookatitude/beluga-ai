#!/usr/bin/env bash
# post-build-learn.sh — Extracts learnings from go build/test/vet failures.
# Writes to implementer's rules directory.
#
# Environment:
#   BELUGA_TASK_ID    — task identifier
#   BELUGA_BUILD_LOG  — path to combined build/test/vet output (temp file)
#
# Output: writes a learning rule file to implementer's rules/
#         and appends an entry to the learnings index.

set -euo pipefail

TEAMS_DIR="$(cd "$(dirname "$0")/.." && pwd)"
IMPLEMENTER_RULES="$TEAMS_DIR/agents/implementer/rules"
LEARNINGS_INDEX="$TEAMS_DIR/state/learnings-index.md"
BUILD_LOG="${BELUGA_BUILD_LOG:-/dev/null}"
DATE="$(date -u +%Y-%m-%d)"
TASK_ID="${BELUGA_TASK_ID:-unknown}"

# Only proceed if the build log exists and contains failures
if [ ! -f "$BUILD_LOG" ]; then
    echo "No build log found, skipping"
    exit 0
fi

# Extract build errors, test failures, and vet warnings
BUILD_ERRORS=$(grep -E "^(.*\.go:[0-9]+:[0-9]+:)" "$BUILD_LOG" | head -20 || true)
TEST_FAILURES=$(grep -E "(FAIL|panic:|--- FAIL)" "$BUILD_LOG" | head -20 || true)
VET_WARNINGS=$(grep -E "(go vet|suspicious|unreachable|shadow)" "$BUILD_LOG" | head -10 || true)
IMPORT_CYCLES=$(grep -i "import cycle" "$BUILD_LOG" | head -5 || true)

# Only write if there's something to learn from
if [ -n "$BUILD_ERRORS" ] || [ -n "$TEST_FAILURES" ] || [ -n "$VET_WARNINGS" ] || [ -n "$IMPORT_CYCLES" ]; then
    mkdir -p "$IMPLEMENTER_RULES"
    TIMESTAMP="$(date -u +%Y%m%dT%H%M%SZ)"
    # Sanitize TASK_ID to alphanumeric/dash/underscore only
    SAFE_TASK_ID="$(echo "$TASK_ID" | tr -cd 'a-zA-Z0-9_-')"
    RULE_FILE="$IMPLEMENTER_RULES/build-${SAFE_TASK_ID}-${TIMESTAMP}.md"

    cat > "$RULE_FILE" << RULEEOF
---
source: post-build-learn
date: $DATE
trigger: build/test/vet failure during task $TASK_ID
---

## Build Errors

${BUILD_ERRORS:-None}

## Test Failures

${TEST_FAILURES:-None}

## Vet Warnings

${VET_WARNINGS:-None}

## Import Cycles

${IMPORT_CYCLES:-None}

## Implementer Rule

Avoid these patterns. Check for these issues before signaling task completion.
RULEEOF

    echo "- [implementer] $DATE [post-build-learn] — build failures in task $TASK_ID → $(basename "$RULE_FILE")" >> "$LEARNINGS_INDEX"
    echo "Build learning extracted to $RULE_FILE"
else
    echo "Build passed clean, no learnings to extract"
fi
