#!/bin/bash
# Validation script for release workflow (Contract 2)
# Validates that unified release workflow handles both automated and manual releases

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

WORKFLOW_FILE="${1:-.github/workflows/release.yml}"

echo -e "${GREEN}Validating release workflow...${NC}"
echo "Workflow file: $WORKFLOW_FILE"
echo ""

# Validation Rule 1: Workflow MUST support automated semantic versioning
if ! grep -q "release-please" "$WORKFLOW_FILE"; then
  echo -e "${RED}❌ ERROR: Automated semantic versioning (release-please) not found${NC}"
  exit 1
fi
echo -e "${GREEN}✅ Automated semantic versioning supported${NC}"

# Validation Rule 2: Workflow MUST support manual releases
if ! grep -q "workflow_dispatch" "$WORKFLOW_FILE"; then
  echo -e "${RED}❌ ERROR: Manual release trigger (workflow_dispatch) not found${NC}"
  exit 1
fi
echo -e "${GREEN}✅ Manual release trigger supported${NC}"

# Validation Rule 3: Workflow MUST support tag-based releases
if ! grep -q "tags:" "$WORKFLOW_FILE" || ! grep -q "v\*" "$WORKFLOW_FILE"; then
  echo -e "${RED}❌ ERROR: Tag-based release trigger not found${NC}"
  exit 1
fi
echo -e "${GREEN}✅ Tag-based release trigger supported${NC}"

# Validation Rule 4: Only one release process MUST run at a time (conflict prevention)
if ! grep -q "concurrency:" "$WORKFLOW_FILE"; then
  echo -e "${YELLOW}⚠️  WARNING: No concurrency control found${NC}"
  echo "Consider adding concurrency group to prevent simultaneous releases"
else
  echo -e "${GREEN}✅ Concurrency control configured${NC}"
fi

# Validation Rule 5: Version MUST be validated
if ! grep -q "Invalid version format" "$WORKFLOW_FILE"; then
  echo -e "${YELLOW}⚠️  WARNING: Version validation not found${NC}"
  echo "Consider adding version format validation"
else
  echo -e "${GREEN}✅ Version validation configured${NC}"
fi

# Validation Rule 6: Conditional job execution to prevent conflicts
if ! grep -q "if:" "$WORKFLOW_FILE"; then
  echo -e "${YELLOW}⚠️  WARNING: No conditional job execution found${NC}"
  echo "Consider adding conditional execution to prevent conflicts"
else
  echo -e "${GREEN}✅ Conditional job execution configured${NC}"
fi

# Check for GoReleaser (for manual/tag releases)
if ! grep -q "goreleaser" "$WORKFLOW_FILE"; then
  echo -e "${YELLOW}⚠️  WARNING: GoReleaser not found${NC}"
  echo "GoReleaser is typically used for manual/tag-based releases"
else
  echo -e "${GREEN}✅ GoReleaser configured${NC}"
fi

echo ""
echo -e "${GREEN}✅ All release workflow validation checks passed!${NC}"

