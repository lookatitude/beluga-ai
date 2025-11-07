# test-analyzer

A comprehensive Go tool for analyzing test files and detecting performance issues, automatically fixing them, and generating detailed reports.

## Overview

The `test-analyzer` tool helps identify and fix common performance anti-patterns in Go test files, including:

- **Infinite loops** without exit conditions
- **Missing timeout mechanisms** in test functions
- **Large iteration counts** in loops
- **Sleep delays** that slow down tests
- **Actual implementation usage** instead of mocks in unit tests
- **Missing mock implementations**
- **Benchmark helper misuse** in regular tests

## Features

- 🔍 **Pattern Detection**: Automatically detects 9+ performance issue types
- 🔧 **Auto-Fix**: Applies fixes with dual validation (interface compatibility + test execution)
- 📊 **Multiple Report Formats**: JSON, HTML, Markdown, and Plain text
- 🎯 **Package-Level Analysis**: Analyze entire packages or specific files
- ✅ **Safe Fixes**: Creates backups and validates fixes before applying
- 🚀 **Fast**: Efficient AST-based analysis

## Installation

```bash
go install github.com/lookatitude/beluga-ai/cmd/test-analyzer@latest
```

Or build from source:

```bash
cd cmd/test-analyzer
go build -o test-analyzer
```

## Quick Start

### Analyze a Package (Dry Run)

```bash
test-analyzer --dry-run pkg/llms
```

### Generate JSON Report

```bash
test-analyzer --dry-run --output json --output-file report.json pkg/llms
```

### Auto-Fix Issues

```bash
test-analyzer --auto-fix pkg/llms
```

### Analyze Multiple Packages

```bash
test-analyzer --dry-run pkg/llms pkg/memory pkg/orchestration
```

## Usage

### Basic Analysis

```bash
# Analyze a single package
test-analyzer pkg/llms

# Analyze with specific output format
test-analyzer --output html pkg/llms

# Dry run (analyze without applying fixes)
test-analyzer --dry-run pkg/llms
```

### Auto-Fix Mode

```bash
# Automatically apply fixes (with validation)
test-analyzer --auto-fix pkg/llms

# Apply fixes for specific issue types only
test-analyzer --auto-fix --fix-types AddTimeout,ReplaceWithMock pkg/llms

# Skip validation (not recommended)
test-analyzer --auto-fix --skip-validation pkg/llms
```

### Filtering Options

```bash
# Filter by severity
test-analyzer --severity high pkg/llms

# Filter by issue type
test-analyzer --issue-types InfiniteLoop,MissingTimeout pkg/llms

# Exclude specific packages
test-analyzer --exclude-packages pkg/monitoring pkg/llms pkg/memory
```

### Output Options

```bash
# JSON output
test-analyzer --output json pkg/llms

# HTML report with charts
test-analyzer --output html --output-file report.html pkg/llms

# Markdown report
test-analyzer --output markdown --output-file report.md pkg/llms

# Plain text (terminal-friendly)
test-analyzer --output plain pkg/llms
```

## Command-Line Flags

### Analysis Flags

- `--dry-run`: Analyze without applying fixes (default: true)
- `--output`: Output format: `stdout`, `json`, `html`, `markdown`, `plain` (default: `stdout`)
- `--output-file`: File path for output (if not stdout)
- `--include-benchmarks`: Include benchmark tests in analysis (default: true)
- `--exclude-packages`: Comma-separated list of packages to exclude

### Fix Flags

- `--auto-fix`: Automatically apply fixes without confirmation
- `--fix-types`: Comma-separated list of fix types to apply
- `--skip-validation`: Skip validation (not recommended)
- `--backup-dir`: Directory for backups (default: `.test-analyzer-backups`)

### Threshold Flags

- `--unit-test-timeout`: Maximum execution time for unit tests (default: 1s)
- `--integration-test-timeout`: Maximum execution time for integration tests (default: 10s)
- `--load-test-timeout`: Maximum execution time for load tests (default: 30s)
- `--simple-iteration-threshold`: Iteration threshold for simple operations (default: 100)
- `--complex-iteration-threshold`: Iteration threshold for complex operations (default: 20)
- `--sleep-threshold`: Total sleep duration threshold per test (default: 100ms)

