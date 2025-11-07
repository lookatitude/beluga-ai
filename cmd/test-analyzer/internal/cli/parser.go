package cli

import (
	"flag"
	"fmt"
	"strings"
	"time"
)

// ParseFlags parses command-line flags and returns a Config.
func ParseFlags(args []string) (*Config, error) {
	fs := flag.NewFlagSet("test-analyzer", flag.ContinueOnError)

	config := &Config{}

	// Analysis flags
	fs.BoolVar(&config.DryRun, "dry-run", true, "Analyze without applying fixes")
	fs.StringVar(&config.Output, "output", "stdout", "Output format: stdout, json, html, markdown")
	fs.StringVar(&config.OutputFile, "output-file", "", "File path for output (if not stdout)")
	fs.BoolVar(&config.IncludeBenchmarks, "include-benchmarks", true, "Include benchmark tests in analysis")
	fs.StringVar(&config.ExcludePackagesStr, "exclude-packages", "", "Comma-separated list of packages to exclude")

	// Fix flags
	fs.BoolVar(&config.AutoFix, "auto-fix", false, "Automatically apply fixes without confirmation")
	fs.StringVar(&config.FixTypesStr, "fix-types", "", "Comma-separated list of fix types to apply")
	fs.BoolVar(&config.SkipValidation, "skip-validation", false, "Skip validation (not recommended)")
	fs.StringVar(&config.BackupDir, "backup-dir", ".test-analyzer-backups", "Directory for backups")

	// Threshold flags
	fs.DurationVar(&config.UnitTestTimeout, "unit-test-timeout", 1*time.Second, "Maximum execution time for unit tests")
	fs.DurationVar(&config.IntegrationTestTimeout, "integration-test-timeout", 10*time.Second, "Maximum execution time for integration tests")
	fs.DurationVar(&config.LoadTestTimeout, "load-test-timeout", 30*time.Second, "Maximum execution time for load tests")
	fs.IntVar(&config.SimpleIterationThreshold, "simple-iteration-threshold", 100, "Iteration threshold for simple operations")
	fs.IntVar(&config.ComplexIterationThreshold, "complex-iteration-threshold", 20, "Iteration threshold for complex operations")
	fs.DurationVar(&config.SleepThreshold, "sleep-threshold", 100*time.Millisecond, "Total sleep duration threshold per test")

	// Filter flags
	fs.StringVar(&config.Severity, "severity", "", "Minimum severity to report: low, medium, high, critical")
	fs.StringVar(&config.IssueTypesStr, "issue-types", "", "Comma-separated list of issue types to include")
	fs.StringVar(&config.PackagesStr, "packages", "", "Comma-separated list of packages to analyze")

	// Output flags
	fs.BoolVar(&config.Verbose, "verbose", false, "Verbose output")
	fs.BoolVar(&config.Quiet, "quiet", false, "Quiet mode (errors only)")
	fs.BoolVar(&config.Color, "color", true, "Colorized output")

	// Help flag
	fs.BoolVar(&config.Help, "help", false, "Show help message")
	fs.BoolVar(&config.Help, "h", false, "Show help message (short)")

	// Parse flags
	if err := fs.Parse(args); err != nil {
		return nil, fmt.Errorf("parsing flags: %w", err)
	}

	// Parse comma-separated strings
	if config.ExcludePackagesStr != "" {
		config.ExcludePackages = strings.Split(config.ExcludePackagesStr, ",")
	}
	if config.FixTypesStr != "" {
		config.FixTypes = strings.Split(config.FixTypesStr, ",")
	}
	if config.IssueTypesStr != "" {
		config.IssueTypes = strings.Split(config.IssueTypesStr, ",")
	}
	if config.PackagesStr != "" {
		config.Packages = strings.Split(config.PackagesStr, ",")
	}

	// Get remaining positional arguments (package paths)
	config.PackagePaths = fs.Args()

	// If --auto-fix is set, automatically disable dry-run
	if config.AutoFix {
		config.DryRun = false
	}

	return config, nil
}

// Config holds configuration for the test analyzer.
type Config struct {
	// Analysis flags
	DryRun           bool
	Output           string
	OutputFile       string
	IncludeBenchmarks bool
	ExcludePackages  []string
	ExcludePackagesStr string // Internal: for parsing

	// Fix flags
	AutoFix         bool
	FixTypes        []string
	FixTypesStr     string // Internal: for parsing
	SkipValidation  bool
	BackupDir       string

	// Threshold flags
	UnitTestTimeout       time.Duration
	IntegrationTestTimeout time.Duration
	LoadTestTimeout       time.Duration
	SimpleIterationThreshold int
	ComplexIterationThreshold int
	SleepThreshold        time.Duration

	// Filter flags
	Severity   string
	IssueTypes []string
	IssueTypesStr string // Internal: for parsing
	Packages   []string
	PackagesStr string // Internal: for parsing

	// Output flags
	Verbose bool
	Quiet   bool
	Color   bool

	// Help flag
	Help bool

	// Package paths from positional arguments
	PackagePaths []string
}

