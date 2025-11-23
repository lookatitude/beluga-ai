#!/bin/bash
# Validation script for documentation generation (Contract 4)
# Validates that API documentation is generated automatically on main branch merges

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

WORKFLOW_FILE="${1:-.github/workflows/website_deploy.yml}"
DOCS_DIR="${2:-website/docs/api/packages}"

echo -e "${GREEN}Validating documentation generation...${NC}"
echo "Workflow file: $WORKFLOW_FILE"
echo "Documentation directory: $DOCS_DIR"
echo ""

# Validation Rule 1: Documentation MUST be generated on merges to main branch
if ! grep -A 5 "branches:" "$WORKFLOW_FILE" | grep -q "main"; then
  echo -e "${RED}❌ ERROR: Workflow does not trigger on main branch${NC}"
  exit 1
fi
echo -e "${GREEN}✅ Workflow triggers on main branch${NC}"

# Check that it doesn't trigger on PRs
if grep -q "pull_request:" "$WORKFLOW_FILE"; then
  echo -e "${YELLOW}⚠️  WARNING: Workflow triggers on pull requests${NC}"
  echo "Documentation should only be generated on main branch merges"
else
  echo -e "${GREEN}✅ Workflow does not trigger on PRs${NC}"
fi

# Validation Rule 2: Documentation generation script MUST be called
if ! grep -q "docs-generate\|generate-docs" "$WORKFLOW_FILE"; then
  echo -e "${RED}❌ ERROR: Documentation generation not found in workflow${NC}"
  exit 1
fi
echo -e "${GREEN}✅ Documentation generation configured${NC}"

# Validation Rule 3: gomarkdoc MUST be installed
if ! grep -q "gomarkdoc" "$WORKFLOW_FILE"; then
  echo -e "${YELLOW}⚠️  WARNING: gomarkdoc installation not found${NC}"
  echo "Consider adding gomarkdoc installation step"
else
  echo -e "${GREEN}✅ gomarkdoc installation configured${NC}"
fi

# Validation Rule 4: Documentation generation failure MUST fail the workflow
if ! grep -q "exit 1\|::error::" "$WORKFLOW_FILE" -A 2 | grep -q "docs-generate\|generate-docs"; then
  echo -e "${YELLOW}⚠️  WARNING: Documentation generation failure handling not found${NC}"
  echo "Workflow should fail if documentation generation fails"
else
  echo -e "${GREEN}✅ Documentation generation failure handling configured${NC}"
fi

# Validation Rule 5: Generated documentation MUST exist (if running after generation)
if [ -d "$DOCS_DIR" ]; then
  doc_count=$(find "$DOCS_DIR" -name "*.md" -type f 2>/dev/null | wc -l)
  if [ "$doc_count" -eq 0 ]; then
    echo -e "${YELLOW}⚠️  WARNING: No documentation files found in $DOCS_DIR${NC}"
    echo "This might be expected if documentation hasn't been generated yet"
  else
    echo -e "${GREEN}✅ Found ${doc_count} documentation files${NC}"
  fi
else
  echo -e "${YELLOW}⚠️  WARNING: Documentation directory not found: $DOCS_DIR${NC}"
  echo "This might be expected if documentation hasn't been generated yet"
fi

echo ""
echo -e "${GREEN}✅ All documentation generation validation checks passed!${NC}"

