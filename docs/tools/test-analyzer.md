# test-analyzer - Comprehensive Documentation

## Table of Contents

1. [Introduction](#introduction)
2. [Installation](#installation)
3. [Basic Usage](#basic-usage)
4. [Advanced Usage](#advanced-usage)
5. [All Flags Reference](#all-flags-reference)
6. [Issue Types](#issue-types)
7. [Fix Types](#fix-types)
8. [Report Formats](#report-formats)
9. [Examples](#examples)
10. [Troubleshooting](#troubleshooting)
11. [Architecture](#architecture)

## Introduction

`test-analyzer` is a powerful tool for identifying and fixing performance issues in Go test files. It uses AST (Abstract Syntax Tree) analysis to detect common anti-patterns and can automatically apply fixes with validation.

### Key Features

- **Comprehensive Detection**: Identifies 9+ types of performance issues
- **Safe Auto-Fix**: Applies fixes with dual validation
- **Multiple Formats**: Supports JSON, HTML, Markdown, and Plain text reports
- **Package Support**: Analyzes entire packages or specific files
- **Backup System**: Creates backups before any modifications

## Installation

### From Source

```bash
git clone https://github.com/lookatitude/beluga-ai.git
cd beluga-ai/cmd/test-analyzer
go build -o test-analyzer
```

### Using go install

```bash
go install github.com/lookatitude/beluga-ai/cmd/test-analyzer@latest
```

## Basic Usage

### Analyze a Package

```bash
test-analyzer pkg/llms
```

### Dry Run (Recommended First Step)

```bash
test-analyzer --dry-run pkg/llms
```

### Generate Report to File

```bash
test-analyzer --dry-run --output json --output-file report.json pkg/llms
```

## Advanced Usage

### Auto-Fix with Validation

```bash
test-analyzer --auto-fix pkg/llms
```

This will:
1. Analyze the package
2. Apply fixes for detected issues
3. Validate fixes (interface compatibility + test execution)
4. Create backups before modifications

### Filter by Severity

```bash
# Only show high and critical issues
test-analyzer --severity high pkg/llms
```

### Filter by Issue Type

```bash
# Only detect infinite loops and missing timeouts
test-analyzer --issue-types InfiniteLoop,MissingTimeout pkg/llms
```

### Custom Thresholds

```bash
test-analyzer \
  --simple-iteration-threshold 200 \
  --complex-iteration-threshold 50 \
  --sleep-threshold 200ms \
  pkg/llms
```

## All Flags Reference

### Analysis Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--dry-run` | bool | `true` | Analyze without applying fixes |
| `--output` | string | `stdout` | Output format: `stdout`, `json`, `html`, `markdown`, `plain` |
| `--output-file` | string | `""` | File path for output (if not stdout) |
| `--include-benchmarks` | bool | `true` | Include benchmark tests in analysis |
| `--exclude-packages` | string | `""` | Comma-separated list of packages to exclude |

### Fix Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--auto-fix` | bool | `false` | Automatically apply fixes without confirmation |
| `--fix-types` | string | `""` | Comma-separated list of fix types to apply |
| `--skip-validation` | bool | `false` | Skip validation (not recommended) |
| `--backup-dir` | string | `.test-analyzer-backups` | Directory for backups |

### Threshold Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--unit-test-timeout` | duration | `1s` | Maximum execution time for unit tests |
| `--integration-test-timeout` | duration | `10s` | Maximum execution time for integration tests |
| `--load-test-timeout` | duration | `30s` | Maximum execution time for load tests |
| `--simple-iteration-threshold` | int | `100` | Iteration threshold for simple operations |
| `--complex-iteration-threshold` | int | `20` | Iteration threshold for complex operations |
| `--sleep-threshold` | duration | `100ms` | Total sleep duration threshold per test |

### Filter Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--severity` | string | `""` | Minimum severity: `low`, `medium`, `high`, `critical` |
| `--issue-types` | string | `""` | Comma-separated list of issue types to include |
| `--packages` | string | `""` | Comma-separated list of packages to analyze |

### Output Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--verbose` | bool | `false` | Verbose output |
| `--quiet` | bool | `false` | Quiet mode (errors only) |
| `--color` | bool | `true` | Colorized output |

## Issue Types

### InfiniteLoop

**Severity**: Critical  
**Description**: Infinite loop without exit condition  
**Example**:
```go
func TestInfinite(t *testing.T) {
    for {
        // No exit condition
    }
}
```

### MissingTimeout

**Severity**: High (Unit), Medium (Integration), Low (Load)  
**Description**: Test function missing timeout mechanism  
**Example**:
```go
func TestLongRunning(t *testing.T) {
    // Missing context.WithTimeout or time.After
    select {} // Could hang forever
}
```

### LargeIteration

**Severity**: Medium (Simple), High (Complex)  
**Description**: Loop with excessive iteration count  
**Example**:
```go
func TestLargeLoop(t *testing.T) {
    for i := 0; i < 1000; i++ { // Exceeds threshold
        // Simple operations
    }
}
```

### HighConcurrency

**Severity**: High  
**Description**: Complex operations (network, I/O, DB) in loops  
**Example**:
```go
func TestComplexLoop(t *testing.T) {
    for i := 0; i < 50; i++ {
        http.Get("https://example.com") // Network call in loop
    }
}
```

### SleepDelay

**Severity**: Medium (<1s), High (>1s)  
**Description**: Accumulated sleep delays exceeding threshold  
**Example**:
```go
func TestWithSleep(t *testing.T) {
    time.Sleep(50 * time.Millisecond)
    time.Sleep(30 * time.Millisecond)
    time.Sleep(40 * time.Millisecond) // Total > 100ms threshold
}
```

### ActualImplementationUsage

**Severity**: High  
**Description**: Unit test using actual implementation instead of mocks  
**Example**:
```go
func TestUnit(t *testing.T) {
    client := http.Client{} // Should use mock
    client.Get("https://example.com")
}
```

### MixedMockRealUsage

**Severity**: Medium  
**Description**: Unit test mixing mocks and real implementations  
**Example**:
```go
func TestMixed(t *testing.T) {
    mockClient := NewMockClient()
    realDB := sql.Open(...) // Mixed usage
}
```

### MissingMock

**Severity**: Medium  
**Description**: Unit test using interface but no mock found  
**Example**:
```go
func TestUnit(t *testing.T) {
    var client HTTPClient // Interface used but no mock
    client.Get("https://example.com")
}
```

### BenchmarkHelperUsage

**Severity**: Low  
**Description**: Regular test using benchmark helpers  
**Example**:
```go
func TestRegular(t *testing.T) {
    b := &testing.B{}
    b.ResetTimer() // Benchmark helper in regular test
}
```

## Fix Types

### AddTimeout

Adds `context.WithTimeout` to test functions missing timeout mechanisms.

**Before**:
```go
func TestLongRunning(t *testing.T) {
    select {}
}
```

**After**:
```go
func TestLongRunning(t *testing.T) {
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    select {
    case <-ctx.Done():
        return
    }
}
```

### ReduceIterations

Reduces excessive iteration counts in loops.

**Before**:
```go
for i := 0; i < 1000; i++ {
    // Simple operations
}
```

**After**:
```go
for i := 0; i < 100; i++ { // Reduced to threshold
    // Simple operations
}
```

### OptimizeSleep

Removes or reduces sleep durations.

**Before**:
```go
time.Sleep(50 * time.Millisecond)
time.Sleep(30 * time.Millisecond)
```

**After**:
```go
// Sleep calls removed or reduced
```

### AddLoopExit

Adds exit conditions to infinite loops.

**Before**:
```go
for {
    // No exit
}
```

**After**:
```go
for {
    if done {
        break
    }
}
```

### ReplaceWithMock

Replaces actual implementations with mocks in unit tests.

**Before**:
```go
func TestUnit(t *testing.T) {
    client := http.Client{}
    client.Get("https://example.com")
}
```

**After**:
```go
func TestUnit(t *testing.T) {
    client := NewMockHTTPClient()
    client.Get("https://example.com")
}
```

### CreateMock

Generates and inserts missing mock implementations.

## Report Formats

### JSON Format

Structured JSON suitable for programmatic processing:

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
  },
  "issues_by_severity": {
    "Critical": 2,
    "High": 8,
    "Medium": 5
  },
  "packages": [...]
}
```

### HTML Format

Interactive HTML report with:
- Charts and visualizations (Chart.js)
- Color-coded severity indicators
- Expandable sections
- Package comparison tables
- Issue breakdown by type and severity

### Markdown Format

Markdown report with:
- Code blocks for examples
- Tables for statistics
- Section headers for organization
- Issue details with locations

### Plain Text Format

Terminal-friendly output:
- Color support (when enabled)
- Simple formatting
- Easy to read in terminal
- Summary statistics

## Examples

### Example 1: Complete Analysis Workflow

```bash
# Step 1: Analyze with dry-run
test-analyzer --dry-run --output json --output-file initial.json pkg/llms

# Step 2: Review issues
cat initial.json | jq '.issues_by_type'

# Step 3: Auto-fix high severity issues
test-analyzer --auto-fix --severity high pkg/llms

# Step 4: Generate final report
test-analyzer --dry-run --output html --output-file final.html pkg/llms
```

### Example 2: CI/CD Integration

```bash
#!/bin/bash
# CI script example

# Analyze all packages
test-analyzer --dry-run --output json --output-file ci-report.json \
  pkg/llms pkg/memory pkg/orchestration

# Check if issues found
ISSUES=$(jq '.issues_found' ci-report.json)
if [ "$ISSUES" -gt 0 ]; then
    echo "Found $ISSUES issues"
    exit 1
fi
```

### Example 3: Selective Fixing

```bash
# Only fix timeout and mock issues
test-analyzer --auto-fix \
  --fix-types AddTimeout,ReplaceWithMock \
  pkg/llms
```

### Example 4: Custom Thresholds

```bash
# Customize thresholds for your project
test-analyzer \
  --simple-iteration-threshold 200 \
  --complex-iteration-threshold 50 \
  --sleep-threshold 200ms \
  --unit-test-timeout 2s \
  pkg/llms
```

## Troubleshooting

### Common Issues

#### "No test files found"

**Cause**: Directory doesn't contain `*_test.go` files  
**Solution**: Ensure you're pointing to a directory with test files

#### "Permission denied"

**Cause**: Insufficient file permissions  
**Solution**: Check read/write permissions for the target directory

#### "Fix validation failed"

**Cause**: Fix broke interface compatibility or tests  
**Solution**: 
1. Check validation output for details
2. Review the applied fix
3. Use `--skip-validation` only if necessary (not recommended)

#### "Backup directory creation failed"

**Cause**: Insufficient permissions or disk space  
**Solution**: 
1. Check write permissions
2. Ensure sufficient disk space
3. Specify custom backup directory with `--backup-dir`

#### "Tool hangs during analysis"

**Cause**: Analyzing very large codebase or infinite loop in test  
**Solution**:
1. Use `--exclude-packages` to skip problematic packages
2. Analyze packages individually
3. Check for infinite loops in test files

### Debug Mode

Enable verbose output for debugging:

```bash
test-analyzer --verbose --dry-run pkg/llms
```

## Architecture

### Component Overview

```
┌─────────────┐
│   CLI       │
└──────┬──────┘
       │
       ▼
┌─────────────┐     ┌──────────────┐     ┌─────────────┐
│  Analyzer   │────▶│   Parser     │────▶│   Detector   │
└──────┬──────┘     └──────────────┘     └─────────────┘
       │
       ▼
┌─────────────┐     ┌──────────────┐     ┌─────────────┐
│   Fixer     │────▶│ Mock Generator│────▶│  Reporter   │
└─────────────┘     └──────────────┘     └─────────────┘
```

### Data Flow

1. **Parsing**: AST parser extracts test functions from files
2. **Detection**: Pattern detectors analyze functions for issues
3. **Analysis**: Analyzer aggregates results across packages
4. **Fixing**: Fixer applies fixes with validation
5. **Reporting**: Reporter generates output in requested format

### Pattern Detection

Patterns are detected using AST analysis:

- **AST Traversal**: Uses `go/ast` to traverse code structure
- **Pattern Matching**: Identifies specific code patterns
- **Context Analysis**: Considers surrounding code context
- **Type Analysis**: Uses `go/types` for interface analysis

### Fix Application

Fixes are applied safely:

1. **Backup Creation**: Timestamped backup created
2. **Code Modification**: Changes applied using `go/format`
3. **Dual Validation**: 
   - Interface compatibility check (for mocks)
   - Test execution verification
4. **Rollback**: Automatic rollback on validation failure

## Best Practices

1. **Always start with dry-run**: Analyze before fixing
2. **Review fixes**: Check validation output before accepting
3. **Use version control**: Commit before auto-fix
4. **Test after fixes**: Run tests to verify fixes work
5. **Customize thresholds**: Adjust thresholds for your project needs
6. **Filter appropriately**: Use severity/type filters to focus on important issues

## See Also

- [Quick Start Guide](../../specs/001-go-package-by/quickstart.md)
- [CLI Interface Documentation](../../specs/001-go-package-by/contracts/cli-interface.md)
- [Main README](../../README.md)

