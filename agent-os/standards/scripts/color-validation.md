# Color Validation Output

Use consistent colors and emoji for validation script output.

```bash
#!/bin/bash
set -e

# Color definitions
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'  # No Color

# Rule 1: Check required condition
if ! grep -q "required_pattern" "$FILE"; then
    echo -e "${RED}❌ ERROR: Required pattern not found${NC}"
    exit 1
fi
echo -e "${GREEN}✅ Required pattern found${NC}"

# Rule 2: Check optional condition (warning only)
if [ some_optional_condition ]; then
    echo -e "${YELLOW}⚠️  WARNING: Optional issue detected${NC}"
    echo "Consider fixing this for better results"
else
    echo -e "${GREEN}✅ Optional check passed${NC}"
fi

# Final summary
echo ""
echo -e "${GREEN}✅ All validation checks passed!${NC}"
```

## Status Conventions

| Color | Emoji | Meaning | Exit Code |
|-------|-------|---------|-----------|
| RED | ❌ | Error - must fix | `exit 1` |
| YELLOW | ⚠️ | Warning - should fix | continue |
| GREEN | ✅ | Pass | continue |

## Guidelines
- Errors exit with code 1 immediately
- Warnings print but continue execution
- Always print final summary
- Number rules for easy reference in logs
