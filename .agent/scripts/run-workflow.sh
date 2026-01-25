#!/bin/bash
# Workflow Orchestration Runner
#
# Executes multi-persona workflows defined in .agent/orchestration/workflows/
#
# Usage:
#   ./run-workflow.sh <workflow-name> [options]
#
# Options:
#   --dry-run    Show steps without executing
#   --verbose    Enable verbose output
#   --input      Input file for workflow (overrides default)
#   --help       Show this help message
#
# Examples:
#   ./run-workflow.sh verify-architecture
#   ./run-workflow.sh verify-architecture --dry-run
#   ./run-workflow.sh fix-and-validate --input issues.md

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
AGENT_DIR="$(dirname "$SCRIPT_DIR")"
WORKFLOWS_DIR="$AGENT_DIR/orchestration/workflows"
REPORTS_DIR="$AGENT_DIR/orchestration/reports"

# Default options
DRY_RUN=false
VERBOSE=false
INPUT_FILE=""
WORKFLOW_NAME=""

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --dry-run)
            DRY_RUN=true
            shift
            ;;
        --verbose)
            VERBOSE=true
            shift
            ;;
        --input)
            INPUT_FILE="$2"
            shift 2
            ;;
        --help|-h)
            echo "Workflow Orchestration Runner"
            echo ""
            echo "Usage: $0 <workflow-name> [options]"
            echo ""
            echo "Options:"
            echo "  --dry-run    Show steps without executing"
            echo "  --verbose    Enable verbose output"
            echo "  --input      Input file for workflow"
            echo "  --help       Show this help message"
            echo ""
            echo "Available workflows:"
            if [[ -d "$WORKFLOWS_DIR" ]]; then
                for f in "$WORKFLOWS_DIR"/*.yaml; do
                    if [[ -f "$f" ]]; then
                        name=$(basename "$f" .yaml)
                        desc=$(grep -m1 "^description:" "$f" 2>/dev/null | sed 's/description: //' || echo "No description")
                        printf "  %-25s %s\n" "$name" "$desc"
                    fi
                done
            else
                echo "  (no workflows found)"
            fi
            exit 0
            ;;
        -*)
            echo -e "${RED}Error: Unknown option $1${NC}"
            exit 1
            ;;
        *)
            if [[ -z "$WORKFLOW_NAME" ]]; then
                WORKFLOW_NAME="$1"
            else
                echo -e "${RED}Error: Unexpected argument $1${NC}"
                exit 1
            fi
            shift
            ;;
    esac
done

# Validate workflow name
if [[ -z "$WORKFLOW_NAME" ]]; then
    echo -e "${RED}Error: Workflow name required${NC}"
    echo "Usage: $0 <workflow-name> [options]"
    echo "Run '$0 --help' for available workflows"
    exit 1
fi

WORKFLOW_FILE="$WORKFLOWS_DIR/$WORKFLOW_NAME.yaml"
if [[ ! -f "$WORKFLOW_FILE" ]]; then
    echo -e "${RED}Error: Workflow '$WORKFLOW_NAME' not found${NC}"
    echo "Available workflows:"
    for f in "$WORKFLOWS_DIR"/*.yaml; do
        if [[ -f "$f" ]]; then
            echo "  - $(basename "$f" .yaml)"
        fi
    done
    exit 1
fi

# Create timestamped report directory
TIMESTAMP=$(date +%Y-%m-%dT%H-%M-%S)
RUN_DIR="$REPORTS_DIR/$WORKFLOW_NAME-$TIMESTAMP"

log() {
    echo -e "${BLUE}[$(date +%H:%M:%S)]${NC} $1"
}

log_step() {
    echo -e "${CYAN}[STEP]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

verbose() {
    if [[ "$VERBOSE" == "true" ]]; then
        echo -e "${YELLOW}[VERBOSE]${NC} $1"
    fi
}

# Parse YAML (basic parser for our workflow format)
parse_workflow() {
    local file="$1"

    # Extract workflow name and description
    WORKFLOW_DISPLAY_NAME=$(grep -m1 "^name:" "$file" | sed 's/name: //')
    WORKFLOW_DESC=$(grep -m1 "^description:" "$file" | sed 's/description: //')

    # Extract steps (simplified - in production use yq or similar)
    STEPS=()
    local in_steps=false
    local current_step=""

    while IFS= read -r line; do
        if [[ "$line" =~ ^steps: ]]; then
            in_steps=true
            continue
        fi

        if [[ "$in_steps" == "true" ]]; then
            if [[ "$line" =~ ^[[:space:]]*-[[:space:]]*name:[[:space:]]*(.*) ]]; then
                if [[ -n "$current_step" ]]; then
                    STEPS+=("$current_step")
                fi
                current_step="${BASH_REMATCH[1]}"
            fi
        fi
    done < "$file"

    if [[ -n "$current_step" ]]; then
        STEPS+=("$current_step")
    fi
}

# Get step details
get_step_detail() {
    local file="$1"
    local step_name="$2"
    local field="$3"

    awk -v step="$step_name" -v field="$field" '
        /^[[:space:]]*-[[:space:]]*name:[[:space:]]*/ {
            current_step = $NF
            gsub(/[[:space:]]*$/, "", current_step)
        }
        current_step == step && $0 ~ "^[[:space:]]*" field ":" {
            sub(/^[[:space:]]*/ field ":[[:space:]]*/, "")
            print
            exit
        }
    ' "$file"
}

