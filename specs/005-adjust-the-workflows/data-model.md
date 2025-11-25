# Data Model: GitHub Workflows and Pipelines

**Feature**: 005-adjust-the-workflows  
**Date**: 2025-01-27  
**Phase**: 1 - Design

## Entities

### PipelineJob
Represents a single workflow job that performs a specific validation or build task.

**Properties**:
- `name` (string, required): Job identifier (e.g., "lint", "unit-tests", "security")
- `criticality` (enum: "critical" | "advisory", required): Whether job blocks merge on failure
- `dependencies` (array of strings): Names of jobs that must complete before this job runs
- `status` (enum: "pending" | "running" | "success" | "failure" | "cancelled", required): Current execution status
- `continue_on_error` (boolean, required): Whether job allows pipeline to continue on failure
- `steps` (array of Step): Individual steps within the job
- `outputs` (map of string to string): Job outputs (e.g., coverage percentage, test results)

**Validation Rules**:
- Critical jobs MUST NOT have `continue_on_error: true`
- Advisory jobs MUST have `continue_on_error: true`
- Job names MUST be unique within a workflow
- Dependencies MUST reference existing jobs (no circular dependencies)

**State Transitions**:
- `pending` → `running` (when job starts)
- `running` → `success` (when all steps pass)
- `running` → `failure` (when any step fails and continue_on_error is false)
- `running` → `success` (when steps fail but continue_on_error is true - advisory only)
- Any state → `cancelled` (when workflow is cancelled)

### CheckResult
Represents the outcome of a validation check within a pipeline job.

**Properties**:
- `check_type` (enum: "policy" | "lint" | "format" | "unit-test" | "integration-test" | "security" | "build" | "coverage", required): Type of check performed
- `status` (enum: "pass" | "warn" | "fail", required): Check outcome
- `coverage_percentage` (float, optional): Test coverage percentage (0-100, only for test checks)
- `blocking` (boolean, required): Whether check failure blocks the pipeline
- `message` (string, optional): Human-readable result message
- `artifacts` (array of string, optional): Paths to generated artifacts (reports, coverage files)
- `annotations` (array of Annotation): GitHub Actions annotations (error, warning, notice)

**Validation Rules**:
- `coverage_percentage` MUST be between 0 and 100 if present
- `blocking` MUST be false if `status` is "warn"
- `blocking` MUST be true if `status` is "fail" and `check_type` is "security" or "build" or "unit-test" or "integration-test"
- `blocking` MUST be false if `check_type` is "policy" or "lint" or "format" or "coverage"

**State Transitions**:
- Check starts → `status: "pass"` (default)
- Issues found (advisory) → `status: "warn"`, `blocking: false`
- Issues found (critical) → `status: "fail"`, `blocking: true`
- Coverage below threshold → `status: "warn"`, `blocking: false`, `coverage_percentage` set

### ReleaseArtifact
Represents the output of a release process.

**Properties**:
- `version_tag` (string, required): Git tag for the release (e.g., "v1.0.0")
- `release_notes` (string, optional): Release notes/changelog content
- `documentation_generated` (boolean, required): Whether API documentation was generated
- `website_updated` (boolean, required): Whether website was updated with new docs
- `build_artifacts` (array of string, optional): Paths to build artifacts (archives, checksums)
- `publication_status` (enum: "pending" | "published" | "failed", required): Whether release was published
- `go_releaser_output` (string, optional): GoReleaser execution output

**Validation Rules**:
- `version_tag` MUST match semantic versioning format (vX.Y.Z)
- `documentation_generated` MUST be true if `publication_status` is "published"
- `website_updated` MUST be true if `documentation_generated` is true
- `publication_status` MUST be "published" for successful releases

**State Transitions**:
- Release triggered → `publication_status: "pending"`
- GoReleaser runs → `build_artifacts` populated
- Documentation generated → `documentation_generated: true`
- Website deployed → `website_updated: true`
- Release published → `publication_status: "published"`
- Any step fails → `publication_status: "failed"`

