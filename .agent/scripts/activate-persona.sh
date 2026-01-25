#!/bin/bash
#
# Activate a Beluga AI agent persona
#
# Usage: ./activate-persona.sh <persona-name>
#
# Available personas:
#   - backend-developer
#   - architect
#   - researcher
#   - qa
#   - documentation-writer
#

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
PERSONAS_DIR="$PROJECT_ROOT/.agent/personas"
CURSOR_RULES_DIR="$PROJECT_ROOT/.cursor/rules"
ACTIVE_RULE_FILE="$CURSOR_RULES_DIR/active-persona.mdc"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

usage() {
    echo "Usage: $0 <persona-name>"
    echo ""
    echo "Available personas:"
    echo "  backend-developer    - Implementation focus (build, test, commit)"
    echo "  architect            - Design focus (read-only)"
    echo "  researcher           - Analysis focus (read-only)"
    echo "  qa                   - Testing focus (test, lint, security)"
    echo "  documentation-writer - Documentation focus (docs only)"
    echo ""
    echo "Examples:"
    echo "  $0 backend-developer"
    echo "  $0 qa"
    exit 1
}

list_personas() {
    echo -e "${BLUE}Available personas:${NC}"
    for dir in "$PERSONAS_DIR"/*/; do
        if [ -d "$dir" ]; then
            name=$(basename "$dir")
            if [ -f "$dir/PERSONA.md" ]; then
                # Extract description from YAML frontmatter
                desc=$(grep -A1 "^description:" "$dir/PERSONA.md" 2>/dev/null | tail -1 | sed 's/description: //')
                echo -e "  ${GREEN}$name${NC} - $desc"
            fi
        fi
    done
}

deactivate_persona() {
    if [ -L "$ACTIVE_RULE_FILE" ] || [ -f "$ACTIVE_RULE_FILE" ]; then
        rm -f "$ACTIVE_RULE_FILE"
        echo -e "${YELLOW}Deactivated previous persona${NC}"
    fi
}

activate_persona() {
    local persona=$1
    local persona_dir="$PERSONAS_DIR/$persona"
    local rule_file="$persona_dir/rules/main.mdc"

    # Check if persona exists
    if [ ! -d "$persona_dir" ]; then
        echo -e "${RED}Error: Persona '$persona' not found${NC}"
        echo ""
        list_personas
        exit 1
    fi

    # Check if rule file exists
    if [ ! -f "$rule_file" ]; then
        echo -e "${RED}Error: Rule file not found: $rule_file${NC}"
        exit 1
    fi

    # Ensure cursor rules directory exists
    mkdir -p "$CURSOR_RULES_DIR"

    # Deactivate any existing persona
    deactivate_persona

    # Create symlink to persona rules
    # Use relative path for portability
    local rel_path="../../.agent/personas/$persona/rules/main.mdc"
    ln -sf "$rel_path" "$ACTIVE_RULE_FILE"

    echo -e "${GREEN}✓ Activated persona: $persona${NC}"
    echo ""
    echo -e "${BLUE}Persona details:${NC}"
    echo "  Location: $persona_dir"
    echo "  Rules: $rule_file"
    echo ""

    # Show persona summary
    if [ -f "$persona_dir/PERSONA.md" ]; then
        echo -e "${BLUE}Skills:${NC}"
        grep -A10 "^skills:" "$persona_dir/PERSONA.md" 2>/dev/null | grep "^  -" | sed 's/  - /  • /' || true
        echo ""

        echo -e "${BLUE}Permissions:${NC}"
        grep -A10 "^permissions:" "$persona_dir/PERSONA.md" 2>/dev/null | grep "^  " | head -5 || true
    fi
}

show_current() {
    if [ -L "$ACTIVE_RULE_FILE" ]; then
        target=$(readlink "$ACTIVE_RULE_FILE")
        persona=$(echo "$target" | sed 's|.*/personas/||' | sed 's|/rules/main.mdc||')
        echo -e "${BLUE}Currently active persona: ${GREEN}$persona${NC}"
    else
        echo -e "${YELLOW}No persona currently active${NC}"
    fi
}

# Main
if [ $# -eq 0 ]; then
    show_current
    echo ""
    usage
fi

case "$1" in
    --help|-h)
        usage
        ;;
    --list|-l)
        list_personas
        ;;
    --current|-c)
        show_current
        ;;
    --deactivate|-d)
        deactivate_persona
        echo -e "${GREEN}✓ Persona deactivated${NC}"
        ;;
    *)
        activate_persona "$1"
        ;;
esac