# Execute a workflow step
execute_step() {
    local step_name="$1"
    local persona="$2"
    local skill="$3"

    log_step "Executing: $step_name"
    verbose "  Persona: $persona"
    verbose "  Skill: $skill"

    if [[ "$DRY_RUN" == "true" ]]; then
        echo "  [DRY RUN] Would activate persona: $persona"
        echo "  [DRY RUN] Would execute skill: $skill"
        return 0
    fi

    # Activate persona
    if [[ -x "$SCRIPT_DIR/activate-persona.sh" ]]; then
        "$SCRIPT_DIR/activate-persona.sh" "$persona" >/dev/null 2>&1 || {
            log_warning "Could not activate persona: $persona"
        }
    fi

    log_success "Step completed: $step_name"
}

# Main execution
main() {
    log "Starting workflow: $WORKFLOW_NAME"

    if [[ "$DRY_RUN" == "true" ]]; then
        log_warning "DRY RUN MODE - No changes will be made"
    fi

    # Parse workflow
    parse_workflow "$WORKFLOW_FILE"

    echo ""
    echo -e "${CYAN}Workflow:${NC} $WORKFLOW_DISPLAY_NAME"
    echo -e "${CYAN}Description:${NC} $WORKFLOW_DESC"
    echo -e "${CYAN}Steps:${NC} ${#STEPS[@]}"
    echo ""

    if [[ "$DRY_RUN" != "true" ]]; then
        # Create report directory
        mkdir -p "$RUN_DIR"
        log "Reports will be saved to: $RUN_DIR"
        echo ""
    fi

    # Execute each step
    local step_num=0
    for step_name in "${STEPS[@]}"; do
        ((step_num++))
        echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
        echo -e "${CYAN}Step $step_num/${#STEPS[@]}:${NC} $step_name"
        echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

        # Get step details
        local persona=$(get_step_detail "$WORKFLOW_FILE" "$step_name" "persona")
        local skill=$(get_step_detail "$WORKFLOW_FILE" "$step_name" "skill")
        local output=$(get_step_detail "$WORKFLOW_FILE" "$step_name" "output")

        verbose "Persona: $persona"
        verbose "Skill: $skill"
        verbose "Output: $output"

        # Execute step
        execute_step "$step_name" "$persona" "$skill"

        echo ""
    done

    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    log_success "Workflow completed: $WORKFLOW_NAME"

    if [[ "$DRY_RUN" != "true" ]]; then
        echo ""
        echo -e "${GREEN}Reports saved to:${NC} $RUN_DIR"
    fi
}

main
