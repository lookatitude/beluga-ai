# Quickstart: Validating GitHub Workflow Fixes

**Feature**: Fix GitHub Workflows, Coverage, PR Checks, and Documentation Generation  
**Date**: 2025-01-27

## Prerequisites

- GitHub repository with Actions enabled
- Go 1.24.0+ installed locally
- Access to repository (for testing workflows)

## Validation Steps

### 1. Verify Coverage Calculation Fix

**Goal**: Ensure coverage is calculated correctly and shows accurate percentage (not 0%)

**Steps**:
```bash
# Run tests with coverage
go test -v -race -coverprofile=coverage.unit.out -covermode=atomic $(go list ./pkg/... | grep -v -E '(specs|examples)')

# Verify coverage file exists
test -f coverage.unit.out && echo "✅ Coverage file generated" || echo "❌ Coverage file missing"

# Parse coverage percentage
pct=$(go tool cover -func=coverage.unit.out | tail -n1 | awk '{print $3}' | sed 's/%//')
echo "📊 Coverage: ${pct}%"

# Verify coverage is not 0% (assuming tests pass)
if [ "$pct" == "0" ] || [ -z "$pct" ]; then
  echo "❌ Coverage calculation failed or shows 0%"
  exit 1
else
  echo "✅ Coverage calculated correctly: ${pct}%"
fi

# Check threshold (should be >= 80%)
if awk "BEGIN {exit !($pct < 80)}"; then
  echo "⚠️  Coverage ${pct}% is below 80% threshold"
else
  echo "✅ Coverage ${pct}% meets 80% threshold"
fi
```

**Expected Result**: Coverage percentage is calculated and displayed correctly (not 0% when tests pass)

### 2. Verify Release Workflow Consolidation

**Goal**: Ensure unified release workflow supports both automated and manual releases

**Steps**:
```bash
# Check workflow file exists
test -f .github/workflows/release.yml && echo "✅ Release workflow exists" || echo "❌ Release workflow missing"

# Verify workflow supports multiple triggers
grep -q "workflow_dispatch" .github/workflows/release.yml && echo "✅ Manual trigger supported" || echo "❌ Manual trigger missing"
grep -q "tags:" .github/workflows/release.yml && echo "✅ Tag trigger supported" || echo "❌ Tag trigger missing"
grep -q "release-please" .github/workflows/release.yml && echo "✅ Automated versioning supported" || echo "❌ Automated versioning missing"

# Verify no duplicate release workflows
count=$(ls -1 .github/workflows/release*.yml 2>/dev/null | wc -l)
if [ "$count" -gt 1 ]; then
  echo "⚠️  Multiple release workflows found (should be consolidated)"
else
  echo "✅ Single release workflow found"
fi
```

**Expected Result**: Single unified release workflow with support for automated and manual releases

### 3. Verify PR Check Configuration

**Goal**: Ensure PR checks accurately reflect status with critical/advisory distinction

**Steps**:
```bash
# Check CI workflow has proper check configuration
test -f .github/workflows/ci-cd.yml && echo "✅ CI workflow exists" || echo "❌ CI workflow missing"

# Verify critical checks are defined
grep -q "unit-tests" .github/workflows/ci-cd.yml && echo "✅ Unit tests check present" || echo "❌ Unit tests check missing"
grep -q "security" .github/workflows/ci-cd.yml && echo "✅ Security check present" || echo "❌ Security check missing"
grep -q "build" .github/workflows/ci-cd.yml && echo "✅ Build check present" || echo "❌ Build check missing"

# Verify coverage threshold check
grep -q "80" .github/workflows/ci-cd.yml && echo "✅ Coverage threshold (80%) configured" || echo "❌ Coverage threshold missing"
```

**Expected Result**: PR checks are properly configured with critical checks blocking merge

### 4. Verify Documentation Generation

**Goal**: Ensure documentation is generated automatically on main branch merges

**Steps**:
```bash
# Check documentation generation script exists
test -f scripts/generate-docs.sh && echo "✅ Documentation script exists" || echo "❌ Documentation script missing"

# Check script is executable
test -x scripts/generate-docs.sh && echo "✅ Script is executable" || echo "❌ Script not executable"

# Verify website deploy workflow includes doc generation
if [ -f .github/workflows/website_deploy.yml ]; then
  grep -q "docs-generate\|generate-docs" .github/workflows/website_deploy.yml && echo "✅ Documentation generation in workflow" || echo "❌ Documentation generation missing from workflow"
else
  echo "⚠️  Website deploy workflow not found"
fi

# Test documentation generation locally
./scripts/generate-docs.sh && echo "✅ Documentation generation works" || echo "❌ Documentation generation failed"

# Verify generated docs exist
if [ -d "website/docs/api/packages" ]; then
  count=$(ls -1 website/docs/api/packages/*.md 2>/dev/null | wc -l)
  if [ "$count" -gt 0 ]; then
    echo "✅ Generated ${count} documentation files"
  else
    echo "⚠️  No documentation files generated"
  fi
else
  echo "⚠️  Documentation output directory not found"
fi
```

**Expected Result**: Documentation generation script works and is integrated into website deploy workflow

### 5. Validate Workflow Files

**Goal**: Ensure all workflow files are valid YAML

**Steps**:
```bash
# Check for YAML syntax errors (requires yamllint or similar)
for workflow in .github/workflows/*.yml; do
  echo "Validating $workflow..."
  # Using Python yaml module as fallback
  python3 -c "import yaml; yaml.safe_load(open('$workflow'))" 2>/dev/null && echo "✅ $workflow is valid YAML" || echo "❌ $workflow has YAML errors"
done
```

**Expected Result**: All workflow files are valid YAML with no syntax errors

### 6. End-to-End Validation

**Goal**: Create a test PR to validate all fixes work together

**Steps**:
1. Create a test branch: `git checkout -b test-workflow-fixes`
2. Make a small change (e.g., add a comment to a Go file)
3. Commit and push: `git commit -am "test: workflow validation" && git push`
4. Create a PR on GitHub
5. Verify:
   - PR checks run and show accurate status
   - Coverage is calculated and displayed (not 0%)
   - All critical checks pass or fail appropriately
   - Advisory checks show warnings if needed
6. Merge PR to main (if all checks pass)
7. Verify:
   - Documentation is generated automatically
   - Website deployment includes new documentation

**Expected Result**: All workflows function correctly with accurate reporting

## Success Criteria

✅ Coverage calculation shows accurate percentage (not 0% when tests pass)  
✅ Single unified release workflow supports automated and manual releases  
✅ PR checks accurately reflect status with critical/advisory distinction  
✅ Documentation is generated automatically on main branch merges  
✅ All workflow files are valid YAML  
✅ End-to-end validation passes

## Troubleshooting

**Coverage shows 0%**:
- Check if `coverage.unit.out` file is generated
- Verify coverage file is valid: `go tool cover -func=coverage.unit.out`
- Check test command includes `-coverprofile` flag

**Release workflow conflicts**:
- Verify only one release workflow exists
- Check workflow triggers don't overlap
- Ensure conditional execution prevents simultaneous releases

**PR checks not passing**:
- Verify check jobs are defined correctly
- Check check conclusions are set appropriately
- Ensure critical vs advisory distinction is implemented

**Documentation not generating**:
- Verify workflow triggers on main branch merges
- Check documentation script is executable
- Verify gomarkdoc is installed in workflow
- Check script output for errors

