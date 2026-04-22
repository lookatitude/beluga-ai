package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	evalcmd "github.com/lookatitude/beluga-ai/v2/cmd/beluga/eval"
)

// evalRun is indirected so tests can substitute the runner. Production
// code always dispatches to evalcmd.Run which owns the exec-once IPC
// and framework-layer runner wire-up.
var evalRun = evalcmd.Run

// newEvalCmd returns the cobra subcommand for `beluga eval`. It loads
// .beluga/eval.yaml (if present), merges flag overrides, runs the
// evaluation against a JSON dataset, always writes eval-report.json to
// the project root, and optionally emits <report>.junit.xml alongside
// when --format junit is passed. Exit code follows the dataset's
// aggregate pass signal: any per-row exec error → non-zero exit.
func newEvalCmd() *cobra.Command {
	var (
		projectRoot  string
		datasetFlag  string
		configPath   string
		metricsFlag  []string
		evalProvider string
		judgeModel   string
		format       string
		rowTimeout   time.Duration
		maxRows      int
		maxCost      float64
		dryRun       bool
		parallel     int
	)

	cmd := &cobra.Command{
		Use:   "eval [dataset.json] [flags]",
		Short: "Run evaluations against a dataset",
		Long: "Evaluates the scaffolded project against a JSON dataset. Writes a " +
			"machine-readable eval-report.json artefact on every run and, with " +
			"--format junit, an additional <report>.junit.xml file consumable by " +
			"dorny/test-reporter. Child binaries are dispatched once per row via " +
			"BELUGA_ENV=eval + BELUGA_EVAL_SAMPLE_JSON.",
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dataset := datasetFlag
			if dataset == "" && len(args) == 1 {
				dataset = args[0]
			}
			return runEval(cmd.Context(), evalRunOptions{
				projectRoot:  projectRoot,
				dataset:      dataset,
				configPath:   configPath,
				metrics:      metricsFlag,
				evalProvider: evalProvider,
				judgeModel:   judgeModel,
				format:       format,
				rowTimeout:   rowTimeout,
				maxRows:      maxRows,
				maxCost:      maxCost,
				dryRun:       dryRun,
				parallel:     parallel,
			}, cmd.OutOrStdout(), cmd.ErrOrStderr())
		},
	}

	cmd.Flags().StringVar(&projectRoot, "project-root", ".",
		"scaffolded project root (directory with go.mod + .beluga/project.yaml)")
	cmd.Flags().StringVar(&datasetFlag, "dataset", "",
		"path to dataset JSON (defaults to positional argument)")
	cmd.Flags().StringVar(&configPath, "config", "",
		"path to eval config file (default: <project-root>/.beluga/eval.yaml)")
	cmd.Flags().StringSliceVar(&metricsFlag, "metric", nil,
		"metric name (repeatable); overrides any metrics set in the config file")
	cmd.Flags().StringVar(&evalProvider, "eval-provider", "",
		"eval-provider name (braintrust, deepeval, ragas) — reserved for S4.5")
	cmd.Flags().StringVar(&judgeModel, "judge-model", "",
		"LLM model name for LLM-judge metrics — reserved for S4.5")
	cmd.Flags().StringVar(&format, "format", "",
		"additive output format alongside the default JSON report ('junit')")
	cmd.Flags().DurationVar(&rowTimeout, "row-timeout", 0,
		"per-row subprocess wall-clock cap (default 30s)")
	cmd.Flags().IntVar(&maxRows, "max-rows", 0,
		"cap the number of rows actually executed (0 = no cap)")
	cmd.Flags().Float64Var(&maxCost, "max-cost", 0,
		"cap the total USD cost across rows (0 = no cap) — reserved for S4.5")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false,
		"print the planned run (row count, row_timeout) without launching subprocesses")
	cmd.Flags().IntVar(&parallel, "parallel", 0,
		"worker-pool size for per-row subprocess dispatch (default 1)")

	cmd.AddCommand(newEvalSchemaCmd())
	return cmd
}

type evalRunOptions struct {
	projectRoot  string
	dataset      string
	configPath   string
	metrics      []string
	evalProvider string
	judgeModel   string
	format       string
	rowTimeout   time.Duration
	maxRows      int
	maxCost      float64
	dryRun       bool
	parallel     int
}

