#!/bin/bash
# Comprehensive Package Compliance Audit Script
# Audits all Beluga AI framework packages for v2 compliance and generates detailed report
# Usage: ./scripts/comprehensive-audit.sh

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
PKG_DIR="$REPO_ROOT/pkg"
AUDIT_DIR="$REPO_ROOT/docs/audit"
REPORT_FILE="$REPO_ROOT/docs/audit/comprehensive-audit-report.md"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
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

# Initialize report data
declare -A PACKAGE_ISSUES
declare -A PACKAGE_WARNINGS
declare -A PACKAGE_STATUS
declare -A PACKAGE_DETAILS

# Check if a file exists
file_exists() {
    local file="$1"
    [ -f "$file" ]
}

# Check if a directory exists
dir_exists() {
    local dir="$1"
    [ -d "$dir" ]
}

# Check for OTEL metrics
check_otel_metrics() {
    local pkg_path="$1"
    local metrics_file="$pkg_path/metrics.go"
    
    if [ -f "$metrics_file" ]; then
        if grep -q "go.opentelemetry.io/otel" "$metrics_file" 2>/dev/null; then
            return 0
        fi
    fi
    
    # Check other files too
    if grep -r -q "go.opentelemetry.io/otel.*metric" "$pkg_path" --include="*.go" 2>/dev/null; then
        return 0
    fi
    
    return 1
}

# Check for OTEL tracing
check_otel_tracing() {
    local pkg_path="$1"
    
    if grep -r -q "go.opentelemetry.io/otel.*trace\|StartSpan\|tracer\." "$pkg_path" --include="*.go" 2>/dev/null; then
        return 0
    fi
    
    return 1
}

# Check for structured logging
check_structured_logging() {
    local pkg_path="$1"
    
    if grep -r -q "Logger\|logger\|zap\|slog\|logWithOTELContext\|LogWithOTELContext" "$pkg_path" --include="*.go" 2>/dev/null; then
        return 0
    fi
    
    return 1
}

# Audit a single package
audit_package() {
    local pkg_name="$1"
    local pkg_path="$PKG_DIR/$pkg_name"
    
    if [ ! -d "$pkg_path" ]; then
        echo -e "${RED}Error: Package $pkg_name not found at $pkg_path${NC}" >&2
        return 1
    fi
    
    local issues=0
    local warnings=0
    local missing_files=()
    local missing_dirs=()
    local otel_metrics=false
    local otel_tracing=false
    local structured_logging=false
    
    # Check required files
    for file in "${REQUIRED_FILES[@]}"; do
        local file_path="$pkg_path/$file"
        if ! file_exists "$file_path"; then
            ((issues++))
            missing_files+=("$file")
        fi
    done
    
    # Check required directories
    for dir in "${REQUIRED_DIRS[@]}"; do
        local dir_path="$pkg_path/$dir"
        if ! dir_exists "$dir_path"; then
            ((warnings++))
            missing_dirs+=("$dir")
        fi
    done
    
    # Check OTEL integration
    if check_otel_metrics "$pkg_path"; then
        otel_metrics=true
    else
        ((issues++))
    fi
    
    if check_otel_tracing "$pkg_path"; then
        otel_tracing=true
    else
        ((warnings++))
    fi
    
    if check_structured_logging "$pkg_path"; then
        structured_logging=true
    else
        ((warnings++))
    fi
    
    # Store results
    PACKAGE_ISSUES["$pkg_name"]=$issues
    PACKAGE_WARNINGS["$pkg_name"]=$warnings
    
    local details=""
    details+="Missing Files: ${missing_files[*]}\n"
    details+="Missing Dirs: ${missing_dirs[*]}\n"
    details+="OTEL Metrics: $otel_metrics\n"
    details+="OTEL Tracing: $otel_tracing\n"
    details+="Structured Logging: $structured_logging\n"
    PACKAGE_DETAILS["$pkg_name"]="$details"
    
    if [ $issues -eq 0 ] && [ $warnings -eq 0 ]; then
        PACKAGE_STATUS["$pkg_name"]="COMPLIANT"
        return 0
    elif [ $issues -eq 0 ]; then
        PACKAGE_STATUS["$pkg_name"]="WARNINGS"
        return 0
    else
        PACKAGE_STATUS["$pkg_name"]="NON_COMPLIANT"
        return 1
    fi
}

