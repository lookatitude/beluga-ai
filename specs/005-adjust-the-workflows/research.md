# Research: Adjust GitHub Workflows and Pipelines

**Feature**: 005-adjust-the-workflows  
**Date**: 2025-01-27  
**Phase**: 0 - Research

## Research Questions

### 1. How to implement auto-fix for linting and formatting in GitHub Actions?

**Decision**: Use `golangci-lint` with `--fix` flag and `gofmt` with write mode.

**Rationale**: 
- `golangci-lint` supports `--fix` flag that automatically fixes many linting issues
- `gofmt -w` can automatically format code
- Both tools are already used in the repository (Makefile has `lint-fix` target)
- Auto-fix should run but not block the pipeline (advisory check)

**Alternatives Considered**:
- Manual fix suggestions only: Rejected - user requirement explicitly states auto-fix
- Separate fix job that commits changes: Rejected - too complex, may cause merge conflicts
- Pre-commit hooks: Rejected - not applicable to CI/CD pipeline context

**Implementation**:
- Add `--fix` flag to golangci-lint action
- Run `gofmt -w` to format code
- Use `continue-on-error: true` for lint job (advisory)
- Upload fixed files as artifacts (optional, for review)

### 2. How to enforce coverage thresholds with warnings (not blocking)?

**Decision**: Calculate coverage, check threshold, emit warning annotation if below 80%, but don't exit with error.

**Rationale**:
- User requirement: "if coverage below 80% give warning but continue"
- GitHub Actions supports `::warning::` annotations that appear in PR checks
- Coverage calculation already exists in workflow (needs adjustment)
- Must distinguish between "coverage calculation failed" (error) vs "coverage below threshold" (warning)

**Alternatives Considered**:
- Fail pipeline if coverage below threshold: Rejected - contradicts user requirement
- No coverage check: Rejected - user requirement specifies 80% threshold with warning
- Separate coverage job with continue-on-error: Considered but current approach is cleaner

**Implementation**:
- Calculate coverage using `go tool cover -func`
- Extract percentage, compare to 80%
- If below threshold: `echo "::warning::Coverage X% is below 80% threshold"`
- Don't exit with error (allow pipeline to continue)
- Apply to both unit tests and integration tests separately

### 3. How to ensure security checks block the pipeline?

**Decision**: Security job must not have `continue-on-error: true`, and all security tools must exit with error code on failure.

**Rationale**:
- User requirement: "Security checks - Run security check, if they pass continue, otherwise fail the pipeline"
- Current workflow already has security job without continue-on-error
- Need to ensure gosec, govulncheck, gitleaks, Trivy all fail appropriately
- Security failures are critical and must block merging

**Alternatives Considered**:
- Security warnings only: Rejected - security is explicitly critical per user requirement
- Separate critical vs advisory security checks: Considered but adds complexity, all security should be critical

**Implementation**:
- Verify security job does NOT have `continue-on-error: true`
- Ensure gosec exits with error if issues found (already implemented)
- Ensure gitleaks exits with error if secrets detected (already implemented)
- Trivy can use `continue-on-error: true` for file system scan (already configured) but results uploaded to GitHub Security tab
- Security job must be in critical path (required for merge)

### 4. How to test workflows using `gh` CLI?

**Decision**: Use `gh workflow view` to inspect workflows, `gh workflow run` to test workflows, and validation scripts to verify configuration.

**Rationale**:
- User requirement: "use gh to grab the pipelines and test them"
- `gh workflow view` can fetch workflow YAML for inspection
- `gh workflow run` can trigger workflows manually for testing
- Validation scripts already exist and can be enhanced
- Can validate workflow syntax and structure without running full pipeline

**Alternatives Considered**:
- Manual testing only: Rejected - user explicitly requested `gh` usage
- Act workflow testing: Considered but `gh` CLI is more accessible and already mentioned by user