### Filter Flags

- `--severity`: Minimum severity to report: `low`, `medium`, `high`, `critical`
- `--issue-types`: Comma-separated list of issue types to include
- `--packages`: Comma-separated list of packages to analyze

### Output Flags

- `--verbose`: Verbose output
- `--quiet`: Quiet mode (errors only)
- `--color`: Colorized output (default: true)

## Architecture

The tool is organized into several key components:

### Core Components

1. **AST Parser** (`internal/ast`): Parses Go test files and extracts test functions
2. **Pattern Detectors** (`internal/patterns`): Detects performance issues using AST analysis
3. **Analyzer** (`analyzer.go`): Coordinates analysis across packages and files
4. **Fixer** (`fixer.go`): Applies automated fixes with validation
5. **Mock Generator** (`internal/mocks`): Generates mock implementations
6. **Reporter** (`reporter.go`): Generates reports in multiple formats

### Pattern Detectors

- `InfiniteLoopDetector`: Detects infinite loops without exit conditions
- `TimeoutDetector`: Detects missing timeout mechanisms
- `IterationsDetector`: Detects large iteration counts
- `ComplexityDetector`: Detects complex operations in loops
- `SleepDetector`: Detects sleep delays
- `ImplementationDetector`: Detects actual implementation usage
- `MocksDetector`: Detects missing mock implementations
- `BenchmarkDetector`: Detects benchmark helper misuse

### Fix Types

- `AddTimeout`: Adds `context.WithTimeout` to test functions
- `ReduceIterations`: Reduces excessive iteration counts
- `OptimizeSleep`: Removes or reduces sleep durations
- `AddLoopExit`: Adds exit conditions to infinite loops
- `ReplaceWithMock`: Replaces actual implementations with mocks
- `CreateMock`: Generates and inserts missing mock implementations

## Examples

### Example 1: Analyze and Generate HTML Report

```bash
test-analyzer --dry-run --output html --output-file report.html pkg/llms
```

### Example 2: Auto-Fix High Severity Issues Only

```bash
test-analyzer --auto-fix --severity high pkg/llms
```

### Example 3: Analyze Multiple Packages with JSON Output

```bash
test-analyzer --dry-run --output json \
  pkg/llms pkg/memory pkg/orchestration \
  > analysis.json
```

### Example 4: Fix Specific Issue Types

```bash
test-analyzer --auto-fix \
  --fix-types AddTimeout,ReplaceWithMock \
  pkg/llms
```

## Safety Features

- **Backups**: All file modifications create timestamped backups
- **Dual Validation**: Fixes are validated using interface compatibility checks and test execution
- **Dry Run Mode**: Default mode analyzes without modifying files
- **Rollback**: Applied fixes can be rolled back using backups

## Report Formats

### JSON

Structured JSON output suitable for programmatic processing:

```json
{
  "packages_analyzed": 1,
  "files_analyzed": 5,
  "functions_analyzed": 42,
  "issues_found": 15,
  "issues_by_type": {
    "InfiniteLoop": 2,
    "MissingTimeout": 8,
    "LargeIteration": 5
  }
}
```

### HTML

Interactive HTML report with:
- Charts and visualizations
- Color-coded severity indicators
- Expandable sections
- Package comparison tables

### Markdown

Markdown report with:
- Code blocks for examples
- Tables for statistics
- Section headers for organization

### Plain Text

Terminal-friendly output with:
- Color support (when enabled)
- Simple formatting
- Easy to read in terminal

## Troubleshooting

### Issue: "No test files found"

Ensure you're pointing to a directory containing `*_test.go` files.

### Issue: "Permission denied"

Check file permissions. The tool needs read access for analysis and write access for fixes.

### Issue: "Fix validation failed"

The fix may have broken interface compatibility or caused tests to fail. Check the validation output for details.

### Issue: "Backup directory not found"

The tool will create the backup directory automatically. Ensure you have write permissions.

## Contributing

See the main repository README for contribution guidelines.

## License

Part of the Beluga AI Framework. See main repository for license information.