# Generate markdown report
generate_report() {
    local total_packages=0
    local compliant=0
    local warnings=0
    local non_compliant=0
    
    mkdir -p "$AUDIT_DIR"
    
    {
        echo "# Comprehensive Package Compliance Audit Report"
        echo ""
        echo "**Generated**: $(date -u +"%Y-%m-%d %H:%M:%S UTC")"
        echo "**Auditor**: Comprehensive Audit Script"
        echo ""
        echo "## Executive Summary"
        echo ""
        echo "This report provides a comprehensive audit of all Beluga AI framework packages for v2 compliance."
        echo ""
        echo "### Compliance Status"
        echo ""
        echo "| Status | Count | Packages |"
        echo "|--------|-------|----------|"
        
        # Count packages by status
        for pkg_name in "${!PACKAGE_STATUS[@]}"; do
            ((total_packages++))
            case "${PACKAGE_STATUS[$pkg_name]}" in
                COMPLIANT)
                    ((compliant++))
                    ;;
                WARNINGS)
                    ((warnings++))
                    ;;
                NON_COMPLIANT)
                    ((non_compliant++))
                    ;;
            esac
        done
        
        echo "| ✅ Compliant | $compliant | - |"
        echo "| ⚠️  Warnings | $warnings | - |"
        echo "| ❌ Non-Compliant | $non_compliant | - |"
        echo "| **Total** | **$total_packages** | - |"
        echo ""
        echo "## Package Details"
        echo ""
        
        # Sort packages alphabetically
        IFS=$'\n' sorted_packages=($(sort <<<"${!PACKAGE_STATUS[*]}"))
        
        for pkg_name in "${sorted_packages[@]}"; do
            local status="${PACKAGE_STATUS[$pkg_name]}"
            local issue_count="${PACKAGE_ISSUES[$pkg_name]}"
            local warning_count="${PACKAGE_WARNINGS[$pkg_name]}"
            local details="${PACKAGE_DETAILS[$pkg_name]}"
            
            echo "### pkg/$pkg_name/"
            echo ""
            
            case "$status" in
                COMPLIANT)
                    echo "**Status**: ✅ Full Compliance"
                    ;;
                WARNINGS)
                    echo "**Status**: ⚠️  Has Warnings ($warning_count warning(s))"
                    ;;
                NON_COMPLIANT)
                    echo "**Status**: ❌ Non-Compliant ($issue_count issue(s), $warning_count warning(s))"
                    ;;
            esac
            
            echo ""
            echo "**Issues**: $issue_count"
            echo "**Warnings**: $warning_count"
            echo ""
            
            # Detailed breakdown
            echo "#### Required Files"
            echo ""
            for file in "${REQUIRED_FILES[@]}"; do
                local file_path="$PKG_DIR/$pkg_name/$file"
                if file_exists "$file_path"; then
                    echo "- [x] \`$file\` - **PRESENT** ✓"
                else
                    echo "- [ ] \`$file\` - **MISSING** ✗"
                fi
            done
            
            echo ""
            echo "#### Required Directories"
            echo ""
            for dir in "${REQUIRED_DIRS[@]}"; do
                local dir_path="$PKG_DIR/$pkg_name/$dir"
                if dir_exists "$dir_path"; then
                    echo "- [x] \`$dir/\` - **PRESENT** ✓"
                else
                    echo "- [ ] \`$dir/\` - **MISSING** ○"
                fi
            done
            
            echo ""
            echo "#### OTEL Integration"
            echo ""
            
            local pkg_path="$PKG_DIR/$pkg_name"
            if check_otel_metrics "$pkg_path"; then
                echo "- [x] OTEL metrics: **PRESENT** ✓"
            else
                echo "- [ ] OTEL metrics: **MISSING** ✗"
            fi
            
            if check_otel_tracing "$pkg_path"; then
                echo "- [x] OTEL tracing: **PRESENT** ✓"
            else
                echo "- [ ] OTEL tracing: **MISSING** ○"
            fi
            
            if check_structured_logging "$pkg_path"; then
                echo "- [x] Structured logging: **PRESENT** ✓"
            else
                echo "- [ ] Structured logging: **MISSING** ○"
            fi
            
            echo ""
            echo "---"
            echo ""
        done
        
        echo "## Summary of Inconsistencies"
        echo ""
        echo "### Missing Required Files"
        echo ""
        for file in "${REQUIRED_FILES[@]}"; do
            local missing_packages=()
            for pkg_name in "${sorted_packages[@]}"; do
                local file_path="$PKG_DIR/$pkg_name/$file"
                if ! file_exists "$file_path"; then
                    missing_packages+=("$pkg_name")
                fi
            done
            
            if [ ${#missing_packages[@]} -gt 0 ]; then
                echo "- **$file**: ${missing_packages[*]}"
            fi
        done
        
        echo ""
        echo "### Missing Required Directories"
        echo ""
        for dir in "${REQUIRED_DIRS[@]}"; do
            local missing_packages=()
            for pkg_name in "${sorted_packages[@]}"; do
                local dir_path="$PKG_DIR/$pkg_name/$dir"
                if ! dir_exists "$dir_path"; then
                    missing_packages+=("$pkg_name")
                fi
            done
            
            if [ ${#missing_packages[@]} -gt 0 ]; then
                echo "- **$dir/**: ${missing_packages[*]}"
            fi
        done
        
        echo ""
        echo "### Missing OTEL Integration"
        echo ""
        local missing_metrics=()
        local missing_tracing=()
        local missing_logging=()
        
        for pkg_name in "${sorted_packages[@]}"; do
            local pkg_path="$PKG_DIR/$pkg_name"
            if ! check_otel_metrics "$pkg_path"; then
                missing_metrics+=("$pkg_name")
            fi
            if ! check_otel_tracing "$pkg_path"; then
                missing_tracing+=("$pkg_name")
            fi
            if ! check_structured_logging "$pkg_path"; then
                missing_logging+=("$pkg_name")
            fi
        done
        
        if [ ${#missing_metrics[@]} -gt 0 ]; then
            echo "- **OTEL Metrics**: ${missing_metrics[*]}"
        fi
        if [ ${#missing_tracing[@]} -gt 0 ]; then
            echo "- **OTEL Tracing**: ${missing_tracing[*]}"
        fi
        if [ ${#missing_logging[@]} -gt 0 ]; then
            echo "- **Structured Logging**: ${missing_logging[*]}"
        fi
        
    } > "$REPORT_FILE"
    
    echo -e "${GREEN}Report generated: $REPORT_FILE${NC}"
}

# Main execution
main() {
    echo "Comprehensive Package Compliance Audit"
    echo "======================================="
    echo ""
    
    local total_issues=0
    local total_warnings=0
    local compliant=0
    local non_compliant=0
    local with_warnings=0
    
    for pkg_dir in "$PKG_DIR"/*; do
        if [ -d "$pkg_dir" ]; then
            pkg_name=$(basename "$pkg_dir")
            echo -e "${BLUE}Auditing: $pkg_name${NC}"
            
            if audit_package "$pkg_name"; then
                if [ "${PACKAGE_STATUS[$pkg_name]}" = "COMPLIANT" ]; then
                    ((compliant++))
                else
                    ((with_warnings++))
                fi
            else
                ((non_compliant++))
            fi
            
            total_issues=$((total_issues + PACKAGE_ISSUES[$pkg_name]))
            total_warnings=$((total_warnings + PACKAGE_WARNINGS[$pkg_name]))
        fi
    done
    
    echo ""
    echo "======================================="
    echo "Summary:"
    echo "  ✅ Compliant packages: $compliant"
    echo "  ⚠️  Packages with warnings: $with_warnings"
    echo "  ❌ Non-compliant packages: $non_compliant"
    echo "  Total issues: $total_issues"
    echo "  Total warnings: $total_warnings"
    echo ""
    
    generate_report
    
    if [ $non_compliant -eq 0 ] && [ $total_issues -eq 0 ]; then
        echo -e "${GREEN}All packages are v2 compliant!${NC}"
        exit 0
    else
        echo -e "${YELLOW}Some packages need alignment. See report for details.${NC}"
        exit 1
    fi
}

main "$@"
