# Contract: Workflow Validation and Testing

**Feature**: 005-adjust-the-workflows  
**Date**: 2025-01-27  
**Contract ID**: C006

## Purpose
Ensure workflows can be validated and tested using `gh` CLI and validation scripts.

## Input
- Workflow YAML files (`.github/workflows/*.yml`)
- `gh` CLI access to repository
- Validation scripts (`scripts/validate-*.sh`)

## Validation Rules

### Rule 1: Workflow Files Must Be Valid YAML
- All workflow files MUST be valid YAML syntax
- YAML validation MUST use Python yaml module or yamllint
- Invalid YAML MUST cause validation to fail

### Rule 2: Workflow Files Must Have Required Fields
- All workflows MUST have `name` field
- All workflows MUST have `on` field (triggers)
- All workflows MUST have `jobs` field
- Missing required fields MUST cause validation to fail

### Rule 3: Jobs Must Have Required Structure
- All jobs MUST have `runs-on` field
- All jobs MUST have `steps` field
- Jobs missing required fields MUST cause validation to fail

### Rule 4: Workflows Must Be Testable with `gh` CLI
- Workflows MUST be viewable via `gh workflow view <name> --yaml`
- Workflows MUST be runnable via `gh workflow run <name>` (for manual testing)
- Workflow definitions MUST be fetchable for validation

### Rule 5: Validation Scripts Must Check Configuration
- `scripts/validate-workflows.sh` MUST check YAML syntax and structure
- `scripts/validate-pr-checks.sh` MUST check critical vs advisory configuration
- `scripts/validate-coverage.sh` MUST check coverage calculation logic
- `scripts/validate-release.sh` MUST check release workflow configuration
- All validation scripts MUST be executable and provide clear error messages

## Success Criteria
- All workflow files pass YAML validation
- All workflows have required fields
- All jobs have required structure
- Workflows can be viewed and tested with `gh` CLI
- Validation scripts successfully validate workflow configuration
- Validation scripts provide clear error messages

## Failure Modes
- Invalid YAML syntax → Error: "Invalid YAML syntax in {workflow}"
- Missing required fields → Error: "Missing '{field}' field in {workflow}"
- Jobs missing structure → Error: "Job '{job}' missing required fields"
- `gh` CLI cannot view workflow → Error: "Workflow not accessible via gh CLI"
- Validation scripts fail → Error: "Validation script failed: {reason}"

## Test Validation
```bash
# Validate YAML syntax
python3 -c "import yaml; yaml.safe_load(open('.github/workflows/ci-cd.yml'))" || echo "ERROR: Invalid YAML"

# Validate required fields
grep -q "^name:" .github/workflows/ci-cd.yml || echo "ERROR: Missing name field"
grep -q "^on:" .github/workflows/ci-cd.yml || echo "ERROR: Missing on field"
grep -q "^jobs:" .github/workflows/ci-cd.yml || echo "ERROR: Missing jobs field"

# Validate gh CLI access
gh workflow view "CI/CD" --yaml > /dev/null || echo "ERROR: Cannot view workflow via gh CLI"

# Validate validation scripts
./scripts/validate-workflows.sh || echo "ERROR: Workflow validation failed"
./scripts/validate-pr-checks.sh || echo "ERROR: PR check validation failed"
```

## Implementation Notes
- Use `gh workflow view` to fetch workflow definitions for validation
- Use `gh workflow run` to test workflows manually (with appropriate inputs)
- Validation scripts should be idempotent and provide clear output
- Validation should check both syntax and semantic correctness
- Validation scripts should be runnable locally and in CI

