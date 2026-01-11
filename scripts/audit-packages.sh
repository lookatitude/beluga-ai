#!/bin/bash
# Package Compliance Audit Script
# Audits all Beluga AI framework packages for v2 compliance
# Usage: ./scripts/audit-packages.sh [package_name]

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
PKG_DIR="$REPO_ROOT/pkg"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Required files for v2 compliance
REQUIRED_FILES=(
    "config.go"
    "metrics.go"
    "errors.go"
    "test_utils.go"
    "advanced_test.go"
    "README.md"
)

# Required directories (if applicable)
REQUIRED_DIRS=(
    "iface"
)

# Optional directories (for multi-provider packages)
OPTIONAL_DIRS=(
    "providers"
    "internal"
)

# Check if a file exists
file_exists() {
    local file="$1"
    if [ -f "$file" ]; then
        echo -e "${GREEN}✓${NC}"
        return 0
    else
        echo -e "${RED}✗${NC}"
        return 1
    fi
}

# Check if a directory exists
dir_exists() {
    local dir="$1"
    if [ -d "$dir" ]; then
        echo -e "${GREEN}✓${NC}"
        return 0
    else
        echo -e "${YELLOW}○${NC}"
        return 1
    fi
}

# Audit a single package
audit_package() {
    local pkg_name="$1"
    local pkg_path="$PKG_DIR/$pkg_name"
    
    if [ ! -d "$pkg_path" ]; then
        echo -e "${RED}Error: Package $pkg_name not found at $pkg_path${NC}" >&2
        return 1
    fi
    
    echo "Auditing package: $pkg_name"
    echo "=================================="
    
    local issues=0
    local warnings=0
    
    # Check required files
    echo ""
    echo "Required Files:"
    for file in "${REQUIRED_FILES[@]}"; do
        local file_path="$pkg_path/$file"
        echo -n "  $file: "
        if ! file_exists "$file_path"; then
            ((issues++))
        fi
    done
    
    # Check required directories
    echo ""
    echo "Required Directories:"
    for dir in "${REQUIRED_DIRS[@]}"; do
        local dir_path="$pkg_path/$dir"
        echo -n "  $dir/: "
        if ! dir_exists "$dir_path"; then
            ((warnings++))
        fi
    done
    
    # Check optional directories
    echo ""
    echo "Optional Directories:"
    for dir in "${OPTIONAL_DIRS[@]}"; do
        local dir_path="$pkg_path/$dir"
        echo -n "  $dir/: "
        dir_exists "$dir_path" > /dev/null || true
    done
    
    # Check for OTEL integration
    echo ""
    echo "OTEL Integration:"
    if grep -q "go.opentelemetry.io/otel" "$pkg_path/metrics.go" 2>/dev/null; then
        echo -e "  OTEL metrics: ${GREEN}✓${NC}"
    else
        echo -e "  OTEL metrics: ${RED}✗${NC}"
        ((issues++))
    fi
    
    # Check for structured logging
    if grep -r -q "Logger\|logger\|zap\|slog" "$pkg_path" --include="*.go" 2>/dev/null; then
        echo -e "  Structured logging: ${GREEN}✓${NC}"
    else
        echo -e "  Structured logging: ${YELLOW}○${NC}"
        ((warnings++))
    fi
    
    # Check for test coverage
    echo ""
    echo "Testing:"
    if [ -f "$pkg_path/advanced_test.go" ]; then
        echo -e "  advanced_test.go: ${GREEN}✓${NC}"
    else
        echo -e "  advanced_test.go: ${RED}✗${NC}"
        ((issues++))
    fi
    
    if [ -f "$pkg_path/test_utils.go" ]; then
        echo -e "  test_utils.go: ${GREEN}✓${NC}"
    else
        echo -e "  test_utils.go: ${RED}✗${NC}"
        ((issues++))
    fi
    
    # Summary
    echo ""
    echo "=================================="
    if [ $issues -eq 0 ] && [ $warnings -eq 0 ]; then
        echo -e "${GREEN}✓ Package $pkg_name is v2 compliant${NC}"
        return 0
    elif [ $issues -eq 0 ]; then
        echo -e "${YELLOW}○ Package $pkg_name has $warnings warning(s)${NC}"
        return 0
    else
        echo -e "${RED}✗ Package $pkg_name has $issues issue(s) and $warnings warning(s)${NC}"
        return 1
    fi
}

# Main execution
main() {
    if [ $# -eq 0 ]; then
        # Audit all packages
        echo "Auditing all Beluga AI framework packages for v2 compliance"
        echo "============================================================"
        echo ""
        
        local total_issues=0
        local total_warnings=0
        local compliant=0
        local non_compliant=0
        
        for pkg_dir in "$PKG_DIR"/*; do
            if [ -d "$pkg_dir" ]; then
                pkg_name=$(basename "$pkg_dir")
                if audit_package "$pkg_name"; then
                    ((compliant++))
                else
                    ((non_compliant++))
                fi
                echo ""
            fi
        done
        
        echo "============================================================"
        echo "Summary:"
        echo "  Compliant packages: $compliant"
        echo "  Non-compliant packages: $non_compliant"
        
        if [ $non_compliant -eq 0 ]; then
            echo -e "${GREEN}All packages are v2 compliant!${NC}"
            exit 0
        else
            echo -e "${RED}Some packages need alignment${NC}"
            exit 1
        fi
    else
        # Audit specific package
        audit_package "$1"
    fi
}

main "$@"
