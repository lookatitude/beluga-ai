# Validation Contracts: GitHub Workflows

**Feature**: Fix GitHub Workflows, Coverage, PR Checks, and Documentation Generation  
**Date**: 2025-01-27

## Contract 1: Coverage Calculation Validation

**Purpose**: Ensure test coverage is calculated correctly and reported accurately

**Input**: Test execution results, coverage profile file

**Validation Rules**:
1. Coverage file (`coverage.unit.out`) MUST exist after test execution
2. Coverage file MUST be valid Go coverage format (parseable by `go tool cover`)
3. Total coverage MUST be calculated from all packages, including those with 0% coverage
4. Coverage percentage MUST be >= 0 and <= 100
5. Coverage MUST be reported in PR checks (not show 0% when tests pass)
6. Coverage threshold of 80% MUST be enforced (critical check)

**Success Criteria**:
- Coverage file exists and is valid
- Total coverage is calculated correctly (not 0% when tests pass)
- Coverage percentage is displayed in PR checks
- Coverage threshold check passes or fails appropriately

**Failure Modes**:
- Coverage file missing → Error: "Coverage file not generated"
- Coverage file invalid → Error: "Coverage file format invalid"
- Coverage calculation fails → Error: "Failed to calculate coverage"
- Coverage below threshold → Failure: "Coverage X% is below 80% threshold"

**Test Validation**:
```bash
# Validate coverage file exists
test -f coverage.unit.out || echo "ERROR: Coverage file missing"

# Validate coverage can be parsed
go tool cover -func=coverage.unit.out > /dev/null || echo "ERROR: Coverage file invalid"

# Validate coverage percentage
pct=$(go tool cover -func=coverage.unit.out | tail -n1 | awk '{print $3}' | sed 's/%//')
if [ -z "$pct" ]; then
  echo "ERROR: Failed to extract coverage percentage"
fi

# Validate threshold
if awk "BEGIN {exit !($pct < 80)}"; then
  echo "FAILURE: Coverage ${pct}% is below 80% threshold"
  exit 1
fi
```

## Contract 2: Release Workflow Validation

**Purpose**: Ensure unified release workflow handles both automated and manual releases without conflicts

**Input**: Workflow trigger (automated/manual/tag), version information

**Validation Rules**:
1. Workflow MUST support automated semantic versioning (release-please on main)
2. Workflow MUST support manual releases (workflow_dispatch)
3. Workflow MUST support tag-based releases (tag triggers)
4. Only one release process MUST run at a time (conflict prevention)
5. Version MUST be valid semantic version (vX.Y.Z format)
6. Release artifacts MUST be created successfully

**Success Criteria**:
- Workflow triggers correctly for all three trigger types
- No conflicting releases occur simultaneously
- Version is validated before release
- Release is created successfully on GitHub

**Failure Modes**:
- Invalid version format → Error: "Invalid version format: must be vX.Y.Z"
- Conflicting release in progress → Error: "Another release is in progress"
- Release creation fails → Error: "Failed to create release: {reason}"

**Test Validation**:
```bash
# Validate version format
if ! [[ "$VERSION" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo "ERROR: Invalid version format: $VERSION"
  exit 1
fi

# Validate no conflicting release (check GitHub API or workflow status)
# This would be implemented in workflow logic
```

## Contract 3: PR Check Status Validation

**Purpose**: Ensure PR checks accurately reflect validation status with proper critical/advisory distinction

**Input**: Check results, check type (critical/advisory)

**Validation Rules**:
1. Critical checks (tests, security, build) MUST block merge on failure
2. Advisory checks (lint, coverage threshold) MUST show warnings but allow merge
3. Check status MUST accurately reflect actual validation result
4. Override MUST be required for critical check failures
5. Check conclusions MUST be one of: success, failure, neutral, cancelled, skipped, timed_out, action_required

