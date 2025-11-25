#!/bin/bash
# Validation script for manual workflow triggers (Contract C007)
# Validates that all workflows support workflow_dispatch with proper input parameters and conditional logic

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

WORKFLOWS_DIR="${1:-.github/workflows}"

echo -e "${GREEN}Validating manual trigger configuration...${NC}"
echo "Workflows directory: $WORKFLOWS_DIR"
echo ""

failed=0

# Check each workflow file
for workflow in "$WORKFLOWS_DIR"/*.yml "$WORKFLOWS_DIR"/*.yaml; do
  if [ ! -f "$workflow" ]; then
    continue
  fi
  
  workflow_name=$(basename "$workflow")
  echo "Validating $workflow_name..."
  
  # Validation Rule 1: All workflows MUST support workflow_dispatch
  if ! grep -q "workflow_dispatch:" "$workflow"; then
    echo -e "${RED}❌ ERROR: $workflow_name does not support manual triggers (workflow_dispatch)${NC}"
    failed=1
    continue
  fi
  echo -e "${GREEN}✅ $workflow_name supports workflow_dispatch${NC}"
  
  # Validation Rule 2: workflow_dispatch MUST have input parameters (for step control)
  if ! grep -q "inputs:" "$workflow" -A 10; then
    echo -e "${YELLOW}⚠️  WARNING: $workflow_name has workflow_dispatch but no inputs defined${NC}"
    echo "Consider adding input parameters for granular step control"
  else
    echo -e "${GREEN}✅ $workflow_name has input parameters${NC}"
    
    # Check that inputs are properly structured
    input_count=$(grep -c "description:" "$workflow" || echo "0")
    if [ "$input_count" -gt 0 ]; then
      echo -e "${GREEN}✅ $workflow_name has $input_count input parameter(s)${NC}"
    fi
  fi
  
  # Validation Rule 3: Jobs MUST use conditional logic based on inputs
  job_count=$(grep -c "^  [a-zA-Z_-]*:" "$workflow" || echo "0")
  conditional_count=$(grep -c "if:.*github.event.inputs\|if:.*github.event_name.*workflow_dispatch" "$workflow" || echo "0")
  
  if [ "$job_count" -gt 0 ] && [ "$conditional_count" -eq 0 ]; then
    echo -e "${YELLOW}⚠️  WARNING: $workflow_name has jobs but no conditional logic based on inputs${NC}"
    echo "Jobs should use conditional 'if:' statements to respect input parameters"
  elif [ "$conditional_count" -gt 0 ]; then
    echo -e "${GREEN}✅ $workflow_name uses conditional logic ($conditional_count conditions)${NC}"
  fi
  
  # Validation Rule 4: Input parameters MUST have proper structure
  # Check for required fields: description, required, default, type
  if grep -q "inputs:" "$workflow"; then
    input_section=$(grep -A 50 "inputs:" "$workflow" | head -20)
    
    # Check for description
    if echo "$input_section" | grep -q "description:"; then
      echo -e "${GREEN}✅ Input parameters have descriptions${NC}"
    else
      echo -e "${YELLOW}⚠️  WARNING: Some input parameters may be missing descriptions${NC}"
    fi
    
    # Check for type
    if echo "$input_section" | grep -q "type: boolean\|type: string"; then
      echo -e "${GREEN}✅ Input parameters have types defined${NC}"
    else
      echo -e "${YELLOW}⚠️  WARNING: Some input parameters may be missing type definitions${NC}"
    fi
  fi
  
  echo ""
done

if [ "$failed" -eq 1 ]; then
  echo -e "${RED}❌ Some workflows have manual trigger validation errors${NC}"
  exit 1
fi

echo -e "${GREEN}✅ All workflows support manual triggers correctly!${NC}"



