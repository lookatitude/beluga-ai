# Quickstart Guide: Test Analyzer Tool

## Overview
This guide demonstrates how to use the test analyzer tool to identify and fix long-running test issues in the Beluga AI Framework.

## Prerequisites
- Go 1.24.0 or later
- Access to the Beluga AI Framework repository
- Tests should be runnable (dependencies installed)

## Installation

### Build the Tool
```bash
cd /home/miguelp/Projects/lookatitude/beluga-ai
go build -o bin/test-analyzer ./cmd/test-analyzer
```

### Verify Installation
```bash
./bin/test-analyzer --help
```

## Basic Usage

### Step 1: Dry Run Analysis
Start with a dry run to see what issues exist without making changes:

```bash
./bin/test-analyzer --dry-run
```

**Expected Output**:
- List of packages analyzed
- Summary statistics (files, functions, issues found)
- Categorized issues by type and severity
- Recommendations for fixes

**Example Output**:
```
Analyzing test suite...
Packages analyzed: 14
Files analyzed: 120
Functions analyzed: 500
Issues found: 45

Issues by type:
  InfiniteLoop: 2
  MissingTimeout: 15
  LargeIteration: 8
  ActualImplementationUsage: 20

Issues by severity:
  Critical: 2
  High: 18
  Medium: 20
  Low: 5

Top problematic packages:
  1. pkg/llms: 10 issues
  2. pkg/memory: 8 issues
  3. pkg/agents: 7 issues
```

### Step 2: Analyze Specific Package
Focus on a specific package:

```bash
./bin/test-analyzer --dry-run pkg/llms
```

**Expected Output**:
- Detailed analysis for `pkg/llms` package
- File-level and function-level issues
- Specific line numbers and code patterns

### Step 3: Generate Report
Generate a detailed report in JSON format:

```bash
./bin/test-analyzer --dry-run --output json --output-file report.json
```

**Expected Result**:
- `report.json` file created with complete analysis data
- Can be used for programmatic processing or integration with CI/CD

### Step 4: Review Issues
Examine the report to understand what needs to be fixed:

```bash
cat report.json | jq '.issues_by_type'
cat report.json | jq '.packages[] | select(.issues > 5)'
```

## Applying Fixes

### Step 5: Auto-Fix with Specific Types
Apply fixes for specific issue types:

```bash
./bin/test-analyzer --auto-fix --fix-types AddTimeout,ReplaceWithMock
```

**Expected Behavior**:
- Analyzes all packages
- Applies fixes for timeout and mock replacement issues
- Creates backups before modifications
- Validates fixes (interface compatibility + test execution)
- Reports success/failure for each fix

**Expected Output**:
```
Applying fixes...
  pkg/llms/llms_test.go: Added timeout to TestLLMCall [SUCCESS]
  pkg/memory/memory_test.go: Replaced real implementation with mock [SUCCESS]
  pkg/agents/agents_test.go: Added timeout to TestAgentRun [SUCCESS]
  
Fixes applied: 30
Fixes failed: 2
Validation passed: 28
```

### Step 6: Verify Fixes
Run tests to verify fixes work:

```bash
go test ./pkg/llms/... -v
go test ./pkg/memory/... -v
```

**Expected Result**:
- Tests pass
- Execution time improved
- No regressions

### Step 7: Generate Post-Fix Report
Generate a report after fixes to see improvements:

```bash
./bin/test-analyzer --dry-run --output json --output-file report-after.json
```

Compare before and after:
```bash
diff <(jq '.summary' report.json) <(jq '.summary' report-after.json)
```

## Advanced Usage

### Custom Thresholds
Adjust thresholds for your specific needs:

```bash
./bin/test-analyzer --dry-run \
  --unit-test-timeout 500ms \
  --simple-iteration-threshold 50 \
  --complex-iteration-threshold 15
```

### Filter by Severity
Only report high-severity issues:

```bash
./bin/test-analyzer --dry-run --severity high
```

### Exclude Packages
Skip certain packages from analysis:

```bash
./bin/test-analyzer --dry-run --exclude-packages pkg/monitoring,pkg/server
```

### Generate HTML Report
Create an interactive HTML report:

```bash
./bin/test-analyzer --dry-run --output html --output-file report.html
open report.html  # macOS
xdg-open report.html  # Linux
```

## Integration with CI/CD

### Pre-commit Hook
Add to `.git/hooks/pre-commit`:

```bash
#!/bin/bash
./bin/test-analyzer --dry-run --severity critical
if [ $? -ne 0 ]; then
  echo "Critical test performance issues detected!"
  exit 1
fi
```

### CI Pipeline
Add to CI configuration (e.g., `.github/workflows/tests.yml`):

```yaml
- name: Analyze Test Performance
  run: |
    go build -o bin/test-analyzer ./cmd/test-analyzer
    ./bin/test-analyzer --dry-run --output json --output-file test-analysis.json
    
- name: Upload Analysis Report
  uses: actions/upload-artifact@v3
  with:
    name: test-analysis
    path: test-analysis.json
```

## Troubleshooting

### Issue: Tool fails to parse test files
**Solution**: Ensure Go toolchain is properly installed and test files are valid Go code.

### Issue: Fixes break existing tests
**Solution**: The tool should automatically rollback failed fixes. Check validation output for details.

### Issue: Mock generation fails
**Solution**: Check that interface definitions are accessible and valid. May need to generate template mocks for complex cases.

### Issue: Analysis takes too long
**Solution**: Use `--exclude-packages` to skip non-critical packages, or analyze packages individually.

## Success Criteria

After running the tool, you should see:

1. ✅ All critical and high-severity issues identified
2. ✅ Fixes applied successfully (if using `--auto-fix`)
3. ✅ Tests pass after fixes
4. ✅ Execution time improved
5. ✅ No test regressions
6. ✅ Comprehensive report generated

## Next Steps

1. Review generated reports to understand test suite health
2. Apply fixes incrementally (start with critical issues)
3. Integrate tool into CI/CD pipeline
4. Set up regular analysis schedule
5. Monitor test execution times over time

## Example Workflow

Complete workflow from analysis to fix:

```bash
# 1. Initial analysis
./bin/test-analyzer --dry-run --output json --output-file initial.json

# 2. Review critical issues
./bin/test-analyzer --dry-run --severity critical

# 3. Apply fixes for critical issues
./bin/test-analyzer --auto-fix --fix-types AddTimeout --severity critical

# 4. Verify fixes
go test ./pkg/... -timeout 5m

# 5. Generate final report
./bin/test-analyzer --dry-run --output json --output-file final.json

# 6. Compare results
diff <(jq '.summary.issues_found' initial.json) <(jq '.summary.issues_found' final.json)
```

## Validation Test

Run this validation to ensure the tool works correctly:

```bash
# Test 1: Dry run should complete successfully
./bin/test-analyzer --dry-run pkg/core
if [ $? -ne 0 ]; then
  echo "FAIL: Dry run failed"
  exit 1
fi

# Test 2: JSON output should be valid
./bin/test-analyzer --dry-run --output json --output-file test.json pkg/core
jq . test.json > /dev/null
if [ $? -ne 0 ]; then
  echo "FAIL: Invalid JSON output"
  exit 1
fi

# Test 3: Help should display
./bin/test-analyzer --help | grep -q "test-analyzer"
if [ $? -ne 0 ]; then
  echo "FAIL: Help not displayed"
  exit 1
fi

echo "SUCCESS: All validation tests passed"
```