### WorkflowConfiguration
Represents the configuration of a GitHub Actions workflow file.

**Properties**:
- `workflow_name` (string, required): Workflow file name (e.g., "ci-cd.yml")
- `triggers` (array of Trigger): Events that trigger the workflow
- `manual_trigger_enabled` (boolean, required): Whether workflow supports manual triggering via workflow_dispatch
- `manual_trigger_inputs` (array of WorkflowInput, optional): Input parameters for manual triggering
- `jobs` (array of PipelineJob): Jobs defined in the workflow
- `env` (map of string to string, optional): Environment variables
- `permissions` (map of string to string, optional): GitHub permissions required
- `concurrency` (Concurrency, optional): Concurrency control configuration

**Validation Rules**:
- Workflow MUST have at least one trigger
- Workflow MUST have at least one job
- Job names MUST be unique within workflow
- YAML syntax MUST be valid
- If `manual_trigger_enabled` is true, `workflow_dispatch` MUST be in triggers
- If `manual_trigger_inputs` exist, each input MUST have corresponding conditional logic in jobs

### WorkflowInput
Represents an input parameter for manual workflow triggering.

**Properties**:
- `name` (string, required): Input parameter name (e.g., "run_lint", "run_tests")
- `description` (string, required): Human-readable description of what the input controls
- `type` (enum: "boolean" | "string" | "choice", required): Input data type
- `required` (boolean, required): Whether input is required (should be false for step controls)
- `default` (string, required): Default value (typically "false" for boolean step controls)

**Validation Rules**:
- Input names MUST be unique within a workflow
- Boolean inputs for step control MUST default to "false"
- Input descriptions MUST clearly indicate which step they control

### Trigger
Represents a workflow trigger event.

**Properties**:
- `type` (enum: "push" | "pull_request" | "workflow_dispatch" | "schedule" | "tag", required): Trigger type
- `branches` (array of string, optional): Branch names (for push/pull_request)
- `tags` (array of string, optional): Tag patterns (for tag triggers)
- `paths` (array of string, optional): File paths that trigger workflow (for push)

### Concurrency
Represents concurrency control for workflow execution.

**Properties**:
- `group` (string, required): Concurrency group identifier
- `cancel_in_progress` (boolean, required): Whether to cancel in-progress runs

### Annotation
Represents a GitHub Actions annotation for check results.

**Properties**:
- `level` (enum: "error" | "warning" | "notice", required): Annotation severity
- `message` (string, required): Annotation message
- `file` (string, optional): File path (for file-level annotations)
- `line` (integer, optional): Line number (for line-level annotations)

## Relationships

- **PipelineJob** → **CheckResult** (one-to-many): A job contains multiple check results
- **PipelineJob** → **PipelineJob** (many-to-many via dependencies): Jobs depend on other jobs
- **ReleaseArtifact** → **WorkflowConfiguration** (many-to-one): Releases are created by release workflow
- **WorkflowConfiguration** → **PipelineJob** (one-to-many): A workflow contains multiple jobs
- **WorkflowConfiguration** → **WorkflowInput** (one-to-many): A workflow can have multiple manual trigger inputs
- **WorkflowInput** → **PipelineJob** (many-to-many via conditionals): Inputs control which jobs execute

## Validation Rules Summary

### Workflow-Level Rules
1. All workflow files MUST be valid YAML
2. All workflows MUST have `name`, `on`, and `jobs` fields
3. All jobs MUST have `runs-on` and `steps` fields
4. Job dependencies MUST not form cycles

### Job-Level Rules
1. Critical jobs (security, tests, build) MUST NOT have `continue-on-error: true`
2. Advisory jobs (policy, lint, coverage) MUST have `continue-on-error: true`
3. Coverage checks MUST emit warnings (not errors) if below 80% threshold
4. Security checks MUST fail pipeline on any security issue

### Release-Level Rules
1. Release version tags MUST follow semantic versioning (vX.Y.Z)
2. Documentation MUST be generated before website update
3. Website MUST be updated before release publication
4. Only one release process MUST run at a time (concurrency control)