func runEval(ctx context.Context, opts evalRunOptions, stdout, stderr io.Writer) error {
	if ctx == nil {
		ctx = context.Background()
	}

	root, err := filepath.Abs(opts.projectRoot)
	if err != nil {
		return fmt.Errorf("resolve project-root: %w", err)
	}

	cfg, err := evalcmd.LoadConfig(root, opts.configPath)
	if err != nil {
		return err
	}
	applyEvalOverrides(cfg, opts)
	if err := cfg.ApplyDefaults(); err != nil {
		return err
	}

	report, err := evalRun(ctx, cfg, stdout, stderr)
	if err != nil {
		return err
	}

	if err := writeReports(cfg, report, stdout); err != nil {
		return err
	}
	if err := evalcmd.RenderText(stdout, report); err != nil {
		return err
	}

	if len(report.Errors) > 0 && !cfg.DryRun {
		return &runExitError{code: 1, err: fmt.Errorf("eval: %d row error(s)", len(report.Errors))}
	}
	return nil
}

func applyEvalOverrides(cfg *evalcmd.CLIConfig, opts evalRunOptions) {
	if opts.dataset != "" {
		cfg.Dataset = opts.dataset
	}
	if len(opts.metrics) > 0 {
		cfg.Metrics = opts.metrics
	}
	if opts.evalProvider != "" {
		cfg.EvalProvider = opts.evalProvider
	}
	if opts.judgeModel != "" {
		cfg.JudgeModel = opts.judgeModel
	}
	if opts.format != "" {
		cfg.Format = opts.format
	}
	if opts.rowTimeout > 0 {
		cfg.RowTimeout = opts.rowTimeout
	}
	if opts.maxRows > 0 {
		cfg.MaxRows = opts.maxRows
	}
	if opts.maxCost > 0 {
		cfg.MaxCost = opts.maxCost
	}
	if opts.dryRun {
		cfg.DryRun = true
	}
	if opts.parallel > 0 {
		cfg.Parallel = opts.parallel
	}
}

func writeReports(cfg *evalcmd.CLIConfig, report *evalcmd.Report, stdout io.Writer) error {
	jsonPath := cfg.ReportPath
	if jsonPath == "" {
		jsonPath = filepath.Join(cfg.ProjectRoot, evalcmd.DefaultReportFilename)
	}
	if err := writeReportJSON(jsonPath, report); err != nil {
		return fmt.Errorf("write json report: %w", err)
	}
	fmt.Fprintf(stdout, "wrote %s\n", jsonPath)

	if cfg.WantsJUnit() {
		junitPath := cfg.JUnitReportPath()
		if err := writeReportJUnit(junitPath, report); err != nil {
			return fmt.Errorf("write junit report: %w", err)
		}
		fmt.Fprintf(stdout, "wrote %s\n", junitPath)
	}
	return nil
}

func writeReportJSON(path string, report *evalcmd.Report) error {
	f, err := os.Create(filepath.Clean(path)) //nolint:gosec // G304: path is derived from cfg.ReportPath (flag/yaml/default); no remote input.
	if err != nil {
		return err
	}
	defer f.Close()
	return evalcmd.RenderJSON(f, report)
}

func writeReportJUnit(path string, report *evalcmd.Report) error {
	f, err := os.Create(filepath.Clean(path)) //nolint:gosec // G304: path is derived from cfg.JUnitReportPath; no remote input.
	if err != nil {
		return err
	}
	defer f.Close()
	return evalcmd.RenderJUnit(f, report)
}

// newEvalSchemaCmd is the hidden `beluga eval schema` subcommand — it
// prints the embedded JSON Schema for editor validation. Hidden because
// users don't need to see it in --help; it is only invoked by tooling.
func newEvalSchemaCmd() *cobra.Command {
	return &cobra.Command{
		Use:    "schema",
		Short:  "Print the embedded dataset JSON Schema",
		Hidden: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			_, err := cmd.OutOrStdout().Write(evalcmd.DatasetSchema())
			return err
		},
	}
}
