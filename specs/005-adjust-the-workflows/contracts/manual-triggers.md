# Contract: Manual Workflow Triggers

**Feature**: 005-adjust-the-workflows  
**Date**: 2025-01-27  
**Contract ID**: C007

## Purpose
Ensure all workflow steps can be triggered manually via `workflow_dispatch` event with input parameters to control individual step execution.

## Input
- Manual workflow trigger via GitHub UI, GitHub CLI, or REST API
- Workflow input parameters (boolean flags for each step)

## Validation Rules

### Rule 1: All Workflows Must Support Manual Triggers
- All workflow files MUST have `workflow_dispatch` in the `on:` section
- Manual triggers MUST be accessible via GitHub UI, GitHub CLI (`gh workflow run`), and REST API
- Manual triggers MUST work independently of automatic triggers (push, pull_request, etc.)

### Rule 2: Input Parameters Must Control Step Execution
- Each major workflow step MUST have a corresponding boolean input parameter
- Input parameters MUST be named descriptively (e.g., `run_policy`, `run_lint`, `run_security`, `run_unit_tests`, `run_integration_tests`, `run_build`, `run_release`)
- Input parameters MUST default to `false` for safety
- Steps MUST use conditional `if:` statements based on input values

### Rule 3: Step Execution Logic
- Steps MUST execute if corresponding input is `true` OR if workflow is triggered automatically (not via workflow_dispatch)
- When manually triggered, only steps with `true` inputs MUST execute
- When automatically triggered, all steps MUST execute (unless explicitly disabled)
- Default behavior (no inputs specified) MUST execute all steps

### Rule 4: Input Parameter Structure
- Input parameters MUST be defined under `workflow_dispatch.inputs`
- Each input MUST have: `description`, `required: false`, `default: 'false'`, `type: boolean`
- Input descriptions MUST clearly indicate what step they control

## Success Criteria
- All workflows support manual triggering via workflow_dispatch
- Individual steps can be selectively executed via input parameters
- Manual triggers work via GitHub UI, GitHub CLI, and REST API
- Automatic triggers continue to work as before
- Input parameters are clearly documented and intuitive

## Failure Modes
- Workflow missing workflow_dispatch → Error: "Workflow must support manual triggers"
- Step missing conditional logic → Error: "Step must check input parameter before executing"
- Input parameter missing for step → Error: "Step must have corresponding input parameter"
- Manual trigger not accessible → Error: "Manual trigger must work via UI, CLI, and API"

## Test Validation
```bash
# Validate workflow_dispatch exists
grep -q "workflow_dispatch:" .github/workflows/ci-cd.yml || echo "ERROR: workflow_dispatch not found"

# Validate input parameters exist
grep -q "inputs:" .github/workflows/ci-cd.yml || echo "ERROR: inputs not defined"

# Validate step conditionals
grep -A 3 "name:.*Policy" .github/workflows/ci-cd.yml | grep -q "if:.*run_policy\|if:.*inputs.run_policy" || echo "ERROR: Policy step missing conditional"

# Test manual trigger via gh CLI
gh workflow run "CI/CD" --ref main -f run_lint=true || echo "ERROR: Manual trigger failed"

# Validate GitHub UI access (manual check)
# Navigate to Actions tab, verify "Run workflow" button exists
```

## Implementation Notes
- Use `workflow_dispatch` event in all workflow `on:` sections
- Define inputs like:
  ```yaml
  on:
    workflow_dispatch:
      inputs:
        run_policy:
          description: 'Run policy checks'
          required: false
          default: 'false'
          type: boolean
  ```
- Use conditionals like: `if: ${{ github.event.inputs.run_policy == 'true' || github.event_name != 'workflow_dispatch' }}`
- Support both individual step execution and full pipeline execution
- Document input parameters in workflow comments or README