**Implementation**:
- Use `gh workflow view <workflow-name> --yaml` to fetch workflow definitions
- Use `gh workflow run <workflow-name>` to test workflows
- Enhance validation scripts to check for:
  - Correct job ordering and dependencies
  - Critical vs advisory job configuration
  - Coverage threshold enforcement
  - Auto-fix flags in lint jobs
  - Security check blocking behavior

### 5. How to integrate GoReleaser with documentation generation and website updates?

**Decision**: Release workflow should: 1) Run GoReleaser, 2) Generate API docs using `scripts/generate-docs.sh`, 3) Build and deploy website, 4) Tag and publish release.

**Rationale**:
- User requirement specifies: "use the go releaser to generate a release", "Generate godocs", "Update website based on the docs", "Tag and publish the release"
- GoReleaser already configured (`.goreleaser.yml` exists)
- Documentation generation script exists (`scripts/generate-docs.sh`)
- Website deployment workflow exists (`website_deploy.yml`) - can be integrated or called from release workflow
- Release workflow already has structure, needs enhancement

**Alternatives Considered**:
- Separate workflows for docs and release: Rejected - user wants integrated release pipeline
- Manual documentation updates: Rejected - user requires automatic generation
- Pre-release documentation: Considered but post-release is more appropriate (docs reflect released version)

**Implementation**:
- Release workflow sequence:
  1. Pre-release checks (tests, build) - already exists
  2. Run GoReleaser to create release artifacts
  3. Generate API documentation using `make docs-generate` or `scripts/generate-docs.sh`
  4. Build Docusaurus website (includes generated docs)
  5. Deploy website to GitHub Pages (or trigger website_deploy workflow)
  6. Tag and publish release (handled by GoReleaser)
- Ensure documentation generation uses released version/tag

### 6. How to ensure policy checks are advisory (warnings only)?

**Decision**: Policy check job should use `continue-on-error: true` and emit warning annotations instead of error annotations.

**Rationale**:
- User requirement: "PR checks & policy checks - should give warnings but not stop the release"
- Current policy job may be blocking (needs verification)
- Policy checks are about code quality standards, not correctness
- Warnings provide feedback without blocking development

**Alternatives Considered**:
- Keep policy checks as blocking: Rejected - contradicts user requirement
- Remove policy checks: Rejected - still valuable as advisory feedback

**Implementation**:
- Add `continue-on-error: true` to policy job
- Change `::error::` to `::warning::` in policy check steps
- Ensure policy job doesn't block other jobs (no dependencies from policy job)
- Policy checks should still run and report, just not fail pipeline

## Technology Decisions

### GitHub Actions Workflows
- **Format**: YAML files in `.github/workflows/`
- **Triggers**: `pull_request`, `push` (branches/tags), `workflow_dispatch`
- **Jobs**: Separate jobs for each concern (policy, lint, security, tests, build, release)
- **Job Dependencies**: Use `needs:` to control execution order
- **Criticality**: `continue-on-error: true` for advisory, default (false) for critical

### Linting and Formatting
- **Tool**: `golangci-lint` with `--fix` flag
- **Formatting**: `gofmt -w` for automatic formatting
- **Job Configuration**: Advisory (continue-on-error: true)

### Coverage Calculation
- **Tool**: `go tool cover -func` for coverage calculation
- **Threshold**: 80% for both unit and integration tests
- **Enforcement**: Warning annotation if below threshold, don't block pipeline
- **Reporting**: Upload coverage artifacts, display in PR checks

### Security Scanning
- **Tools**: gosec, govulncheck, gitleaks, Trivy
- **Job Configuration**: Critical (must pass, no continue-on-error)
- **Reporting**: Upload security reports as artifacts, upload Trivy results to GitHub Security tab

### Release Process
- **Tool**: GoReleaser (`.goreleaser.yml` configured)
- **Documentation**: `scripts/generate-docs.sh` using gomarkdoc
- **Website**: Docusaurus build and deploy to GitHub Pages
- **Triggers**: Tag pushes, workflow_dispatch, automated via release-please

