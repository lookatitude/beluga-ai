#!/bin/bash
# Package Alignment Helper Script
# Helps align a package to v2 standards by creating missing files and directories
# Usage: ./scripts/align-package.sh <package_name>

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
PKG_DIR="$REPO_ROOT/pkg"
TEMPLATES_DIR="$REPO_ROOT/docs/templates"

if [ $# -eq 0 ]; then
    echo "Usage: $0 <package_name>"
    echo "Example: $0 core"
    exit 1
fi

PACKAGE_NAME="$1"
PACKAGE_PATH="$PKG_DIR/$PACKAGE_NAME"

if [ ! -d "$PACKAGE_PATH" ]; then
    echo "Error: Package $PACKAGE_NAME not found at $PACKAGE_PATH" >&2
    exit 1
fi

echo "Aligning package: $PACKAGE_NAME"
echo "=================================="

# Create required directories
echo "Creating required directories..."
mkdir -p "$PACKAGE_PATH/iface"
mkdir -p "$PACKAGE_PATH/internal"
echo "✓ Directories created"

# Check and create required files
echo ""
echo "Checking required files..."

# Check config.go
if [ ! -f "$PACKAGE_PATH/config.go" ]; then
    echo "⚠ config.go missing - may be acceptable for some packages (core, schema)"
fi

# Check metrics.go
if [ ! -f "$PACKAGE_PATH/metrics.go" ]; then
    echo "⚠ metrics.go missing - will need to be created"
    echo "  Use template: docs/templates/otel-metrics.go.template"
fi

# Check errors.go
if [ ! -f "$PACKAGE_PATH/errors.go" ]; then
    if [ -f "$PACKAGE_PATH/iface/errors.go" ]; then
        echo "✓ errors.go found in iface/ (acceptable)"
    else
        echo "⚠ errors.go missing - will need to be created"
    fi
fi

# Check test_utils.go
if [ ! -f "$PACKAGE_PATH/test_utils.go" ]; then
    echo "⚠ test_utils.go missing - will need to be created"
    echo "  Use template: docs/templates/test-utils.go.template"
fi

# Check advanced_test.go
if [ ! -f "$PACKAGE_PATH/advanced_test.go" ]; then
    echo "⚠ advanced_test.go missing - will need to be created"
    echo "  Use template: docs/templates/advanced-test.go.template"
fi

echo ""
echo "=================================="
echo "Package alignment check complete!"
echo ""
echo "Next steps:"
echo "1. Review audit report: docs/audit/${PACKAGE_NAME}-compliance.md"
echo "2. Create missing files using templates in docs/templates/"
echo "3. Add OTEL integration following patterns in pkg/monitoring/"
echo "4. Run tests: go test ./pkg/$PACKAGE_NAME/..."
echo "5. Verify compliance: ./scripts/audit-packages.sh $PACKAGE_NAME"
