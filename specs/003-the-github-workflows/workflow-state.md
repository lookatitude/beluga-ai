# Current Workflow State Documentation

**Date**: 2025-01-27  
**Task**: T001 - Verify current workflow state

## Workflow Files

### 1. `.github/workflows/ci-cd.yml`
- **Triggers**: 
  - `pull_request`: All PRs
  - `push`: Main branch
- **Jobs**: lint, security, unit-tests, integration-tests, coverage, build
- **Status**: Active, needs coverage calculation fix

### 2. `.github/workflows/release.yml`
- **Triggers**:
  - `push.tags`: Version tags (v*)
  - `workflow_dispatch`: Manual trigger with tag input
- **Jobs**: pre-release, release
- **Status**: Active, needs consolidation with release_please.yml

### 3. `.github/workflows/release_please.yml`
- **Triggers**:
  - `push.branches`: Main branch
- **Jobs**: release-please
- **Status**: Active, needs consolidation with release.yml

### 4. `.github/workflows/website_deploy.yml`
- **Triggers**:
  - `push.branches`: Main branch
  - `push.paths`: website/**, docs/**, pkg/**, scripts/generate-docs.sh, Makefile
  - `workflow_dispatch`: Manual trigger
- **Jobs**: deploy
- **Status**: Active, needs documentation generation integration

## Issues Identified

1. **Coverage Calculation**: Shows 0% when tests pass (lines 376-384 in ci-cd.yml)
2. **Release Workflows**: Two separate workflows that could conflict
3. **PR Checks**: No distinction between critical and advisory checks
4. **Documentation**: Not automatically generated on main merges

