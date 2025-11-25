#!/bin/bash
# Validation script for coverage calculation (Contract 1)
# Validates that coverage is calculated correctly and reported accurately

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

COVERAGE_FILE="${1:-coverage.unit.out}"
THRESHOLD="${2:-80}"

echo -e "${GREEN}Validating coverage calculation...${NC}"
echo "Coverage file: $COVERAGE_FILE"
echo "Threshold: ${THRESHOLD}%"
echo ""

# Validation Rule 1: Coverage file MUST exist
if [ ! -f "$COVERAGE_FILE" ]; then
  echo -e "${RED}❌ ERROR: Coverage file not generated${NC}"
  echo "Coverage file '$COVERAGE_FILE' does not exist"
  exit 1
fi
echo -e "${GREEN}✅ Coverage file exists${NC}"

# Validation Rule 2: Coverage file MUST be valid Go coverage format
if ! go tool cover -func="$COVERAGE_FILE" > /dev/null 2>&1; then
  echo -e "${RED}❌ ERROR: Coverage file format invalid${NC}"
  echo "Coverage file '$COVERAGE_FILE' is not a valid Go coverage format"
  exit 1
fi
echo -e "${GREEN}✅ Coverage file format is valid${NC}"

# Validation Rule 3: Total coverage MUST be calculated
coverage=$(go tool cover -func="$COVERAGE_FILE" 2>/dev/null | grep total | awk '{print $3}' || echo "")
if [ -z "$coverage" ]; then
  echo -e "${RED}❌ ERROR: Failed to calculate coverage${NC}"
  echo "Could not extract coverage percentage from file"
  exit 1
fi
echo -e "${GREEN}✅ Coverage calculated: ${coverage}${NC}"

# Validation Rule 4: Coverage percentage MUST be >= 0 and <= 100
pct=$(echo "$coverage" | sed 's/%//' | tr -d '\n\r' | tr -d ' ')
if ! [[ "$pct" =~ ^[0-9.]+$ ]]; then
  echo -e "${RED}❌ ERROR: Invalid coverage percentage format: ${coverage}${NC}"
  exit 1
fi

if awk "BEGIN {exit !($pct < 0 || $pct > 100)}"; then
  echo -e "${RED}❌ ERROR: Coverage percentage out of range: ${coverage}${NC}"
  echo "Coverage must be between 0% and 100%"
  exit 1
fi
echo -e "${GREEN}✅ Coverage percentage is valid: ${coverage}${NC}"

# Validation Rule 5: Coverage MUST be reported (not 0% when tests pass)
# This is a logical check - if coverage is 0%, it might indicate a problem
if [ "$pct" = "0" ] || [ "$pct" = "0.0" ]; then
  echo -e "${YELLOW}⚠️  WARNING: Coverage is 0%${NC}"
  echo "This might indicate:"
  echo "  - No tests were run"
  echo "  - Tests failed before coverage was generated"
  echo "  - Coverage file is empty or invalid"
  echo ""
  echo "Checking test output..."
  if [ -f test-output.txt ]; then
    if grep -q "^ok  " test-output.txt; then
      echo -e "${RED}❌ ERROR: Tests passed but coverage is 0% - this is a bug${NC}"
      exit 1
    fi
  fi
fi

# Validation Rule 6: Coverage threshold MUST use warnings (not errors) and not exit with error
# This validation script checks the workflow file, not the coverage itself
WORKFLOW_FILE="${3:-.github/workflows/ci-cd.yml}"
if [ -f "$WORKFLOW_FILE" ]; then
  echo ""
  echo -e "${GREEN}Validating coverage threshold configuration in workflow...${NC}"
  
  # Check that coverage threshold uses ::warning:: not ::error::
  if grep -q "coverage.*80\|Coverage.*80" "$WORKFLOW_FILE" -A 5 | grep -q "::warning::"; then
    echo -e "${GREEN}✅ Coverage threshold uses ::warning:: (advisory)${NC}"
  else
    echo -e "${YELLOW}⚠️  WARNING: Coverage threshold may not use ::warning::${NC}"
  fi
  
  # Check that coverage threshold doesn't exit with error
  if grep -q "coverage.*80\|Coverage.*80" "$WORKFLOW_FILE" -A 10 | grep -q "exit 1"; then
    echo -e "${RED}❌ ERROR: Coverage threshold check exits with error code${NC}"
    echo "Coverage threshold should be advisory (warnings only, no exit 1)"
    exit 1
  else
    echo -e "${GREEN}✅ Coverage threshold does not exit with error code${NC}"
  fi
fi

# For this script, we still check if coverage meets threshold (for local validation)
if awk "BEGIN {exit !($pct < $THRESHOLD)}"; then
  echo -e "${YELLOW}⚠️  WARNING: Coverage ${coverage} is below ${THRESHOLD}% threshold${NC}"
  echo "Current coverage: ${coverage}"
  echo "Required coverage: ${THRESHOLD}%"
  echo "Note: This is an advisory check and does not block merge"
else
  echo -e "${GREEN}✅ Coverage ${coverage} meets ${THRESHOLD}% threshold${NC}"
fi

# Show coverage breakdown
echo ""
echo -e "${GREEN}Coverage breakdown:${NC}"
go tool cover -func="$COVERAGE_FILE" | tail -20

echo ""
echo -e "${GREEN}✅ All coverage validation checks passed!${NC}"

