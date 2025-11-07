# Test Analyzer - Quick Start Guide

## Generate All Reports

```bash
./scripts/generate-test-reports.sh
```

This will:
- ✅ Analyze all 4 packages (llms, memory, orchestration, agents)
- ✅ Generate detailed markdown reports for each package
- ✅ Create a summary report with cross-package comparison
- ✅ Show progress and statistics

**Output Location**: `reports/test-analyzer/`

## View Dashboard

```bash
# Open in default browser
open reports/test-analyzer/dashboard.html

# Or on Linux
xdg-open reports/test-analyzer/dashboard.html

# Or manually navigate to the file
```

## Analyze Single Package

```bash
# Analyze specific package
./bin/test-analyzer --dry-run --output markdown \
  --output-file reports/test-analyzer/custom_report.md pkg/llms

# Get JSON output
./bin/test-analyzer --dry-run --output json pkg/llms

# Get plain text output
./bin/test-analyzer --dry-run --output plain pkg/llms
```

## Report Files

- **`dashboard.html`** - Interactive visual dashboard
- **`analysis_summary.md`** - Cross-package summary
- **`*_detailed_report.md`** - Individual package reports

## Common Workflows

### Daily Check
```bash
./scripts/generate-test-reports.sh
open reports/test-analyzer/dashboard.html
```

### Before Committing
```bash
./scripts/generate-test-reports.sh
# Review reports/test-analyzer/analysis_summary.md
```

### CI/CD Integration
```bash
# In your CI script
./scripts/generate-test-reports.sh
# Upload reports/test-analyzer/ as artifact
```

## Troubleshooting

**Script fails:**
- Ensure Go is installed: `go version`
- Build analyzer manually: `go build -o bin/test-analyzer ./cmd/test-analyzer`

**Reports not updating:**
- Check file permissions: `chmod +x scripts/generate-test-reports.sh`
- Verify analyzer exists: `ls -lh bin/test-analyzer`

**Dashboard not displaying:**
- Open in a modern browser (Chrome, Firefox, Safari, Edge)
- Check browser console for errors
- Verify HTML file is valid

## Next Steps

1. Review `analysis_summary.md` for overall status
2. Check individual package reports for details
3. View `dashboard.html` for visual overview
4. Prioritize fixes based on severity

