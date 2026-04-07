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

TEAMS_DIR="$(cd "$(dirname "$0")/.." && pwd)"
VALID_AGENTS="arch-analyst implementer reviewer doc-writer website-dev notion-syncer"

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

AGENT_RULES_DIR="$TEAMS_DIR/agents/$BELUGA_AGENT_NAME/rules"
LEARNINGS_INDEX="$TEAMS_DIR/state/learnings-index.md"
TASK_LOG="${BELUGA_TASK_LOG:-/dev/null}"
DATE="$(date -u +%Y-%m-%d)"
TIMESTAMP="$(date -u +%Y-%m-%dT%H:%M:%SZ)"

mkdir -p "$AGENT_RULES_DIR"
RULE_COUNT=$(find "$AGENT_RULES_DIR" -name "*.md" -type f 2>/dev/null | wc -l)
RULE_FILE="$AGENT_RULES_DIR/learning-${BELUGA_TASK_ID}-$(( RULE_COUNT + 1 )).md"

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
else
    echo "No learnings to extract from task $BELUGA_TASK_ID"
fi