### Workflow Testing
- **Tool**: `gh` CLI (`gh workflow view`, `gh workflow run`)
- **Validation**: Existing scripts in `scripts/validate-*.sh`
- **Local Testing**: Makefile targets (`make ci-local`)

## Dependencies and Integration Points

### Existing Tools (Must Use)
- `scripts/validate-workflows.sh` - Workflow file validation
- `scripts/validate-pr-checks.sh` - PR check configuration
- `scripts/validate-coverage.sh` - Coverage calculation validation
- `scripts/validate-release.sh` - Release workflow validation
- `scripts/generate-docs.sh` - API documentation generation
- `Makefile` targets: `lint-fix`, `test-coverage-threshold`, `ci-local`, `docs-generate`

### Workflow Files to Modify
- `.github/workflows/ci-cd.yml` - Main CI/CD pipeline
- `.github/workflows/release.yml` - Release pipeline
- `.github/workflows/website_deploy.yml` - Website deployment (may need integration with release)

### Configuration Files
- `.goreleaser.yml` - GoReleaser configuration (already exists)
- `release-please-config.json` - Automated versioning (already exists)

## Unknowns Resolved

All technical unknowns from the specification have been resolved:
- ✅ Auto-fix implementation approach determined
- ✅ Coverage threshold warning mechanism determined
- ✅ Security check blocking behavior confirmed
- ✅ Workflow testing approach with `gh` CLI determined
- ✅ Release pipeline integration approach determined
- ✅ Policy check advisory configuration determined

### 7. How to implement manual triggers for all workflow steps?

**Decision**: Use `workflow_dispatch` event with input parameters to control individual step execution.

**Rationale**:
- GitHub Actions supports `workflow_dispatch` for manual workflow triggering
- Input parameters can control which steps execute using conditional `if` statements
- Supports manual triggering via GitHub UI, GitHub CLI (`gh workflow run`), and REST API
- Allows granular control over workflow execution without modifying workflow files
- Common pattern in Go library projects for flexible CI/CD operations

**Alternatives Considered**:
- Separate workflows for each step: Rejected - too many workflow files to maintain
- Environment-based step control: Considered but inputs are more explicit and user-friendly
- Job-level manual triggers: Rejected - GitHub Actions doesn't support manual job triggers, only workflow-level

**Implementation**:
- Add `workflow_dispatch` to all workflow `on:` sections
- Define boolean inputs for each major step (e.g., `run_policy`, `run_lint`, `run_security`, `run_tests`, `run_build`, `run_release`)
- Use conditional `if:` statements on steps/jobs based on input values
- Default all inputs to `false` for safety; require explicit selection
- Support both individual step execution and full pipeline execution

**Pattern from Research**:
- Common in Go projects: Use workflow_dispatch with inputs for flexible execution
- Example pattern: `if: ${{ github.event.inputs.run_lint == 'true' }}`
- Allows developers to run specific checks without running entire pipeline

### 8. What are common release automation patterns in Go library projects?

**Decision**: Implement patterns from changie, langchaingo, and other Go projects: changelog generation, semantic versioning, GoReleaser integration, and automated release notes.

