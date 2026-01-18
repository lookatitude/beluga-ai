#!/bin/bash
# validate-test-patterns.sh
# Validates that all test files follow established testing patterns

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$PROJECT_ROOT"

REPORT_FILE="${PROJECT_ROOT}/docs/pattern-validation-report.md"
ERRORS=0
WARNINGS=0

echo "# Test Pattern Validation Report" > "$REPORT_FILE"
echo "" >> "$REPORT_FILE"
echo "**Generated**: $(date -Iseconds)" >> "$REPORT_FILE"
echo "" >> "$REPORT_FILE"

# Check for test_utils.go files
echo "## test_utils.go Files" >> "$REPORT_FILE"
echo "" >> "$REPORT_FILE"
MISSING_TEST_UTILS=0
for pkg_dir in pkg/*/; do
    pkg_name=$(basename "$pkg_dir")
    if [ ! -f "${pkg_dir}test_utils.go" ]; then
        echo "- ❌ Missing: ${pkg_dir}test_utils.go" >> "$REPORT_FILE"
        ((MISSING_TEST_UTILS++))
        ((ERRORS++))
    else
        echo "- ✅ Found: ${pkg_dir}test_utils.go" >> "$REPORT_FILE"
    fi
done
echo "" >> "$REPORT_FILE"
echo "**Summary**: $MISSING_TEST_UTILS packages missing test_utils.go" >> "$REPORT_FILE"
echo "" >> "$REPORT_FILE"

# Check for advanced_test.go files
echo "## advanced_test.go Files" >> "$REPORT_FILE"
echo "" >> "$REPORT_FILE"
MISSING_ADVANCED_TEST=0
for pkg_dir in pkg/*/; do
    pkg_name=$(basename "$pkg_dir")
    if [ ! -f "${pkg_dir}advanced_test.go" ]; then
        echo "- ❌ Missing: ${pkg_dir}advanced_test.go" >> "$REPORT_FILE"
        ((MISSING_ADVANCED_TEST++))
        ((ERRORS++))
    else
        echo "- ✅ Found: ${pkg_dir}advanced_test.go" >> "$REPORT_FILE"
    fi
done
echo "" >> "$REPORT_FILE"
echo "**Summary**: $MISSING_ADVANCED_TEST packages missing advanced_test.go" >> "$REPORT_FILE"
echo "" >> "$REPORT_FILE"

# Check for AdvancedMock pattern in test_utils.go
echo "## AdvancedMock Pattern" >> "$REPORT_FILE"
echo "" >> "$REPORT_FILE"
MISSING_MOCK=0
for test_utils in pkg/*/test_utils.go; do
    if [ -f "$test_utils" ]; then
        if ! grep -q "AdvancedMock" "$test_utils"; then
            echo "- ⚠️  Missing AdvancedMock: $test_utils" >> "$REPORT_FILE"
            ((MISSING_MOCK++))
            ((WARNINGS++))
        else
            echo "- ✅ Has AdvancedMock: $test_utils" >> "$REPORT_FILE"
        fi
    fi
done
echo "" >> "$REPORT_FILE"
echo "**Summary**: $MISSING_MOCK files missing AdvancedMock pattern" >> "$REPORT_FILE"
echo "" >> "$REPORT_FILE"

# Check for table-driven tests in advanced_test.go
echo "## Table-Driven Tests" >> "$REPORT_FILE"
echo "" >> "$REPORT_FILE"
MISSING_TABLE_TESTS=0
for advanced_test in pkg/*/advanced_test.go; do
    if [ -f "$advanced_test" ]; then
        if ! grep -q "tests := \[\]struct" "$advanced_test"; then
            echo "- ⚠️  Missing table-driven tests: $advanced_test" >> "$REPORT_FILE"
            ((MISSING_TABLE_TESTS++))
            ((WARNINGS++))
        else
            echo "- ✅ Has table-driven tests: $advanced_test" >> "$REPORT_FILE"
        fi
    fi
done
echo "" >> "$REPORT_FILE"
echo "**Summary**: $MISSING_TABLE_TESTS files missing table-driven tests" >> "$REPORT_FILE"
echo "" >> "$REPORT_FILE"

# Final summary
echo "## Summary" >> "$REPORT_FILE"
echo "" >> "$REPORT_FILE"
echo "- **Errors**: $ERRORS" >> "$REPORT_FILE"
echo "- **Warnings**: $WARNINGS" >> "$REPORT_FILE"
echo "- **Status**: $([ $ERRORS -eq 0 ] && echo "✅ PASS" || echo "❌ FAIL")" >> "$REPORT_FILE"

if [ $ERRORS -eq 0 ]; then
    echo "✅ Pattern validation passed"
    exit 0
else
    echo "❌ Pattern validation failed with $ERRORS errors"
    exit 1
fi
