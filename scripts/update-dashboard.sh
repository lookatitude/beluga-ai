#!/bin/bash

# update-dashboard.sh
# Updates the dashboard.html with current analysis data

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
REPORTS_DIR="$PROJECT_ROOT/reports/test-analyzer"
DASHBOARD="$REPORTS_DIR/dashboard.html"
ANALYZER="$PROJECT_ROOT/bin/test-analyzer"

PACKAGES=("pkg/llms" "pkg/memory" "pkg/orchestration" "pkg/agents")

# Collect data from all packages
collect_data() {
    local total_files=0
    local total_functions=0
    local total_issues=0
    local package_data=()
    
    for package in "${PACKAGES[@]}"; do
        local package_name=$(basename "$package")
        local json_output=$(mktemp)
        
        if "$ANALYZER" --dry-run --output json "$package" > "$json_output" 2>/dev/null && [ -s "$json_output" ]; then
            local files=$(python3 -c "import json, sys; d=json.load(sys.stdin); print(d['summary']['files_analyzed'])" < "$json_output" 2>/dev/null || echo "0")
            local functions=$(python3 -c "import json, sys; d=json.load(sys.stdin); print(d['summary']['functions_analyzed'])" < "$json_output" 2>/dev/null || echo "0")
            local issues=$(python3 -c "import json, sys; d=json.load(sys.stdin); print(d['summary']['issues_found'])" < "$json_output" 2>/dev/null || echo "0")
            local timeout_pct=$(python3 -c "import json, sys; d=json.load(sys.stdin); print(round((d['summary']['functions_analyzed'] - d['issues_by_type'].get('MissingTimeout', 0)) / max(d['summary']['functions_analyzed'], 1) * 100, 1))" < "$json_output" 2>/dev/null || echo "0")
            local mock_pct=$(python3 -c "import json, sys; d=json.load(sys.stdin); total=d['summary']['functions_analyzed']; mock=total - d['issues_by_type'].get('ActualImplementationUsage', 0) - d['issues_by_type'].get('MixedMockRealUsage', 0); print(round(mock / max(total, 1) * 100, 1))" < "$json_output" 2>/dev/null || echo "0")
            
            files=${files:-0}
            functions=${functions:-0}
            issues=${issues:-0}
            
            total_files=$((total_files + files))
            total_functions=$((total_functions + functions))
            total_issues=$((total_issues + issues))
            
            # Calculate issue density
            local density=$(echo "scale=2; $issues / $functions" | bc 2>/dev/null || echo "0.00")
            
            # Determine grade
            local grade="F"
            if (( $(echo "$density < 1.5" | bc -l 2>/dev/null || echo 0) )); then
                grade="C+"
            elif (( $(echo "$density < 1.6" | bc -l 2>/dev/null || echo 0) )); then
                grade="B-"
            elif (( $(echo "$density < 1.8" | bc -l 2>/dev/null || echo 0) )); then
                grade="D+"
            else
                grade="D"
            fi
            
            package_data+=("$package_name|$files|$functions|$issues|$density|$timeout_pct|$mock_pct|$grade")
        fi
        
        rm -f "$json_output"
    done
    
    echo "$total_files|$total_functions|$total_issues|${package_data[@]}"
}

# Update dashboard HTML (simplified - would need more sophisticated templating)
update_dashboard() {
    local data=$(collect_data)
    IFS='|' read -r total_files total_functions total_issues rest <<< "$data"
    
    # For now, we'll create a simple update script
    # A full implementation would use a proper templating engine
    echo "Dashboard data collected:"
    echo "  Total Files: $total_files"
    echo "  Total Functions: $total_functions"
    echo "  Total Issues: $total_issues"
    echo ""
    echo "Note: Dashboard HTML uses static data. For dynamic updates,"
    echo "consider using a templating engine or JavaScript to load JSON data."
}

update_dashboard

