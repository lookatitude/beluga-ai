#!/bin/bash
# Generate JSON coverage report from go tool cover output
# Usage: ./scripts/generate-coverage-json.sh <coverprofile> <output.json>

set -e

COVERPROFILE="${1:-coverage/baseline.out}"
OUTPUT="${2:-coverage/baseline-report.json}"

if [ ! -f "$COVERPROFILE" ]; then
    echo "Error: Coverage profile $COVERPROFILE not found" >&2
    exit 1
fi

# Parse go tool cover -func output and convert to JSON
{
    echo "{"
    echo "  \"timestamp\": \"$(date -u +%Y-%m-%dT%H:%M:%SZ)\","
    echo "  \"packages\": ["
    
    FIRST=true
    go tool cover -func="$COVERPROFILE" | while IFS= read -r line; do
        if [[ $line =~ ^total:.*\(([0-9.]+)%\)$ ]]; then
            TOTAL_COVERAGE="${BASH_REMATCH[1]}"
            continue
        fi
        
        if [[ $line =~ ^([^[:space:]]+)[[:space:]]+([0-9.]+)%[[:space:]]+of[[:space:]]+statements ]]; then
            PACKAGE="${BASH_REMATCH[1]}"
            COVERAGE="${BASH_REMATCH[2]}"
            
            if [ "$FIRST" = true ]; then
                FIRST=false
            else
                echo ","
            fi
            
            echo -n "    {"
            echo -n "\"package\": \"$PACKAGE\","
            echo -n "\"coverage\": $COVERAGE"
            echo -n "}"
        fi
    done
    
    echo ""
    echo "  ],"
    echo "  \"total_coverage\": $TOTAL_COVERAGE"
    echo "}"
} > "$OUTPUT"

echo "Coverage report generated: $OUTPUT"
