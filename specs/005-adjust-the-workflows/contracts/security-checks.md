# Contract: Security Checks (Critical - Must Pass)

**Feature**: 005-adjust-the-workflows  
**Date**: 2025-01-27  
**Contract ID**: C004

## Purpose
Ensure security checks must pass; if they fail, the pipeline fails and blocks merging.

## Input
- Code changes in pull request
- Security scanning tools: gosec, govulncheck, gitleaks, Trivy

## Validation Rules

### Rule 1: Security Job Must Be Critical
- Security job MUST NOT have `continue-on-error: true`
- Security job failures MUST cause pipeline to fail
- Security job MUST block merging if it fails
- Security job MUST emit `::error::` annotations on failure

### Rule 2: Multiple Security Tools Must Run
- gosec MUST run and check for security issues
- govulncheck MUST run and check for vulnerabilities
- gitleaks MUST run and check for secrets
- Trivy MUST run for file system scanning (can use continue-on-error for scan, but results uploaded)

### Rule 3: Security Tool Failures Must Block
- gosec finding issues → MUST exit with error code
- gitleaks detecting secrets → MUST exit with error code
- govulncheck finding vulnerabilities → SHOULD be reported (may not block, tool dependent)
- Trivy critical/high findings → Uploaded to GitHub Security tab (may not block scan step)

### Rule 4: Security Reports Must Be Uploaded
- Security scan results MUST be uploaded as artifacts
- Trivy results MUST be uploaded to GitHub Security tab (SARIF format)
- Security reports MUST be retained for at least 30 days

## Success Criteria
- All security tools run on every PR
- Security issues cause pipeline to fail
- Security failures block merging
- Security reports are available for review
- Trivy results appear in GitHub Security tab

## Failure Modes
- Security job has continue-on-error → Error: "Security job must be critical"
- Security tools not running → Error: "All security tools must run"
- Security issues not blocking → Error: "Security failures must block pipeline"
- Security reports not uploaded → Warning: "Security reports should be uploaded"

## Test Validation
```bash
# Validate security job is critical
grep -A 3 "name:.*Security" .github/workflows/ci-cd.yml | grep -q "continue-on-error: true" && echo "ERROR: Security job must not have continue-on-error"

# Validate all security tools are present
grep -q "gosec" .github/workflows/ci-cd.yml || echo "ERROR: gosec not found"
grep -q "govulncheck" .github/workflows/ci-cd.yml || echo "ERROR: govulncheck not found"
grep -q "gitleaks" .github/workflows/ci-cd.yml || echo "ERROR: gitleaks not found"
grep -q "trivy" .github/workflows/ci-cd.yml || echo "WARNING: trivy not found"

# Validate security tools exit on error
grep -A 5 "gosec" .github/workflows/ci-cd.yml | grep -q "exit 1" || echo "WARNING: gosec may not exit on error"
grep -A 5 "gitleaks" .github/workflows/ci-cd.yml | grep -q "exit 1" || echo "WARNING: gitleaks may not exit on error"
```

## Implementation Notes
- Security job should run early in pipeline (can run in parallel with lint)
- gosec: `gosec $(go list ./...)` - exits with error if issues found
- govulncheck: `govulncheck $(go list ./...)` - reports vulnerabilities
- gitleaks: `gitleaks detect` - exits with error if secrets found
- Trivy: Use `aquasecurity/trivy-action@v0.33.1` with SARIF output
- Upload all security reports as artifacts
- Security job must be in critical path (required for merge)

