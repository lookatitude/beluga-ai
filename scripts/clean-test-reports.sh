#!/bin/bash
# Clean up old test-analyzer reports

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

cd "$REPO_ROOT"

echo "Cleaning up old test-analyzer reports..."

# Remove root-level reports
rm -f all-packages-report.* report.html report.json report.md

# Remove reports directory (optional - uncomment if you want to remove all)
# rm -rf reports/test-analyzer/

echo "✓ Cleaned up old reports"
echo ""
echo "Remaining reports in reports/test-analyzer/:"
ls -lh reports/test-analyzer/ 2>/dev/null || echo "  (none)"