**Rationale**:
- **Changie** ([github.com/miniscruff/changie](https://github.com/miniscruff/changie)): File-based changelog management, separates changelog from commit history
- **LangChainGo** and other Go projects: Use GoReleaser with changelog generation
- Common pattern: Generate changelog → Create release → Tag → Publish
- Semantic versioning is standard in Go ecosystem
- Automated release notes improve release quality and consistency

**Common Patterns Identified**:
1. **Changelog Generation**:
   - Use changie or similar tool for structured changelog management
   - File-based approach (`.changes/` directory) keeps changelog separate from commits
   - Generate CHANGELOG.md before release
   - Include changelog in release notes

2. **Release Automation**:
   - GoReleaser for artifact generation and GitHub releases
   - Semantic versioning (vX.Y.Z format)
   - Automated tag creation
   - Release notes from changelog or git commits

3. **Pre-Release Validation**:
   - Run full test suite before release
   - Verify build succeeds
   - Check version format
   - Validate changelog exists

4. **Post-Release Tasks**:
   - Update documentation
   - Deploy website (if applicable)
   - Notify stakeholders (optional)

**Alternatives Considered**:
- Manual changelog management: Rejected - error-prone and inconsistent
- Commit-based changelog: Rejected - changie pattern (file-based) is cleaner
- Release-please only: Considered but changie provides more control over changelog format

**Implementation**:
- Integrate changie for changelog management (optional, can use git-based changelog initially)
- Use GoReleaser changelog generation or changie CLI
- Generate changelog as part of release workflow
- Include changelog in GoReleaser release notes
- Support both automated (release-please) and manual releases with changelog

### 9. How to integrate changelog generation with release pipeline?

**Decision**: Generate changelog before GoReleaser runs, include it in release notes, and support both changie and git-based approaches.

**Rationale**:
- Changelog should be generated before release artifacts are created
- GoReleaser can use changelog for release notes
- Support multiple changelog sources (changie, git commits, manual)
- Changelog generation should be optional (can skip if not configured)

**Implementation**:
- Add changelog generation step before GoReleaser in release workflow
- Support changie CLI if `.changie.yaml` exists
- Fallback to git-based changelog generation if changie not configured
- Pass changelog to GoReleaser via configuration
- Make changelog generation optional via workflow input

**Pattern from Research**:
- Changie workflow: `changie batch` → `changie merge` → Include in release
- GoReleaser pattern: Use `changelog.use: git` or custom changelog file
- Common: Generate changelog, validate it, include in release notes

## Updated Technology Decisions

### Manual Workflow Triggers
- **Event**: `workflow_dispatch` with input parameters
- **Input Types**: Boolean flags for each major step
- **Access Methods**: GitHub UI, `gh workflow run`, REST API
- **Pattern**: Conditional step execution based on inputs

### Release Automation Patterns
- **Changelog Tool**: Changie (file-based) or git-based changelog
- **Release Tool**: GoReleaser (already configured)
- **Versioning**: Semantic versioning (vX.Y.Z)
- **Release Notes**: Generated from changelog or git commits
- **Integration**: Changelog generation → GoReleaser → Tag → Publish

## Updated Dependencies and Integration Points

### New Tools to Consider
- **Changie** (optional): File-based changelog management ([changie.dev](https://changie.dev/))
- **GitHub CLI (`gh`)**: For manual workflow triggering and testing
- **Workflow Inputs**: For granular step control

### Updated Workflow Files to Modify
- `.github/workflows/ci-cd.yml` - Add workflow_dispatch with step inputs
- `.github/workflows/release.yml` - Add workflow_dispatch, changelog generation
- `.github/workflows/website_deploy.yml` - Add workflow_dispatch (if needed)

### New Configuration Files
- `.changie.yaml` (optional) - Changie configuration for changelog management
- Update `.goreleaser.yml` - Include changelog in release notes

## Updated Unknowns Resolved

All technical unknowns from the specification have been resolved:
- ✅ Auto-fix implementation approach determined
- ✅ Coverage threshold warning mechanism determined
- ✅ Security check blocking behavior confirmed
- ✅ Workflow testing approach with `gh` CLI determined
- ✅ Release pipeline integration approach determined
- ✅ Policy check advisory configuration determined
- ✅ Manual trigger implementation approach determined
- ✅ Release automation patterns researched and documented
- ✅ Changelog generation integration approach determined

## Next Steps

Proceed to Phase 1: Design & Contracts to create:
- Data model for workflow jobs and check results (including manual trigger inputs)
- Contracts for each workflow validation requirement (including manual triggers)
- Quickstart guide for testing workflows (including manual trigger examples)
- Agent context file updates

