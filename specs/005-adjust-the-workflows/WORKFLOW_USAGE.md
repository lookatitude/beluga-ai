# GitHub Workflows Usage Guide

**Feature**: 005-adjust-the-workflows  
**Date**: 2025-01-27  
**Purpose**: Guide for using GitHub Actions workflows with manual triggers, changelog generation, and release automation

## Overview

This repository uses GitHub Actions workflows for CI/CD, releases, and website deployment. All workflows support manual triggering via `workflow_dispatch` with input parameters for granular step control.

## Workflows

### 1. CI/CD Workflow (`.github/workflows/ci-cd.yml`)

The main CI/CD pipeline runs on pull requests and pushes to `main`/`develop` branches. It includes:

- **Policy Checks** (advisory): Branch naming, PR description, PR size validation
- **Lint & Format** (advisory): Code linting with auto-fix and formatting
- **Security Scans** (critical): gosec, govulncheck, gitleaks, Trivy
- **Unit Tests** (critical): Test execution with coverage reporting
- **Integration Tests** (critical): Integration test execution
- **Coverage Check** (advisory): Coverage threshold validation (80% warning)
- **Build** (critical): Package compilation verification

#### Manual Trigger Inputs

| Input | Description | Default |
|-------|-------------|---------|
| `run_policy` | Run policy checks | `false` |
| `run_lint` | Run lint and format checks | `false` |
| `run_security` | Run security scans | `false` |
| `run_unit_tests` | Run unit tests | `false` |
| `run_integration_tests` | Run integration tests | `false` |
| `run_build` | Run build verification | `false` |

#### Usage Examples

```bash
# Run all CI/CD steps (default when no inputs provided)
gh workflow run "CI/CD" --ref main

# Run only lint checks
gh workflow run "CI/CD" --ref main -f run_lint=true

# Run security and tests
gh workflow run "CI/CD" --ref main -f run_security=true -f run_unit_tests=true

# Run specific combination
gh workflow run "CI/CD" --ref main \
  -f run_lint=true \
  -f run_security=true \
  -f run_unit_tests=true
```

### 2. Release Workflow (`.github/workflows/release.yml`)

The release workflow supports both automated (release-please) and manual/tag-based releases. It includes:

- **Pre-Release Checks**: Dependency verification, tests, build
- **GoReleaser**: Release artifact generation
- **Changelog Generation**: Git-based changelog generation
- **Documentation Generation**: API documentation generation
- **Website Update**: Website deployment preparation

#### Manual Trigger Inputs

| Input | Description | Default |
|-------|-------------|---------|
| `tag` | Release tag (e.g., v1.0.0) | `''` |
| `run_pre_release` | Run pre-release checks | `false` |
| `run_release` | Run release (GoReleaser) | `false` |
| `run_docs` | Generate and update documentation | `false` |
| `run_website` | Update and deploy website | `false` |

#### Usage Examples

```bash
# Full release with all steps
gh workflow run "Release" --ref main \
  -f tag=v1.0.0 \
  -f run_pre_release=true \
  -f run_release=true \
  -f run_docs=true \
  -f run_website=true

# Release with documentation only
gh workflow run "Release" --ref main \
  -f tag=v1.0.0 \
  -f run_release=true \
  -f run_docs=true

# Pre-release checks only
gh workflow run "Release" --ref main \
  -f run_pre_release=true
```

### 3. Website Deployment Workflow (`.github/workflows/website_deploy.yml`)

Automatically deploys the Docusaurus website to GitHub Pages when documentation changes.

#### Manual Trigger

```bash
# Manually deploy website
gh workflow run "Deploy Docusaurus Website to GitHub Pages" --ref main
```

## Job Criticality

### Critical Jobs (Block Merge on Failure)

- **Security Scans**: Must pass for merge
- **Unit Tests**: Must pass for merge
- **Integration Tests**: Must pass for merge
- **Build**: Must pass for merge

### Advisory Jobs (Warnings Only)

- **Policy Checks**: Provide feedback but don't block
- **Lint & Format**: Auto-fix enabled, warnings only
- **Coverage Check**: Warns if below 80% but doesn't block

## Changelog Generation

The release workflow generates changelogs using git commits. Changelog generation:

- Runs before GoReleaser
- Uses git commit history
- Generates CHANGELOG.md file
- Does not block release on failure (advisory)
- Integrates with GoReleaser for release notes

## Concurrency Control

- **Release Workflow**: Uses concurrency group to prevent simultaneous releases
- **Manual Releases**: Take precedence over automated releases (cancel in-progress)

## Validation Scripts

Use validation scripts to verify workflow configuration:

```bash
# Validate workflow syntax and structure
./scripts/validate-workflows.sh

# Validate PR check configuration
./scripts/validate-pr-checks.sh .github/workflows/ci-cd.yml

# Validate manual trigger configuration
./scripts/validate-manual-triggers.sh

# Validate changelog configuration
./scripts/validate-changelog.sh

# Validate coverage configuration
./scripts/validate-coverage.sh

# Validate release workflow
./scripts/validate-release.sh
```

## Best Practices

1. **Use Manual Triggers for Testing**: Test specific workflow steps before full pipeline
2. **Check Coverage Regularly**: Monitor coverage warnings, aim for ≥80%
3. **Review Security Reports**: Address security issues promptly
4. **Validate Before Release**: Run pre-release checks before creating releases
5. **Update Changelog**: Ensure meaningful commit messages for changelog generation

## Troubleshooting

### Workflow Not Found

```bash
# List all workflows
gh workflow list

# View workflow details
gh workflow view "CI/CD"
```

### Manual Trigger Not Working

- Check that input parameters are spelled correctly
- Ensure workflow supports `workflow_dispatch`
- Verify branch reference is correct

### Changelog Not Generated

- Check git history exists
- Verify previous tags exist (for comparison)
- Review workflow logs for errors

### Validation Script Failures

- Run validation scripts individually to identify specific issues
- Check workflow YAML syntax
- Verify job configurations match requirements

## Related Documentation

- [Quickstart Guide](./quickstart.md) - Step-by-step testing guide
- [Contracts](./contracts/) - Detailed contract specifications
- [Research](./research.md) - Technical decisions and patterns

