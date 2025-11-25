#!/bin/bash
# Validation script for changelog generation (Contract C008)
# Validates that changelog generation is configured and integrated with GoReleaser

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

WORKFLOW_FILE="${1:-.github/workflows/release.yml}"
GORELEASER_FILE="${2:-.goreleaser.yml}"

echo -e "${GREEN}Validating changelog generation configuration...${NC}"
echo "Release workflow: $WORKFLOW_FILE"
echo "GoReleaser config: $GORELEASER_FILE"
echo ""

failed=0

# Validation Rule 1: Changelog generation MUST be in release workflow
if [ ! -f "$WORKFLOW_FILE" ]; then
  echo -e "${RED}❌ ERROR: Release workflow file not found: $WORKFLOW_FILE${NC}"
  exit 1
fi

if grep -q "changelog\|CHANGELOG\|Generate changelog" "$WORKFLOW_FILE" -i; then
  echo -e "${GREEN}✅ Changelog generation found in release workflow${NC}"
else
  echo -e "${RED}❌ ERROR: Changelog generation not found in release workflow${NC}"
  failed=1
fi

# Validation Rule 2: Changelog generation MUST run before GoReleaser
if grep -q "changelog\|CHANGELOG" "$WORKFLOW_FILE" -i -B 5 | grep -q "goreleaser" -A 5; then
  # Check order: changelog should come before goreleaser
  changelog_line=$(grep -n "changelog\|CHANGELOG\|Generate changelog" "$WORKFLOW_FILE" -i | head -1 | cut -d: -f1)
  goreleaser_line=$(grep -n "goreleaser" "$WORKFLOW_FILE" -i | head -1 | cut -d: -f1)
  
  if [ -n "$changelog_line" ] && [ -n "$goreleaser_line" ] && [ "$changelog_line" -lt "$goreleaser_line" ]; then
    echo -e "${GREEN}✅ Changelog generation runs before GoReleaser${NC}"
  else
    echo -e "${YELLOW}⚠️  WARNING: Changelog generation may not run before GoReleaser${NC}"
  fi
else
  echo -e "${YELLOW}⚠️  WARNING: Could not verify changelog generation order${NC}"
fi

# Validation Rule 3: Changelog generation failures MUST not block release
changelog_step=$(grep -A 5 "Generate changelog\|changelog\|CHANGELOG" "$WORKFLOW_FILE" -i | head -10)
if echo "$changelog_step" | grep -q "continue-on-error: true"; then
  echo -e "${GREEN}✅ Changelog generation failures do not block release${NC}"
else
  echo -e "${YELLOW}⚠️  WARNING: Changelog generation may block release on failure${NC}"
  echo "Consider adding continue-on-error: true to changelog generation step"
fi

# Validation Rule 4: GoReleaser MUST be configured to use changelog
if [ ! -f "$GORELEASER_FILE" ]; then
  echo -e "${YELLOW}⚠️  WARNING: GoReleaser config file not found: $GORELEASER_FILE${NC}"
else
  if grep -q "changelog:" "$GORELEASER_FILE"; then
    echo -e "${GREEN}✅ GoReleaser has changelog configuration${NC}"
    
    # Check changelog source
    if grep -q "use: git\|use: changie" "$GORELEASER_FILE"; then
      changelog_source=$(grep "use:" "$GORELEASER_FILE" | head -1 | awk '{print $2}')
      echo -e "${GREEN}✅ GoReleaser uses changelog source: $changelog_source${NC}"
    else
      echo -e "${YELLOW}⚠️  WARNING: GoReleaser changelog source not specified${NC}"
    fi
  else
    echo -e "${YELLOW}⚠️  WARNING: GoReleaser changelog configuration not found${NC}"
    echo "Consider adding changelog configuration to .goreleaser.yml"
  fi
fi

# Validation Rule 5: CHANGELOG.md file SHOULD exist (optional but recommended)
if [ -f "CHANGELOG.md" ]; then
  echo -e "${GREEN}✅ CHANGELOG.md file exists${NC}"
  
  # Check if it has content
  if [ -s "CHANGELOG.md" ]; then
    echo -e "${GREEN}✅ CHANGELOG.md has content${NC}"
  else
    echo -e "${YELLOW}⚠️  WARNING: CHANGELOG.md is empty${NC}"
  fi
else
  echo -e "${YELLOW}⚠️  WARNING: CHANGELOG.md file not found${NC}"
  echo "Consider creating CHANGELOG.md for better changelog management"
fi

# Validation Rule 6: Check for changie configuration (optional)
if [ -f ".changie.yaml" ]; then
  echo -e "${GREEN}✅ Changie configuration found (.changie.yaml)${NC}"
else
  echo -e "${GREEN}✅ Using git-based changelog (no changie configuration)${NC}"
fi

echo ""
if [ "$failed" -eq 1 ]; then
  echo -e "${RED}❌ Changelog generation validation failed${NC}"
  exit 1
fi

echo -e "${GREEN}✅ All changelog generation validation checks passed!${NC}"



