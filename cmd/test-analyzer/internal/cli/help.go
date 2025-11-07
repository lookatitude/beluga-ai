package cli

import (
	"fmt"
	"os"
)

// PrintHelp prints the help message.
func PrintHelp() {
	helpText := `test-analyzer - Identify and fix long-running test issues

USAGE:
    test-analyzer [flags] [packages...]

DESCRIPTION:
    Analyzes Go test files for performance issues and optionally applies fixes.

ARGUMENTS:
    packages    List of package paths to analyze. If not provided, analyzes all
                packages in pkg/ directory.

FLAGS:
    Analysis Flags:
      --dry-run              Analyze without applying fixes (default: true)
      --output FORMAT        Output format: stdout, json, html, markdown (default: stdout)
      --output-file PATH     File path for output (if not stdout)
      --include-benchmarks   Include benchmark tests in analysis (default: true)
      --exclude-packages     Comma-separated list of packages to exclude

    Fix Flags:
      --auto-fix             Automatically apply fixes without confirmation
      --fix-types TYPES      Comma-separated list of fix types to apply
      --skip-validation      Skip validation (not recommended)
      --backup-dir DIR       Directory for backups (default: .test-analyzer-backups)

    Threshold Flags:
      --unit-test-timeout DURATION        Maximum execution time for unit tests (default: 1s)
      --integration-test-timeout DURATION Maximum execution time for integration tests (default: 10s)
      --load-test-timeout DURATION        Maximum execution time for load tests (default: 30s)
      --simple-iteration-threshold N      Iteration threshold for simple operations (default: 100)
      --complex-iteration-threshold N     Iteration threshold for complex operations (default: 20)
      --sleep-threshold DURATION          Total sleep duration threshold per test (default: 100ms)

    Filter Flags:
      --severity LEVEL       Minimum severity to report: low, medium, high, critical
      --issue-types TYPES    Comma-separated list of issue types to include
      --packages PATHS       Comma-separated list of packages to analyze

    Output Flags:
      --verbose              Verbose output
      --quiet                Quiet mode (errors only)
      --color                Colorized output (default: true)

    Help:
      --help, -h             Show this help message

EXIT CODES:
    0    Success
    1    Analysis errors
    2    Fix application errors
    3    Validation errors
    4    Invalid arguments

EXAMPLES:
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
`

	fmt.Fprint(os.Stdout, helpText)
}

