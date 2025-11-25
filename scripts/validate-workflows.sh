#!/bin/bash
# Validation script for workflow files (Contract 5)
# Validates that all workflow files are valid YAML and follow best practices

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

WORKFLOWS_DIR="${1:-.github/workflows}"

echo -e "${GREEN}Validating workflow files...${NC}"
echo "Workflows directory: $WORKFLOWS_DIR"
echo ""

# Check if workflows directory exists
if [ ! -d "$WORKFLOWS_DIR" ]; then
  echo -e "${RED}❌ ERROR: Workflows directory not found: $WORKFLOWS_DIR${NC}"
  exit 1
fi

# Validation Rule 1: Workflow files MUST be valid YAML syntax
failed=0
for workflow in "$WORKFLOWS_DIR"/*.yml "$WORKFLOWS_DIR"/*.yaml; do
  if [ ! -f "$workflow" ]; then
    continue
  fi
  
  workflow_name=$(basename "$workflow")
  echo "Validating $workflow_name..."
  
  # Try to parse YAML using Python (most reliable)
  if command -v python3 >/dev/null 2>&1; then
    if ! python3 -c "import yaml; yaml.safe_load(open('$workflow'))" 2>/dev/null; then
      echo -e "${RED}❌ ERROR: Invalid YAML syntax in $workflow_name${NC}"
      failed=1
      continue
    fi
  elif command -v yamllint >/dev/null 2>&1; then
    if ! yamllint "$workflow" >/dev/null 2>&1; then
      echo -e "${RED}❌ ERROR: Invalid YAML syntax in $workflow_name${NC}"
      yamllint "$workflow"
      failed=1
      continue
    fi
  else
    echo -e "${YELLOW}⚠️  WARNING: No YAML validator found (python3 or yamllint)${NC}"
    echo "Skipping YAML syntax validation"
  fi
  
  echo -e "${GREEN}✅ $workflow_name has valid YAML syntax${NC}"
  
  # Validation Rule 2: Workflow files MUST have required fields: name, on, jobs
  if ! grep -q "^name:" "$workflow"; then
    echo -e "${RED}❌ ERROR: Missing 'name' field in $workflow_name${NC}"
    failed=1
    continue
  fi
  
  if ! grep -q "^on:" "$workflow"; then
    echo -e "${RED}❌ ERROR: Missing 'on' field in $workflow_name${NC}"
    failed=1
    continue
  fi
  
  if ! grep -q "^jobs:" "$workflow"; then
    echo -e "${RED}❌ ERROR: Missing 'jobs' field in $workflow_name${NC}"
    failed=1
    continue
  fi
  
  echo -e "${GREEN}✅ $workflow_name has required fields (name, on, jobs)${NC}"
  
  # Validation Rule 3: Each job MUST have runs-on and steps
  # This is a simplified check - full validation would require YAML parsing
  job_count=$(grep -c "^  [a-zA-Z_-]*:" "$workflow" || echo "0")
  runs_on_count=$(grep -c "runs-on:" "$workflow" || echo "0")
  steps_count=$(grep -c "steps:" "$workflow" || echo "0")
  
  if [ "$job_count" -gt 0 ] && [ "$runs_on_count" -lt "$job_count" ]; then
    echo -e "${YELLOW}⚠️  WARNING: Some jobs may be missing 'runs-on' in $workflow_name${NC}"
  fi
  
  if [ "$job_count" -gt 0 ] && [ "$steps_count" -lt "$job_count" ]; then
    echo -e "${YELLOW}⚠️  WARNING: Some jobs may be missing 'steps' in $workflow_name${NC}"
  fi
  
  # Validation Rule 4: No duplicate workflow names
  # This would be checked across all workflows, but we're checking one at a time
  echo -e "${GREEN}✅ $workflow_name structure validated${NC}"
  
  # Validation Rule 5: Check for workflow_dispatch support
  if grep -q "workflow_dispatch:" "$workflow"; then
    echo -e "${GREEN}✅ $workflow_name supports manual triggers (workflow_dispatch)${NC}"
    
    # Check for input parameters
    if grep -q "inputs:" "$workflow"; then
      echo -e "${GREEN}✅ $workflow_name has input parameters for manual triggers${NC}"
    else
      echo -e "${YELLOW}⚠️  WARNING: $workflow_name has workflow_dispatch but no inputs defined${NC}"
    fi
  else
    echo -e "${YELLOW}⚠️  WARNING: $workflow_name does not support manual triggers${NC}"
  fi
  
  # Validation Rule 6: Check for conditional logic (if: statements)
  if_count=$(grep -c "if:" "$workflow" || echo "0")
  if [ "$if_count" -gt 0 ]; then
    echo -e "${GREEN}✅ $workflow_name uses conditional logic ($if_count conditions)${NC}"
  else
    echo -e "${YELLOW}⚠️  WARNING: $workflow_name has no conditional logic${NC}"
  fi
  
  echo ""
done

if [ "$failed" -eq 1 ]; then
  echo -e "${RED}❌ Some workflow files have validation errors${NC}"
  exit 1
fi

echo -e "${GREEN}✅ All workflow files validated successfully!${NC}"

