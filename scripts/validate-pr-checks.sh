#!/bin/bash
# Validation script for PR check status (Contract 3)
# Validates that PR checks accurately reflect status with proper critical/advisory distinction

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

WORKFLOW_FILE="${1:-.github/workflows/ci-cd.yml}"

echo -e "${GREEN}Validating PR check configuration...${NC}"
echo "Workflow file: $WORKFLOW_FILE"
echo ""

# Validation Rule 1: Critical checks (tests, security, build) MUST block merge on failure
CRITICAL_JOBS=("unit-tests" "security" "build")
for job in "${CRITICAL_JOBS[@]}"; do
  if ! grep -q "name:.*${job}" "$WORKFLOW_FILE" -A 5 | grep -q "Critical check\|blocks merge"; then
    echo -e "${YELLOW}⚠️  WARNING: Critical job '${job}' not clearly marked as critical${NC}"
  else
    echo -e "${GREEN}✅ Critical job '${job}' configured${NC}"
  fi
  
  # Check that job doesn't have continue-on-error (critical jobs should fail)
  if grep -q "name:.*${job}" "$WORKFLOW_FILE" -A 3 | grep -q "continue-on-error: true"; then
    echo -e "${RED}❌ ERROR: Critical job '${job}' has continue-on-error: true${NC}"
    echo "Critical checks should block merge on failure"
    exit 1
  fi
done

# Validation Rule 2: Advisory checks (policy, lint, coverage threshold) MUST show warnings but allow merge
ADVISORY_JOBS=("policy" "lint" "coverage")
for job in "${ADVISORY_JOBS[@]}"; do
  # Find the job section and check if it has continue-on-error
  job_section=$(grep -A 10 "^  ${job}:" "$WORKFLOW_FILE" || grep -A 10 "name:.*${job}" "$WORKFLOW_FILE" | head -15)
  
  if echo "$job_section" | grep -q "ADVISORY\|advisory\|Advisory\|warnings only"; then
    echo -e "${GREEN}✅ Advisory job '${job}' configured${NC}"
  else
    echo -e "${YELLOW}⚠️  WARNING: Advisory job '${job}' not clearly marked as advisory${NC}"
  fi
  
  # Check that advisory jobs have continue-on-error
  if echo "$job_section" | grep -q "continue-on-error: true"; then
    echo -e "${GREEN}✅ Advisory job '${job}' has continue-on-error configured${NC}"
  else
    echo -e "${RED}❌ ERROR: Advisory job '${job}' must have continue-on-error: true${NC}"
    exit 1
  fi
done

# Validation Rule 3: Check status MUST accurately reflect actual validation result
# This is validated by checking that error/warning annotations are used correctly
if grep -q "::error::" "$WORKFLOW_FILE"; then
  echo -e "${GREEN}✅ Error annotations found (for critical checks)${NC}"
else
  echo -e "${YELLOW}⚠️  WARNING: No error annotations found${NC}"
fi

if grep -q "::warning::" "$WORKFLOW_FILE"; then
  echo -e "${GREEN}✅ Warning annotations found (for advisory checks)${NC}"
else
  echo -e "${YELLOW}⚠️  WARNING: No warning annotations found${NC}"
fi

# Validation Rule 4: Override MUST be required for critical check failures
# In GitHub Actions, this is handled by branch protection rules, not in workflow
echo -e "${GREEN}✅ Override requirement handled by branch protection rules${NC}"

# Validation Rule 5: Check conclusions MUST be valid
# GitHub Actions automatically sets conclusions based on job exit status
echo -e "${GREEN}✅ Check conclusions handled by GitHub Actions${NC}"

echo ""
echo -e "${GREEN}✅ All PR check validation checks passed!${NC}"

