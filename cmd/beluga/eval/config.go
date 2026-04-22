// Package eval is the Layer 7 CLI-adapter for `beluga eval`. It owns the
// per-row exec-once IPC to the scaffolded user binary, dataset loading,
// metric dispatch, and report rendering. The heavy lifting (eval.Runner,
// metrics, OTel spans) lives in the framework-layer eval package; this
// subpackage is the thin glue between cobra and that API.
//
// DX-1 S4 brief: research/briefs/2026-04-21-dx1-s4-eval.md.
package eval

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// DefaultRowTimeout is the mandatory per-row subprocess timeout applied
// when the user supplies none. The S4 brief pins this at 30s to satisfy
// the "every external call must have a timeout" security rule while
// keeping the mock-provider smoke eval well under the 30s cold-cache CI
// budget (specialist-devops-expert §Q2).
const DefaultRowTimeout = 30 * time.Second

// DefaultProtocolProbeTimeout bounds how long the CLI waits for the
// user binary to emit its first `{"beluga_eval_protocol":1}` line. The
// brief names 5s as the user-facing migration hint ("please re-scaffold
// or add the BELUGA_ENV=eval branch manually"); anything shorter makes
// cold-start modules noisy on CI.
const DefaultProtocolProbeTimeout = 5 * time.Second

// DefaultReportFilename is where eval-report.json lands when no explicit
// path is configured — always at the project root so CI artefact steps
// can upload a well-known location.
const DefaultReportFilename = "eval-report.json"

// OutputFormatJSON is the always-on machine-readable renderer: every run
// writes eval-report.json regardless of any --format flag. The CLI has
// no --format=json toggle; it is unconditional.
const OutputFormatJSON = "json"

// OutputFormatJUnit is the optional additive renderer selected via
// --format junit. It writes <report>.junit.xml alongside the JSON
// artefact for consumption by dorny/test-reporter on GitHub Actions.
const OutputFormatJUnit = "junit"

// CLIConfig is the merged shape of .beluga/eval.yaml plus flag overrides.
// Flag values beat YAML values; YAML beats built-in defaults. The zero
// value is safe to [ApplyDefaults] and [Validate] against.
type CLIConfig struct {
	// Dataset is the path to the dataset JSON file — always flag- or
	// positional-supplied, never read from YAML, so an eval.yaml cannot
	// silently redirect a CI invocation to a different dataset.
	Dataset string `yaml:"-"`
	// Metrics lists the metric names to run. Defaults to
	// ["exact_match", "latency"] when empty (specialist-ai-ml-expert §Q2).
	Metrics []string `yaml:"metrics,omitempty"`
	// EvalProvider, when set, names an eval.Metric provider whose
	// Score-implementing types are used as scorers (braintrust /
	// deepeval / ragas). Single dispatch per §Q5.
	EvalProvider string `yaml:"eval_provider,omitempty"`
	// JudgeModel is the LLM model name passed to LLM-judge metrics.
	JudgeModel string `yaml:"judge_model,omitempty"`
	// Format is the optional additive output format. "" → JSON only;
	// "junit" → JSON + JUnit XML. No "json" toggle — JSON is always on.
	Format string `yaml:"format,omitempty"`
	// RowTimeout is the wall-clock cap on each row's subprocess
	// invocation. Defaults to [DefaultRowTimeout] when unset.
	RowTimeout time.Duration `yaml:"-"`
	// RowTimeoutRaw is the YAML-facing string form ("30s") parsed
	// into RowTimeout by [ApplyDefaults].
	RowTimeoutRaw string `yaml:"row_timeout,omitempty"`
	// MaxRows, when >0, caps the number of dataset rows actually
	// executed; the rest are skipped with a "not run" marker.
	MaxRows int `yaml:"max_rows,omitempty"`
	// MaxCost, when >0, caps the total USD cost summed across rows.
	// Rows whose projected cost would push the running total past
	// MaxCost are skipped.
	MaxCost float64 `yaml:"max_cost,omitempty"`
	// DryRun prints the planned run (row count, estimated cost) and
	// exits 0 without launching any subprocesses — the safe preview
	// before a real-provider run (specialist-devops-expert §Q2).
	DryRun bool `yaml:"dry_run,omitempty"`
	// Parallel is the worker-pool size. Defaults to 1 to avoid mock
	// fixture-queue cross-contamination (§Q4). Users can override to
	// >1 for real-provider runs.
	Parallel int `yaml:"parallel,omitempty"`
	// ProjectRoot is the scaffolded-project directory (has go.mod +
	// .beluga/project.yaml). Always absolute after [ApplyDefaults].
	ProjectRoot string `yaml:"-"`
	// ConfigPath records which eval.yaml was actually loaded (empty
	// when no file existed and the caller used defaults).
	ConfigPath string `yaml:"-"`
	// ReportPath is where the JSON report lands. Defaults to
	// <ProjectRoot>/eval-report.json.
	ReportPath string `yaml:"report_path,omitempty"`
}

