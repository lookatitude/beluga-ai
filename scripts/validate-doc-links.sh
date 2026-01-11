#!/bin/bash
# Validate documentation links (basic check for markdown link syntax)

set -e

echo "=== Validating Documentation Links ==="

# Check for broken markdown links in key documentation files
check_file() {
    local file=$1
    if [ ! -f "$file" ]; then
        echo "⚠️  File not found: $file"
        return 1
    fi
    
    # Check for markdown links that might be broken
    # Look for patterns like [text](./path) or [text](../path)
    broken_links=$(grep -n '\](' "$file" | grep -v 'http' | grep -v 'https' | grep -v 'mailto' || true)
    
    if [ -n "$broken_links" ]; then
        echo "Checking links in $file..."
        # Basic validation - just check file exists for relative paths
        while IFS= read -r line; do
            link=$(echo "$line" | sed -n 's/.*](\([^)]*\)).*/\1/p')
            if [[ "$link" == ./* ]] || [[ "$link" == ../* ]]; then
                # Extract file path
                file_path=$(dirname "$file")/$link
                if [ ! -f "$file_path" ] && [ ! -d "$file_path" ]; then
                    echo "  ⚠️  Possible broken link: $link (line: $(echo $line | cut -d: -f1))"
                fi
            fi
        done <<< "$broken_links"
    fi
    return 0
}

# Check key documentation files
check_file "docs/concepts/document-loading.md"
check_file "docs/concepts/text-splitting.md"
check_file "docs/concepts/rag.md"
check_file "docs/getting-started/03-document-ingestion.md"
check_file "docs/cookbook/document-ingestion-recipes.md"
check_file "pkg/documentloaders/README.md"
check_file "pkg/textsplitters/README.md"

echo ""
echo "✅ Documentation link validation complete (basic checks)"
