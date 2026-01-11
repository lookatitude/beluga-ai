#!/bin/bash
# Verify all packages match v2 structure standards

set -e

REQUIRED_FILES=("config.go" "metrics.go" "errors.go" "test_utils.go" "advanced_test.go" "README.md")
REQUIRED_DIRS=("iface" "internal")

PACKAGES=("core" "schema" "config" "llms" "chatmodels" "embeddings" "vectorstores" "memory" "retrievers" "prompts" "agents" "orchestration" "server" "monitoring" "voice")

echo "=== Verifying Package Structure ==="
echo

TOTAL_ISSUES=0

for pkg in "${PACKAGES[@]}"; do
    echo "Checking pkg/$pkg/..."
    ISSUES=0
    
    # Check required files
    for file in "${REQUIRED_FILES[@]}"; do
        # Special case: server has errors in iface, not errors.go
        if [ "$pkg" = "server" ] && [ "$file" = "errors.go" ]; then
            if [ ! -f "pkg/$pkg/iface/options.go" ]; then
                echo "  ✗ Missing: errors.go or iface/options.go (server should have errors in iface)"
                ISSUES=$((ISSUES + 1))
            fi
        elif [ ! -f "pkg/$pkg/$file" ]; then
            echo "  ✗ Missing: $file"
            ISSUES=$((ISSUES + 1))
        fi
    done
    
    # Check required directories (iface is required, internal is optional but recommended)
    if [ ! -d "pkg/$pkg/iface" ]; then
        echo "  ✗ Missing: iface/ directory"
        ISSUES=$((ISSUES + 1))
    fi
    
    # Check for providers directory (required for multi-provider packages)
    if [[ "$pkg" == "llms" || "$pkg" == "chatmodels" || "$pkg" == "embeddings" || "$pkg" == "vectorstores" || "$pkg" == "memory" || "$pkg" == "prompts" || "$pkg" == "agents" || "$pkg" == "orchestration" || "$pkg" == "server" || "$pkg" == "monitoring" || "$pkg" == "voice" ]]; then
        if [ ! -d "pkg/$pkg/providers" ]; then
            echo "  ⚠ Missing: providers/ directory (may be acceptable for some packages)"
        fi
    fi
    
    if [ $ISSUES -eq 0 ]; then
        echo "  ✓ All required files/directories present"
    else
        TOTAL_ISSUES=$((TOTAL_ISSUES + ISSUES))
    fi
    
    echo
done

echo "=== Summary ==="
if [ $TOTAL_ISSUES -eq 0 ]; then
    echo "✓ All packages comply with v2 structure standards"
    exit 0
else
    echo "✗ Found $TOTAL_ISSUES issues across packages"
    exit 1
fi
