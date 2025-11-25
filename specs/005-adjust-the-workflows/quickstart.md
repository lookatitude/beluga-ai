# Quickstart: Testing and Validating GitHub Workflows

**Feature**: 005-adjust-the-workflows  
**Date**: 2025-01-27  
**Purpose**: Guide for testing and validating adjusted GitHub Actions workflows

## Prerequisites

1. **GitHub CLI (`gh`) installed and authenticated**
   ```bash
   gh --version
   gh auth status
   ```

2. **Repository access**
   - Must have read access to repository
   - Must have workflow permissions (for running workflows)

3. **Local validation tools**
   - Python 3 (for YAML validation)
   - Bash (for validation scripts)
   - Go 1.24 (for testing Go-related workflows)

## Step 1: View Workflow Definitions

Use `gh` CLI to fetch and inspect workflow definitions:

```bash
# View CI/CD workflow
gh workflow view "CI/CD" --yaml > /tmp/ci-cd.yml

# View release workflow
gh workflow view release.yml --yaml > /tmp/release.yml

# View website deployment workflow
gh workflow view "Deploy Docusaurus Website to GitHub Pages" --yaml > /tmp/website-deploy.yml

# List all workflows
gh workflow list
```

**Expected Result**: Workflow YAML files are fetched and can be inspected.

## Step 2: Validate Workflow Syntax

Validate YAML syntax and structure:

```bash
# Run workflow validation script
./scripts/validate-workflows.sh

# Or validate manually with Python
python3 -c "import yaml; yaml.safe_load(open('.github/workflows/ci-cd.yml'))"
```

**Expected Result**: All workflow files pass YAML validation.

## Step 3: Validate PR Check Configuration

Check that critical vs advisory jobs are properly configured:

```bash
# Validate PR check configuration
./scripts/validate-pr-checks.sh .github/workflows/ci-cd.yml

# Check manually:
# - Policy job should have continue-on-error: true
# - Lint job should have continue-on-error: true
# - Security job should NOT have continue-on-error
# - Unit test job should NOT have continue-on-error
# - Integration test job should NOT have continue-on-error
# - Build job should NOT have continue-on-error
```

**Expected Result**: 
- Advisory jobs (policy, lint, coverage) have `continue-on-error: true`
- Critical jobs (security, tests, build) do NOT have `continue-on-error: true`

## Step 4: Validate Coverage Configuration

Check that coverage threshold warnings are properly configured:

```bash
# Validate coverage calculation
./scripts/validate-coverage.sh

# Check manually in workflow:
# - Coverage threshold check uses ::warning:: (not ::error::)
# - Coverage threshold check does NOT exit with error code
# - Coverage is calculated for both unit and integration tests
```

**Expected Result**: Coverage below 80% generates warnings but doesn't block pipeline.

## Step 5: Validate Lint Auto-Fix

Check that linting and formatting use auto-fix:

```bash
# Check workflow for auto-fix flags
grep -A 5 "golangci-lint" .github/workflows/ci-cd.yml | grep -q "--fix" || echo "ERROR: Auto-fix not enabled"

# Test locally
make lint-fix

# Check that lint job is advisory
grep -A 3 "name:.*Lint" .github/workflows/ci-cd.yml | grep -q "continue-on-error: true" || echo "ERROR: Lint job must be advisory"
```

**Expected Result**: 
- `golangci-lint` runs with `--fix` flag
- `gofmt` or `gofumpt` runs with write mode
- Lint job has `continue-on-error: true`

## Step 6: Validate Security Checks

Check that security checks are critical (must pass):

```bash
# Validate security job configuration
grep -A 3 "name:.*Security" .github/workflows/ci-cd.yml | grep -q "continue-on-error: true" && echo "ERROR: Security job must be critical"

# Check all security tools are present
grep -q "gosec" .github/workflows/ci-cd.yml || echo "ERROR: gosec not found"
grep -q "govulncheck" .github/workflows/ci-cd.yml || echo "ERROR: govulncheck not found"
grep -q "gitleaks" .github/workflows/ci-cd.yml || echo "ERROR: gitleaks not found"
grep -q "trivy" .github/workflows/ci-cd.yml || echo "WARNING: trivy not found"

# Test security checks locally
make security
```

**Expected Result**: 
- Security job does NOT have `continue-on-error: true`
- All security tools (gosec, govulncheck, gitleaks, Trivy) are configured
- Security failures would block the pipeline

## Step 7: Validate Release Pipeline

Check that release workflow includes all required steps:

```bash
# Validate release workflow
./scripts/validate-release.sh .github/workflows/release.yml

# Check manually:
# - GoReleaser is used
# - Documentation generation is included
# - Website update is included
# - Concurrency control is configured
# - Multiple triggers are supported (workflow_dispatch, tags, automated)
```

**Expected Result**: 
- GoReleaser is configured
- Documentation generation step exists
- Website deployment step exists
- Concurrency group prevents simultaneous releases

## Step 8: Test Workflow Execution (Optional)

Test workflows by triggering them manually (requires appropriate permissions):

