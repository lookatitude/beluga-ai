# Test Analyzer Scripts

This directory contains utility scripts for the test-analyzer tool.

## Scripts

### `generate-test-reports.sh`

Automatically generates test analysis reports for all packages and creates a summary.

**Usage:**
```bash
./scripts/generate-test-reports.sh
```

**What it does:**
1. Builds the test-analyzer if needed
2. Analyzes all configured packages (pkg/llms, pkg/memory, pkg/orchestration, pkg/agents)
3. Generates detailed markdown reports for each package
4. Creates a summary report with cross-package comparison
5. Outputs progress and statistics

**Output:**
- Individual package reports in `reports/test-analyzer/`
- Summary report: `reports/test-analyzer/analysis_summary.md`
- Dashboard: `reports/test-analyzer/dashboard.html`

**Configuration:**
Edit the `PACKAGES` array in the script to add/remove packages.

### `update-dashboard.sh`

Updates dashboard with current analysis data (helper script).

**Usage:**
```bash
./scripts/update-dashboard.sh
```

**Note:** Currently, the dashboard uses static data. For dynamic updates, consider using JavaScript to load JSON data or a templating engine.

## Integration

### CI/CD Integration

Add to your CI pipeline:

```yaml
# Example GitHub Actions
- name: Generate Test Reports
  run: ./scripts/generate-test-reports.sh
  
- name: Upload Reports
  uses: actions/upload-artifact@v3
  with:
    name: test-analysis-reports
    path: reports/test-analyzer/
```

### Pre-commit Hook

Add to `.git/hooks/pre-commit`:

```bash
#!/bin/bash
./scripts/generate-test-reports.sh
```

### Scheduled Reports

Add to crontab for daily reports:

```bash
# Run daily at 2 AM
0 2 * * * cd /path/to/project && ./scripts/generate-test-reports.sh
```

## Requirements

- Go 1.21+
- Python 3 (for JSON parsing)
- `bc` command (for calculations, usually pre-installed)

## Troubleshooting

**Script fails to build analyzer:**
- Ensure Go is installed and in PATH
- Check that `cmd/test-analyzer` exists

**JSON parsing fails:**
- Ensure Python 3 is installed
- Check that analyzer output is valid JSON

**Reports not generated:**
- Check file permissions on reports directory
- Verify analyzer binary exists and is executable

