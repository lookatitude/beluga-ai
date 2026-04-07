#!/usr/bin/env bash
# post-review-learn.sh — Extracts learnings after a review cycle completes.
# Cross-pollinates: writes to BOTH reviewer and implementer rules.
#
# Environment:
#   BELUGA_TASK_ID     — task identifier
#   BELUGA_REVIEW_LOG  — path to the reviewer's output log (temp file)
#   BELUGA_VERDICT     — "APPROVED" or "REJECTED"
#
# Output: writes learning rule files to both reviewer and implementer rules/
#         and appends entries to the learnings index.

set -euo pipefail

TEAMS_DIR="$(cd "$(dirname "$0")/.." && pwd)"
REVIEWER_RULES="$TEAMS_DIR/agents/reviewer/rules"
IMPLEMENTER_RULES="$TEAMS_DIR/agents/implementer/rules"
LEARNINGS_INDEX="$TEAMS_DIR/state/learnings-index.md"
REVIEW_LOG="${BELUGA_REVIEW_LOG:-/dev/null}"
DATE="$(date -u +%Y-%m-%d)"
TASK_ID="${BELUGA_TASK_ID:-unknown}"
VERDICT="${BELUGA_VERDICT:-unknown}"

# Extract issues found and positive patterns
ISSUES=""
POSITIVE=""
if [ -f "$REVIEW_LOG" ]; then
    ISSUES=$(grep -i -E "(critical|high|medium|issue|violation|missing)" "$REVIEW_LOG" | head -20 || true)
    POSITIVE=$(grep -i -E "(clean|correct|well.implemented|good pattern|approved)" "$REVIEW_LOG" | head -10 || true)
fi

# Ensure rules directories exist and generate race-safe filenames
TIMESTAMP="$(date -u +%Y%m%dT%H%M%SZ)"
mkdir -p "$REVIEWER_RULES"
mkdir -p "$IMPLEMENTER_RULES"
# Sanitize TASK_ID to alphanumeric/dash/underscore only
SAFE_TASK_ID="$(echo "$TASK_ID" | tr -cd 'a-zA-Z0-9_-')"
SUFFIX="review-${SAFE_TASK_ID}-${TIMESTAMP}.md"

# Write to reviewer's rules (what to watch for)
if [ -n "$ISSUES" ] || [ -n "$POSITIVE" ]; then
    cat > "$REVIEWER_RULES/$SUFFIX" << RULEEOF
---
source: post-review-learn
date: $DATE
trigger: review of task $TASK_ID — verdict $VERDICT
---

## Issues Found

${ISSUES:-None}

## Positive Patterns

${POSITIVE:-None}

## Reviewer Rule

Watch for these patterns in future reviews.
RULEEOF

    echo "- [reviewer] $DATE [post-review-learn] — review $TASK_ID ($VERDICT) → $SUFFIX" >> "$LEARNINGS_INDEX"
fi

# Cross-pollinate: write to implementer's rules (what to avoid/repeat)
if [ -n "$ISSUES" ] || [ -n "$POSITIVE" ]; then
    cat > "$IMPLEMENTER_RULES/$SUFFIX" << RULEEOF
---
source: post-review-learn (cross-pollination)
date: $DATE
trigger: review feedback on task $TASK_ID — verdict $VERDICT
---

## Issues To Avoid

${ISSUES:-None — all clean}

## Patterns To Repeat

${POSITIVE:-None noted}

## Implementer Rule

Apply these learnings in future implementations.
RULEEOF

    echo "- [implementer] $DATE [post-review-learn] — cross-pollinated from review $TASK_ID → $SUFFIX" >> "$LEARNINGS_INDEX"
fi

echo "Review learnings extracted for task $TASK_ID (verdict: $VERDICT)"
