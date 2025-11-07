# CLI Interface Contract

## Command: `test-analyzer`

### Overview
CLI tool for analyzing and fixing long-running test issues in Go packages.

### Usage
```bash
test-analyzer [flags] [packages...]
```

### Arguments
- `packages` (optional): List of package paths to analyze. If not provided, analyzes all packages in `pkg/` directory.

### Flags

#### Analysis Flags
- `--dry-run` (bool, default: false): Analyze without applying fixes
- `--output` (string, default: "stdout"): Output format: "stdout", "json", "html", "markdown"
- `--output-file` (string): File path for output (if not stdout)
- `--include-benchmarks` (bool, default: true): Include benchmark tests in analysis
- `--exclude-packages` (string): Comma-separated list of packages to exclude

#### Fix Flags
- `--auto-fix` (bool, default: false): Automatically apply fixes without confirmation
- `--fix-types` (string): Comma-separated list of fix types to apply (e.g., "AddTimeout,ReplaceWithMock")
- `--skip-validation` (bool, default: false): Skip validation (not recommended)
- `--backup-dir` (string, default: ".test-analyzer-backups"): Directory for backups

#### Threshold Flags
- `--unit-test-timeout` (duration, default: "1s"): Maximum execution time for unit tests
- `--integration-test-timeout` (duration, default: "10s"): Maximum execution time for integration tests
- `--load-test-timeout` (duration, default: "30s"): Maximum execution time for load tests
- `--simple-iteration-threshold` (int, default: 100): Iteration threshold for simple operations
- `--complex-iteration-threshold` (int, default: 20): Iteration threshold for complex operations
- `--sleep-threshold` (duration, default: "100ms"): Total sleep duration threshold per test

#### Filter Flags
- `--severity` (string): Minimum severity to report: "low", "medium", "high", "critical"
- `--issue-types` (string): Comma-separated list of issue types to include
- `--packages` (string): Comma-separated list of packages to analyze

#### Output Flags
- `--verbose` (bool, default: false): Verbose output
- `--quiet` (bool, default: false): Quiet mode (errors only)
- `--color` (bool, default: true): Colorized output

### Exit Codes
- `0`: Success
- `1`: Analysis errors
- `2`: Fix application errors
- `3`: Validation errors
- `4`: Invalid arguments

### Examples

```bash
# Analyze all packages (dry run)
test-analyzer --dry-run

# Analyze specific packages and output JSON
test-analyzer --output json --output-file report.json pkg/llms pkg/memory

# Auto-fix with specific fix types
test-analyzer --auto-fix --fix-types AddTimeout,ReplaceWithMock

# Analyze with custom thresholds
test-analyzer --unit-test-timeout 500ms --simple-iteration-threshold 50

# Generate HTML report
test-analyzer --output html --output-file report.html
```