// LoadConfig reads .beluga/eval.yaml from projectRoot and returns a
// populated [CLIConfig]. Missing eval.yaml when configPath is empty is
// not an error — the caller gets a default config ready for flag
// overrides. An explicit configPath that does not exist IS an error.
func LoadConfig(projectRoot, configPath string) (*CLIConfig, error) {
	cfg := &CLIConfig{ProjectRoot: projectRoot}

	explicit := configPath != ""
	path := configPath
	if path == "" {
		path = filepath.Join(projectRoot, ".beluga", "eval.yaml")
	}
	cleaned := filepath.Clean(path)
	// #nosec G304 -- path is supplied by the developer via --config or
	// derived from projectRoot (scanned to .beluga/eval.yaml). No
	// remote input reaches this read.
	data, err := os.ReadFile(cleaned)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) && !explicit {
			return cfg, nil
		}
		return nil, fmt.Errorf("read %s: %w", cleaned, err)
	}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse %s: %w", cleaned, err)
	}
	cfg.ConfigPath = cleaned
	return cfg, nil
}

// ApplyDefaults fills in mandatory defaults for any zero-valued fields.
// It parses the YAML-facing RowTimeoutRaw string ("30s") into the
// typed RowTimeout duration, enforces Parallel≥1, and defaults
// ReportPath under ProjectRoot. Called after flag overrides have been
// layered on so explicit zero overrides remain zero where the caller
// intended (e.g., MaxRows=0 means "no cap").
func (c *CLIConfig) ApplyDefaults() error {
	if c.RowTimeoutRaw != "" && c.RowTimeout == 0 {
		d, err := time.ParseDuration(c.RowTimeoutRaw)
		if err != nil {
			return fmt.Errorf("row_timeout %q: %w", c.RowTimeoutRaw, err)
		}
		c.RowTimeout = d
	}
	if c.RowTimeout <= 0 {
		c.RowTimeout = DefaultRowTimeout
	}
	if c.Parallel <= 0 {
		c.Parallel = 1
	}
	if len(c.Metrics) == 0 {
		c.Metrics = []string{"exact_match", "latency"}
	}
	if c.ProjectRoot == "" {
		c.ProjectRoot = "."
	}
	abs, err := filepath.Abs(c.ProjectRoot)
	if err != nil {
		return fmt.Errorf("resolve project-root: %w", err)
	}
	c.ProjectRoot = abs
	if c.ReportPath == "" {
		c.ReportPath = filepath.Join(abs, DefaultReportFilename)
	}
	return nil
}

// Validate reports configuration errors that the user can act on. It is
// intentionally conservative: unknown metric names are deferred to
// [ResolveMetrics] so the CLI can list valid names in the error
// message, while shape-level problems (empty dataset path, bad format)
// are caught here before any subprocess work begins.
func (c *CLIConfig) Validate() error {
	if c.Dataset == "" {
		return errors.New("dataset path is required (positional arg or --dataset)")
	}
	switch c.Format {
	case "", OutputFormatJUnit:
	default:
		return fmt.Errorf("unknown --format %q (valid: junit)", c.Format)
	}
	if c.RowTimeout <= 0 {
		return fmt.Errorf("row_timeout must be >0, got %s", c.RowTimeout)
	}
	if c.Parallel < 1 {
		return fmt.Errorf("parallel must be >=1, got %d", c.Parallel)
	}
	if c.MaxRows < 0 {
		return fmt.Errorf("max_rows must be >=0, got %d", c.MaxRows)
	}
	if c.MaxCost < 0 {
		return fmt.Errorf("max_cost must be >=0, got %v", c.MaxCost)
	}
	return nil
}

// WantsJUnit reports whether the JUnit renderer should run in addition
// to the always-on JSON renderer.
func (c *CLIConfig) WantsJUnit() bool {
	return strings.EqualFold(c.Format, OutputFormatJUnit)
}

// JUnitReportPath derives the JUnit sibling path from ReportPath —
// `<base>.junit.xml` next to the JSON report so CI artefact uploads
// pick both up with a single wildcard.
func (c *CLIConfig) JUnitReportPath() string {
	if c.ReportPath == "" {
		return ""
	}
	ext := filepath.Ext(c.ReportPath)
	base := strings.TrimSuffix(c.ReportPath, ext)
	return base + ".junit.xml"
}