**Success Criteria**:
- All checks report accurate status
- Critical checks block merge when failing (unless overridden)
- Advisory checks show warnings but don't block merge
- PR status accurately reflects check results

**Failure Modes**:
- Check status incorrect → Error: "Check status does not match validation result"
- Critical check failure without override → Block: "Critical check failed: {check_name}"
- Check conclusion invalid → Error: "Invalid check conclusion: {conclusion}"

**Test Validation**:
```yaml
# Example workflow step
- name: Set check conclusion
  run: |
    if [ "$CHECK_RESULT" == "failure" ] && [ "$CHECK_TYPE" == "critical" ]; then
      echo "::error::Critical check failed: $CHECK_NAME"
      exit 1  # Blocks merge
    elif [ "$CHECK_RESULT" == "failure" ] && [ "$CHECK_TYPE" == "advisory" ]; then
      echo "::warning::Advisory check failed: $CHECK_NAME"
      # Don't exit, allows merge with warning
    fi
```

## Contract 4: Documentation Generation Validation

**Purpose**: Ensure API documentation is generated automatically on main branch merges

**Input**: Source code packages, documentation generation script

**Validation Rules**:
1. Documentation MUST be generated on merges to main branch
2. Documentation MUST NOT be generated on PRs or other branches
3. All packages in `PACKAGES` list MUST have documentation generated
4. Generated documentation MUST be valid Markdown/MDX format
5. Documentation generation failure MUST fail the workflow
6. Generated files MUST be in `website/docs/api/packages/` directory

**Success Criteria**:
- Documentation is generated automatically on main merges
- All packages have documentation files
- Generated files are valid and deployable
- Workflow fails if generation fails

**Failure Modes**:
- Generation script fails → Error: "Documentation generation failed: {reason}"
- Package missing documentation → Error: "Documentation missing for package: {package}"
- Invalid output format → Error: "Generated documentation format invalid"
- Wrong trigger branch → Skip: "Documentation generation skipped (not main branch)"

**Test Validation**:
```bash
# Validate documentation generation
if [ "$GITHUB_REF" != "refs/heads/main" ]; then
  echo "Skipping documentation generation (not main branch)"
  exit 0
fi

# Generate documentation
./scripts/generate-docs.sh || {
  echo "ERROR: Documentation generation failed"
  exit 1
}

# Validate all packages have documentation
for pkg in "${PACKAGES[@]}"; do
  pkg_name=$(basename "$pkg")
  if [ ! -f "website/docs/api/packages/${pkg_name}.md" ]; then
    echo "ERROR: Documentation missing for package: $pkg_name"
    exit 1
  fi
done
```

## Contract 5: Workflow File Validation

**Purpose**: Ensure workflow files are valid YAML and follow best practices

**Input**: Workflow YAML files

**Validation Rules**:
1. Workflow files MUST be valid YAML syntax
2. Workflow files MUST have required fields: `name`, `on`, `jobs`
3. Each job MUST have `runs-on` and `steps`
4. Workflow triggers MUST be clearly defined
5. No duplicate workflow names

**Success Criteria**:
- All workflow files are valid YAML
- All required fields are present
- No syntax errors
- Workflows can be parsed by GitHub Actions

**Failure Modes**:
- Invalid YAML syntax → Error: "YAML syntax error: {line}:{column}"
- Missing required field → Error: "Missing required field: {field}"
- Duplicate workflow name → Error: "Duplicate workflow name: {name}"

**Test Validation**:
```bash
# Validate YAML syntax
for workflow in .github/workflows/*.yml; do
  yamllint "$workflow" || {
    echo "ERROR: Invalid YAML in $workflow"
    exit 1
  }
done

# Validate required fields (using yq or similar)
for workflow in .github/workflows/*.yml; do
  if ! yq eval '.name' "$workflow" > /dev/null 2>&1; then
    echo "ERROR: Missing 'name' field in $workflow"
    exit 1
  fi
done
```

