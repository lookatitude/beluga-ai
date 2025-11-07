#!/bin/bash

# generate-test-reports.sh
# Auto-generates test analyzer reports for all packages and updates the dashboard

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Directories
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
REPORTS_DIR="$PROJECT_ROOT/reports/test-analyzer"
BIN_DIR="$PROJECT_ROOT/bin"
ANALYZER="$BIN_DIR/test-analyzer"

# Packages to analyze
PACKAGES=(
    "pkg/llms"
    "pkg/memory"
    "pkg/orchestration"
    "pkg/agents"
)

# Create reports directory if it doesn't exist
mkdir -p "$REPORTS_DIR"

echo -e "${BLUE}🧪 Test Analyzer Report Generator${NC}"
echo -e "${BLUE}===================================${NC}\n"

# Check if analyzer exists
if [ ! -f "$ANALYZER" ]; then
    echo -e "${YELLOW}⚠️  Analyzer not found. Building...${NC}"
    cd "$PROJECT_ROOT"
    go build -o "$ANALYZER" ./cmd/test-analyzer
    if [ $? -ne 0 ]; then
        echo -e "${RED}❌ Failed to build analyzer${NC}"
        exit 1
    fi
    echo -e "${GREEN}✅ Analyzer built successfully${NC}\n"
fi

# Function to analyze a package and generate report
analyze_package() {
    local package=$1
    local package_name=$(basename "$package")
    local report_file="$REPORTS_DIR/${package_name}_detailed_report.md"
    
    echo -e "${BLUE}📊 Analyzing ${package}...${NC}"
    
    # Run analyzer and capture JSON output for stats
    local json_output=$(mktemp)
    local json_success=false
    
    # Try to get JSON output (suppress stderr for cleaner output)
    if "$ANALYZER" --dry-run --output json "$package" > "$json_output" 2>/dev/null; then
        json_success=true
    fi
    
    # Extract stats from JSON (with fallbacks)
    local files="0"
    local functions="0"
    local issues="0"
    
    if [ "$json_success" = true ] && [ -s "$json_output" ]; then
        files=$(python3 -c "import json, sys; d=json.load(sys.stdin); print(d['summary']['files_analyzed'])" < "$json_output" 2>/dev/null || echo "0")
        functions=$(python3 -c "import json, sys; d=json.load(sys.stdin); print(d['summary']['functions_analyzed'])" < "$json_output" 2>/dev/null || echo "0")
        issues=$(python3 -c "import json, sys; d=json.load(sys.stdin); print(d['summary']['issues_found'])" < "$json_output" 2>/dev/null || echo "0")
    fi
    
    # Generate markdown report
    if "$ANALYZER" --dry-run --output markdown --output-file "$report_file" "$package" 2>/dev/null; then
        echo -e "${GREEN}✅ Generated ${package_name}_detailed_report.md${NC}"
        echo -e "   Files: ${files}, Functions: ${functions}, Issues: ${issues}"
        rm -f "$json_output"
        return 0
    else
        echo -e "${RED}❌ Failed to generate report for ${package}${NC}"
        rm -f "$json_output"
        return 1
    fi
}

