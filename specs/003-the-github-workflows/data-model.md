# Data Model: GitHub Workflows

**Feature**: Fix GitHub Workflows, Coverage, PR Checks, and Documentation Generation  
**Date**: 2025-01-27

## Entities

### Workflow Configuration
**Purpose**: Represents a GitHub Actions workflow file

**Attributes**:
- `name`: Workflow name (string, required)
- `on`: Trigger configuration (object, required)
  - `push`: Branch/tag triggers (object, optional)
  - `pull_request`: PR triggers (object, optional)
  - `workflow_dispatch`: Manual trigger (object, optional)
- `jobs`: Job definitions (object, required)
  - Each job has: `runs-on`, `steps`, `needs` (optional)

**Validation Rules**:
- Must have at least one job
- Each job must have `runs-on` and `steps`
- Trigger configuration must be valid YAML

### Test Coverage Report
**Purpose**: Represents test coverage calculation results

**Attributes**:
- `coverage_file`: Path to coverage file (string, required, e.g., "coverage.unit.out")
- `total_coverage`: Overall coverage percentage (float, 0-100, required)
- `package_coverage`: Per-package coverage map (map[string]float, optional)
- `packages_without_tests`: List of packages with 0% coverage ([]string, optional)
- `calculation_timestamp`: When coverage was calculated (timestamp, required)

**Validation Rules**:
- `total_coverage` must be >= 0 and <= 100
- Packages without tests must be included in calculation with 0% coverage
- Coverage file must exist and be valid Go coverage format
- Minimum threshold: 80% (enforced)

**State Transitions**:
- `not_calculated` → `calculating` → `calculated` → `validated` | `below_threshold`
- If `below_threshold`: Can transition to `override_allowed` (for advisory checks)

### Release Configuration
**Purpose**: Represents release workflow configuration

**Attributes**:
- `trigger_type`: How release was triggered (enum: "automated" | "manual" | "tag")
- `version`: Release version (string, required, format: "vX.Y.Z")
- `release_notes`: Release notes content (string, optional)
- `artifacts`: List of artifacts to release ([]string, optional)
- `prerelease`: Whether this is a prerelease (boolean, default: false)

**Validation Rules**:
- Version must match semantic versioning (vX.Y.Z)
- Automated releases must have valid version from release-please
- Manual releases require explicit version input
- Tag releases extract version from tag name

**State Transitions**:
- `pending` → `validating` → `building` → `releasing` → `released` | `failed`
- Only one release can be in `releasing` state at a time (conflict prevention)

### PR Check Status
**Purpose**: Represents the status of a PR check

**Attributes**:
- `check_name`: Name of the check (string, required)
- `status`: Current status (enum: "queued" | "in_progress" | "completed")
- `conclusion`: Final result (enum: "success" | "failure" | "neutral" | "cancelled" | "skipped" | "timed_out" | "action_required")
- `check_type`: Type of check (enum: "critical" | "advisory")
- `details`: Additional details/messages (string, optional)
- `override_allowed`: Whether override is allowed for failures (boolean, default: false for critical)

**Validation Rules**:
- Critical checks with "failure" conclusion block PR merge (unless overridden)
- Advisory checks with "failure" conclusion show warning but don't block
- `override_allowed` can only be true for critical checks
- Status must progress: queued → in_progress → completed

**State Transitions**:
- `queued` → `in_progress` → `completed` (with conclusion)
- If `conclusion == "failure"` and `check_type == "critical"`: Requires override to merge
- If `conclusion == "failure"` and `check_type == "advisory"`: Warning only, merge allowed

### Documentation Artifact
**Purpose**: Represents generated API documentation

**Attributes**:
- `package_name`: Package being documented (string, required)
- `output_path`: Where documentation is generated (string, required)
- `generation_timestamp`: When docs were generated (timestamp, required)
- `source_package_path`: Source package path (string, required)
- `format`: Documentation format (enum: "markdown" | "mdx", default: "markdown")
- `frontmatter`: YAML frontmatter for website (object, optional)

**Validation Rules**:
- Output path must be within `website/docs/api/packages/`
- Package must exist in source code
- Generated file must be valid Markdown/MDX
- Frontmatter must include `title` and `sidebar_position`

**State Transitions**:
- `pending` → `generating` → `generated` → `deployed` | `failed`
- Generation must complete before deployment

## Relationships

- **Workflow Configuration** → **PR Check Status**: One workflow can produce multiple check statuses
- **Test Coverage Report** → **PR Check Status**: Coverage check produces a check status
- **Release Configuration** → **Workflow Configuration**: Release workflow uses workflow configuration
- **Documentation Artifact** → **Package**: Each artifact documents one package

## Constraints

1. **Coverage Calculation**: Packages without tests must be included with 0% coverage
2. **Coverage Threshold**: Overall coverage must be >= 80% (enforced for critical check)
3. **Release Conflicts**: Only one release can be in progress at a time
4. **Documentation Generation**: Must run on main branch merges only
5. **Check Types**: Critical checks block merge, advisory checks show warnings

## Data Flow

1. **Coverage Calculation Flow**:
   ```
   Tests Run → coverage.unit.out Generated → Parse Coverage → Calculate Total → Validate Threshold → PR Check Status
   ```

2. **Release Flow**:
   ```
   Trigger (automated/manual/tag) → Validate Version → Build → Release → Create GitHub Release
   ```

3. **Documentation Flow**:
   ```
   Merge to Main → Generate Docs → Validate Output → Build Website → Deploy
   ```

4. **PR Check Flow**:
   ```
   PR Opened → Workflows Triggered → Checks Run → Status Reported → Merge Decision (block/warn/allow)
   ```