```bash
# Test CI/CD workflow (on a test branch)
gh workflow run "CI/CD" --ref <test-branch>

# View workflow run
gh run list --workflow="CI/CD"

# Watch workflow execution
gh run watch

# Test release workflow (manual trigger)
gh workflow run release.yml --field tag=v1.0.0-test
```

**Expected Result**: Workflows can be triggered and executed successfully.

## Step 9: Validate Local CI Checks

Run local CI checks to verify they match workflow behavior:

```bash
# Run comprehensive CI checks locally
make ci-local

# This should run:
# 1. Format check
# 2. Lint & Format
# 3. Go vet
# 4. Security scans
# 5. Unit tests
# 6. Integration tests
# 7. Coverage check
# 8. Build verification
```

**Expected Result**: Local CI checks pass and match workflow behavior.

## Step 10: Test Manual Workflow Triggers

Test manual triggering of workflows and individual steps:

```bash
# List available workflows
gh workflow list

# Trigger CI/CD workflow manually (run all steps)
# When no inputs are provided, all steps run (default behavior)
gh workflow run "CI/CD" --ref main

# Trigger CI/CD workflow with only lint step
gh workflow run "CI/CD" --ref main -f run_lint=true

# Trigger CI/CD workflow with only security checks
gh workflow run "CI/CD" --ref main -f run_security=true

# Trigger CI/CD workflow with only tests
gh workflow run "CI/CD" --ref main -f run_unit_tests=true -f run_integration_tests=true

# Trigger CI/CD workflow with multiple steps
gh workflow run "CI/CD" --ref main -f run_lint=true -f run_security=true -f run_unit_tests=true

# Trigger release workflow manually (pre-release checks only)
gh workflow run "Release" --ref main -f run_pre_release=true -f tag=v1.0.0-test

# Trigger release workflow manually (full release)
gh workflow run "Release" --ref main -f run_release=true -f tag=v1.0.0-test

# Trigger release workflow with documentation generation
gh workflow run "Release" --ref main -f run_release=true -f run_docs=true -f tag=v1.0.0-test

# Trigger release workflow with website update
gh workflow run "Release" --ref main -f run_release=true -f run_docs=true -f run_website=true -f tag=v1.0.0-test

# View workflow run status
gh run list --workflow="CI/CD"

# Watch workflow execution
gh run watch

# View workflow run details
gh run view <run-id>

# View workflow run logs
gh run view <run-id> --log
```

**Expected Result**: 
- Workflows can be triggered manually via GitHub CLI
- Individual steps can be executed selectively via input parameters
- Workflow runs appear in GitHub Actions UI
- Manual triggers work independently of automatic triggers
- Only selected steps execute when inputs are provided

## Step 11: Verify Workflow Integration

Check that workflows use existing repository tools:

```bash
# Check that workflows reference Makefile targets
grep -q "make.*lint\|make.*test\|make.*build" .github/workflows/ci-cd.yml || echo "WARNING: Workflows may not use Makefile"

# Check that workflows reference validation scripts
grep -q "validate-.*\.sh\|scripts/" .github/workflows/ci-cd.yml || echo "WARNING: Workflows may not use validation scripts"

# Check that release workflow uses documentation script
grep -q "generate-docs\|docs-generate" .github/workflows/release.yml || echo "ERROR: Release workflow must generate docs"

# Run all validation scripts
./scripts/validate-workflows.sh
./scripts/validate-pr-checks.sh .github/workflows/ci-cd.yml
./scripts/validate-manual-triggers.sh
./scripts/validate-changelog.sh
```

**Expected Result**: Workflows leverage existing repository tools (Makefile, scripts), and all validation scripts pass.

## Troubleshooting

### Workflow Not Found
```bash
# List all workflows to find correct name
gh workflow list

# Use exact workflow name or file name
gh workflow view "CI/CD" --yaml
gh workflow view ci-cd.yml --yaml
```

### Validation Script Fails
```bash
# Make scripts executable
chmod +x scripts/validate-*.sh

# Run with verbose output
bash -x scripts/validate-workflows.sh
```

### Coverage Calculation Issues
```bash
# Generate coverage locally
make test-coverage

# Check coverage file
go tool cover -func=coverage/coverage.out

# Verify threshold check
make test-coverage-threshold
```

### Security Checks Fail
```bash
# Run security checks locally
make security

# Check individual tools
gosec ./...
govulncheck ./...
gitleaks detect --no-banner
```

## Success Criteria

All validation steps should pass:
- ✅ Workflow files are valid YAML
- ✅ PR checks are properly configured (critical vs advisory)
- ✅ Coverage thresholds generate warnings (not errors)
- ✅ Lint auto-fix is enabled
- ✅ Security checks are critical (must pass)
- ✅ Release pipeline includes all required steps
- ✅ Workflows use existing repository tools
- ✅ Workflows can be tested with `gh` CLI
- ✅ Manual triggers work via GitHub UI, CLI, and API
- ✅ Individual steps can be executed selectively
- ✅ Changelog generation works (if configured)

## Next Steps

After validation passes:
1. Create a test PR to verify workflow behavior
2. Monitor workflow execution in GitHub Actions
3. Verify PR checks show correct status (critical vs advisory)
4. Test release workflow with a test tag
5. Confirm documentation generation and website update work