# Function to generate summary report
generate_summary() {
    echo -e "\n${BLUE}📋 Generating summary report...${NC}"
    
    local summary_file="$REPORTS_DIR/analysis_summary.md"
    
    # Collect stats from all packages
    local total_files=0
    local total_functions=0
    local total_issues=0
    local package_stats=()
    
    for package in "${PACKAGES[@]}"; do
        local package_name=$(basename "$package")
        local json_output=$(mktemp)
        
        if "$ANALYZER" --dry-run --output json "$package" > "$json_output" 2>/dev/null && [ -s "$json_output" ]; then
            local files=$(python3 -c "import json, sys; d=json.load(sys.stdin); print(d['summary']['files_analyzed'])" < "$json_output" 2>/dev/null || echo "0")
            local functions=$(python3 -c "import json, sys; d=json.load(sys.stdin); print(d['summary']['functions_analyzed'])" < "$json_output" 2>/dev/null || echo "0")
            local issues=$(python3 -c "import json, sys; d=json.load(sys.stdin); print(d['summary']['issues_found'])" < "$json_output" 2>/dev/null || echo "0")
            
            # Convert to integers for arithmetic
            files=${files:-0}
            functions=${functions:-0}
            issues=${issues:-0}
            
            total_files=$((total_files + files))
            total_functions=$((total_functions + functions))
            total_issues=$((total_issues + issues))
            
            package_stats+=("$package_name|$files|$functions|$issues")
        else
            # Use defaults if analysis failed
            package_stats+=("$package_name|0|0|0")
        fi
        
        rm -f "$json_output"
    done
    
    # Generate summary markdown
    cat > "$summary_file" << EOF
# Test Analyzer - Complete Analysis Summary

**Generated**: $(date +"%Y-%m-%d %H:%M:%S")  
**Total Packages Analyzed**: ${#PACKAGES[@]}  
**Total Test Files**: $total_files  
**Total Test Functions**: $total_functions  
**Total Issues Found**: $total_issues

---

## Package Comparison

| Package | Files | Functions | Issues | Issues/Func |
|---------|-------|-----------|--------|-------------|
EOF

    for stat in "${package_stats[@]}"; do
        IFS='|' read -r name files functions issues <<< "$stat"
        local density="0.00"
        if [ "$functions" -gt 0 ]; then
            density=$(awk "BEGIN {printf \"%.2f\", $issues/$functions}" 2>/dev/null || echo "0.00")
        fi
        echo "| **$name** | $files | $functions | $issues | $density |" >> "$summary_file"
    done
    
    cat >> "$summary_file" << EOF

---

## Key Insights

1. **Total Issues**: $total_issues across $total_functions test functions
2. **Average Issues per Function**: $(awk "BEGIN {printf \"%.2f\", $total_issues/$total_functions}" 2>/dev/null || echo "0.00")
3. **Package Coverage**: ${#PACKAGES[@]} packages analyzed

---

## Detailed Reports Available

- \`llms_detailed_report.md\` - pkg/llms analysis
- \`memory_detailed_report.md\` - pkg/memory analysis
- \`orchestration_detailed_report.md\` - pkg/orchestration analysis
- \`agents_detailed_report.md\` - pkg/agents analysis

---

## Next Steps

1. Review detailed reports for each package
2. Prioritize fixes based on package criticality
3. Implement automated fixes where possible
4. Track progress with regular analysis runs

---

*Generated by generate-test-reports.sh*
EOF

    echo -e "${GREEN}✅ Generated analysis_summary.md${NC}"
}

# Function to update dashboard with current data
update_dashboard() {
    echo -e "\n${BLUE}🎨 Dashboard information...${NC}"
    
    # Note: Dashboard HTML uses static data for simplicity
    # For dynamic updates, consider:
    # 1. Using JavaScript to load JSON data
    # 2. Using a templating engine (Go templates, Jinja2, etc.)
    # 3. Generating HTML from a template with current data
    
    echo -e "${GREEN}✅ Dashboard available at: ${REPORTS_DIR}/dashboard.html${NC}"
    echo -e "${YELLOW}ℹ️  To view: open ${REPORTS_DIR}/dashboard.html in your browser${NC}"
}

# Main execution
main() {
    cd "$PROJECT_ROOT"
    
    local success_count=0
    local fail_count=0
    
    # Analyze each package
    for package in "${PACKAGES[@]}"; do
        if analyze_package "$package"; then
            success_count=$((success_count + 1))
        else
            fail_count=$((fail_count + 1))
        fi
        echo ""
    done
    
    # Generate summary
    generate_summary
    
    # Update dashboard
    update_dashboard
    
    # Summary
    echo -e "\n${BLUE}===================================${NC}"
    echo -e "${GREEN}✅ Report Generation Complete!${NC}"
    echo -e "${BLUE}===================================${NC}\n"
    echo -e "📊 Packages analyzed: ${success_count}/${#PACKAGES[@]}"
    echo -e "📁 Reports directory: ${REPORTS_DIR}"
    echo -e "🌐 Dashboard: ${REPORTS_DIR}/dashboard.html\n"
    
    if [ $fail_count -gt 0 ]; then
        echo -e "${YELLOW}⚠️  Some packages failed to analyze${NC}"
        exit 1
    fi
}

# Run main function
main

