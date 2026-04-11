#!/usr/bin/env bash
# post-task-learn.sh — Extracts learnings after any agent completes a task.
# Called by the supervisor after an agent dispatch returns.
#
# Environment:
#   BELUGA_AGENT_NAME — validated agent name (arch-analyst, implementer, etc.)
#   BELUGA_TASK_ID    — task identifier from progress.json
#   BELUGA_TASK_LOG   — path to the agent's output log (temp file)
#
# Output: writes a learning rule file to the agent's rules/ directory
#         and appends an entry to the learnings index.

set -euo pipefail

CLAUDE_DIR="$(cd "$(dirname "$0")/.." && pwd)"
REPO_ROOT="$(cd "$CLAUDE_DIR/.." && pwd)"
WIKI_DIR="$REPO_ROOT/.wiki"
VALID_AGENTS="coordinator architect researcher developer-go developer-web reviewer-qa reviewer-security docs-writer marketeer notion-syncer"

# Validate agent name against allowlist
validate_agent_name() {
    local name="$1"
    for valid in $VALID_AGENTS; do
        if [ "$name" = "$valid" ]; then
            return 0
        fi
    done
    echo "ERROR: Invalid agent name: $name" >&2
    exit 1
}

# Ensure required env vars are set
if [ -z "${BELUGA_AGENT_NAME:-}" ] || [ -z "${BELUGA_TASK_ID:-}" ]; then
    echo "ERROR: BELUGA_AGENT_NAME and BELUGA_TASK_ID must be set" >&2
    exit 1
fi

validate_agent_name "$BELUGA_AGENT_NAME"

AGENT_RULES_DIR="$CLAUDE_DIR/agents/$BELUGA_AGENT_NAME/rules"
LEARNINGS_INDEX="$CLAUDE_DIR/state/learnings-index.md"
TASK_LOG="${BELUGA_TASK_LOG:-/dev/null}"
DATE="$(date -u +%Y-%m-%d)"
TIMESTAMP="$(date -u +%Y-%m-%dT%H:%M:%SZ)"

mkdir -p "$AGENT_RULES_DIR"
RULE_COUNT=$(find "$AGENT_RULES_DIR" -name "*.md" -type f 2>/dev/null | wc -l)
RULE_FILE="$AGENT_RULES_DIR/learning-${BELUGA_TASK_ID}-${TIMESTAMP}-$$.md"

# Extract key patterns from the task log:
# - Lines containing "error", "fail", "retry", "unexpected", "workaround"
# - Lines containing "fixed by", "resolved by", "solution"
ERRORS=""
SOLUTIONS=""
if [ -f "$TASK_LOG" ]; then
    ERRORS=$(grep -i -E "(error|fail|retry|unexpected|workaround)" "$TASK_LOG" | head -20 || true)
    SOLUTIONS=$(grep -i -E "(fixed by|resolved by|solution|the fix)" "$TASK_LOG" | head -10 || true)
fi

# Only write a learning if there's something to learn
if [ -n "$ERRORS" ] || [ -n "$SOLUTIONS" ]; then
    cat > "$RULE_FILE" << RULEEOF
---
source: post-task-learn
date: $DATE
trigger: task $BELUGA_TASK_ID completed by $BELUGA_AGENT_NAME
---

## Errors Encountered

$ERRORS

## Solutions Applied

$SOLUTIONS

## Rule

(Review and refine this rule based on the patterns above.)
RULEEOF

    # Append to learnings index
    echo "- [$BELUGA_AGENT_NAME] $DATE [post-task-learn] — task $BELUGA_TASK_ID learnings → $(basename "$RULE_FILE")" >> "$LEARNINGS_INDEX"

    echo "Learning extracted to $RULE_FILE"

    # Optional promotion to .wiki/corrections.md for HIGH-confidence findings.
    # Agents tag their log with "CONFIDENCE: HIGH" to trigger this promotion.
    if [ -f "$TASK_LOG" ] && grep -q "CONFIDENCE:[[:space:]]*HIGH" "$TASK_LOG" 2>/dev/null && [ -f "$WIKI_DIR/corrections.md" ]; then
        NEXT_ID=$(grep -c '^### C-' "$WIKI_DIR/corrections.md" 2>/dev/null || echo 0)
        NEXT_ID=$((NEXT_ID + 1))
        PRINTF_ID=$(printf "C-%03d" "$NEXT_ID")
        SUMMARY=$(grep -m1 -i -E "symptom|root cause|correction" "$TASK_LOG" 2>/dev/null | head -c 200 || echo "see $(basename "$RULE_FILE")")
        {
            echo ""
            echo "### $PRINTF_ID | $DATE | $BELUGA_AGENT_NAME | task-$BELUGA_TASK_ID"
            echo "**Auto-captured from per-agent rules (HIGH confidence).**"
            echo "**Summary:** $SUMMARY"
            echo "**Source:** \`$RULE_FILE\`"
            echo "**Confidence:** HIGH"
        } >> "$WIKI_DIR/corrections.md"
        echo "Promoted to .wiki/corrections.md as $PRINTF_ID"
    fi
else
    echo "No learnings to extract from task $BELUGA_TASK_ID"
fi
