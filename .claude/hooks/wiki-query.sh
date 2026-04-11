#!/usr/bin/env bash
# wiki-query.sh — Retrieval helper for agents.
#
# Given a topic (usually a package name or concept), prints:
#   - Matching entries from .wiki/index.md
#   - Matching corrections from .wiki/corrections.md
#   - Matching pattern files
#   - Matching package-map entry
#
# Usage: .claude/hooks/wiki-query.sh <topic>

set -euo pipefail

if [ $# -lt 1 ]; then
    echo "Usage: $0 <topic>" >&2
    exit 1
fi

TOPIC="$1"
REPO_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
WIKI_DIR="$REPO_ROOT/.wiki"

if [ ! -d "$WIKI_DIR" ]; then
    echo "ERROR: $WIKI_DIR does not exist. Run /wiki-learn to populate." >&2
    exit 1
fi

# Case-insensitive topic match
TOPIC_RE="$(printf '%s' "$TOPIC" | sed 's/[][\.^$*+?()|{}\\]/\\&/g')"

separator() {
    printf '\n=== %s ===\n\n' "$1"
}

separator "INDEX matches"
if [ -f "$WIKI_DIR/index.md" ]; then
    grep -n -i -F "$TOPIC" "$WIKI_DIR/index.md" 2>/dev/null || echo "(no matches in index.md)"
else
    echo "(index.md missing)"
fi

separator "CORRECTIONS matches"
if [ -f "$WIKI_DIR/corrections.md" ]; then
    awk -v topic_re="$TOPIC_RE" '
        BEGIN { IGNORECASE=1; in_entry=0; match_found=0; buf="" }
        /^### C-/ {
            if (in_entry && match_found) { printf "%s\n", buf }
            in_entry=1; match_found=0; buf=$0
            if ($0 ~ topic_re) match_found=1
            next
        }
        in_entry {
            buf = buf "\n" $0
            if ($0 ~ topic_re) match_found=1
        }
        END { if (in_entry && match_found) { printf "%s\n", buf } }
    ' "$WIKI_DIR/corrections.md" 2>/dev/null || echo "(no matches in corrections.md)"
else
    echo "(corrections.md missing)"
fi

separator "PATTERN files referencing topic"
if [ -d "$WIKI_DIR/patterns" ]; then
    grep -l -i -F "$TOPIC" "$WIKI_DIR/patterns"/*.md 2>/dev/null || echo "(no pattern files reference '$TOPIC')"
fi

separator "PACKAGE MAP entry"
if [ -f "$WIKI_DIR/architecture/package-map.md" ]; then
    awk -v topic="$TOPIC" '
        BEGIN { IGNORECASE=1; in_section=0 }
        /^## / {
            if (in_section) exit
            if (tolower($0) ~ tolower(topic)) { in_section=1; print; next }
        }
        in_section { print }
    ' "$WIKI_DIR/architecture/package-map.md" || echo "(no package-map entry for '$TOPIC')"
else
    echo "(package-map.md missing)"
fi

separator "ARCHITECTURE decisions referencing topic"
if [ -f "$WIKI_DIR/architecture/decisions.md" ]; then
    grep -n -i -F "$TOPIC" "$WIKI_DIR/architecture/decisions.md" 2>/dev/null || echo "(no matches)"
fi

separator "RECENT LOG entries referencing topic"
if [ -f "$WIKI_DIR/log.md" ]; then
    grep -n -i -F "$TOPIC" "$WIKI_DIR/log.md" 2>/dev/null | tail -10 || echo "(no matches)"
fi

exit 0
