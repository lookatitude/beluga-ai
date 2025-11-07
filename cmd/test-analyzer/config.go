package main

import (
	"time"
)

// Config holds configuration for the test analyzer.
type Config struct {
	// Analysis flags
	DryRun           bool
	Output           string
	OutputFile       string
	IncludeBenchmarks bool
	ExcludePackages  []string

	// Fix flags
	AutoFix         bool
	FixTypes        []string
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
	Packages   []string

	// Output flags
	Verbose bool
	Quiet   bool
	Color   bool

	// Help flag
	Help bool
}

// DefaultConfig returns a Config with default values.
func DefaultConfig() *Config {
	return &Config{
		DryRun:                  true,
		Output:                  "stdout",
		OutputFile:              "",
		IncludeBenchmarks:        true,
		ExcludePackages:          []string{},
		AutoFix:                 false,
		FixTypes:                []string{},
		SkipValidation:          false,
		BackupDir:               ".test-analyzer-backups",
		UnitTestTimeout:         1 * time.Second,
		IntegrationTestTimeout:  10 * time.Second,
		LoadTestTimeout:         30 * time.Second,
		SimpleIterationThreshold: 100,
		ComplexIterationThreshold: 20,
		SleepThreshold:         100 * time.Millisecond,
		Severity:                "",
		IssueTypes:              []string{},
		Packages:                []string{},
		Verbose:                 false,
		Quiet:                   false,
		Color:                   true,
		Help:                    false,
	}
}

// Option is a functional option for configuring the analyzer.
type Option func(*Config)

// WithDryRun sets the dry run flag.
func WithDryRun(dryRun bool) Option {
	return func(c *Config) {
		c.DryRun = dryRun
	}
}

// WithOutput sets the output format.
func WithOutput(output string) Option {
	return func(c *Config) {
		c.Output = output
	}
}

// WithOutputFile sets the output file path.
func WithOutputFile(outputFile string) Option {
	return func(c *Config) {
		c.OutputFile = outputFile
	}
}

// WithAutoFix sets the auto fix flag.
func WithAutoFix(autoFix bool) Option {
	return func(c *Config) {
		c.AutoFix = autoFix
	}
}

// WithFixTypes sets the fix types to apply.
func WithFixTypes(fixTypes []string) Option {
	return func(c *Config) {
		c.FixTypes = fixTypes
	}
}

// WithUnitTestTimeout sets the unit test timeout.
func WithUnitTestTimeout(timeout time.Duration) Option {
	return func(c *Config) {
		c.UnitTestTimeout = timeout
	}
}

// WithIntegrationTestTimeout sets the integration test timeout.
func WithIntegrationTestTimeout(timeout time.Duration) Option {
	return func(c *Config) {
		c.IntegrationTestTimeout = timeout
	}
}

// WithLoadTestTimeout sets the load test timeout.
func WithLoadTestTimeout(timeout time.Duration) Option {
	return func(c *Config) {
		c.LoadTestTimeout = timeout
	}
}

// WithSimpleIterationThreshold sets the simple iteration threshold.
func WithSimpleIterationThreshold(threshold int) Option {
	return func(c *Config) {
		c.SimpleIterationThreshold = threshold
	}
}

// WithComplexIterationThreshold sets the complex iteration threshold.
func WithComplexIterationThreshold(threshold int) Option {
	return func(c *Config) {
		c.ComplexIterationThreshold = threshold
	}
}

// WithSleepThreshold sets the sleep threshold.
func WithSleepThreshold(threshold time.Duration) Option {
	return func(c *Config) {
		c.SleepThreshold = threshold
	}
}

// WithSeverity sets the minimum severity to report.
func WithSeverity(severity string) Option {
	return func(c *Config) {
		c.Severity = severity
	}
}

// WithVerbose sets the verbose flag.
func WithVerbose(verbose bool) Option {
	return func(c *Config) {
		c.Verbose = verbose
	}
}

// WithQuiet sets the quiet flag.
func WithQuiet(quiet bool) Option {
	return func(c *Config) {
		c.Quiet = quiet
	}
}

// WithColor sets the color flag.
func WithColor(color bool) Option {
	return func(c *Config) {
		c.Color = color
	}
}

// WithExcludePackages sets the packages to exclude.
func WithExcludePackages(packages []string) Option {
	return func(c *Config) {
		c.ExcludePackages = packages
	}
}

