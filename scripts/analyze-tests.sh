#!/bin/bash
# Convenience wrapper script for test-analyzer tool

set -e

# Get the directory where this script is located
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Build the tool if it doesn't exist
TOOL_BIN="$REPO_ROOT/bin/test-analyzer"
if [ ! -f "$TOOL_BIN" ]; then
    echo "Building test-analyzer..."
    cd "$REPO_ROOT"
    go build -o "$TOOL_BIN" ./cmd/test-analyzer
fi

# Run the tool with all passed arguments
exec "$TOOL_BIN" "$@"

