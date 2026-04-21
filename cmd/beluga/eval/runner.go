package eval

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/lookatitude/beluga-ai/v2/cmd/beluga/devloop"
	"github.com/lookatitude/beluga-ai/v2/eval"
	"github.com/lookatitude/beluga-ai/v2/eval/metrics"
)

// ProtocolVersion is the IPC protocol version the scaffolded main.go
// emits as its first stdout line: `{"beluga_eval_protocol":1}`. The
// CLI rejects any child that fails to emit this line within
// [DefaultProtocolProbeTimeout], steering the user to re-scaffold the
// project or add the `BELUGA_ENV=eval` branch manually (S4 Risk 7).
const ProtocolVersion = 1

// protocolProbe is the exact JSON envelope the child emits to claim
// protocol conformance. Matched as the first stdout line before the
// populated-sample payload.
type protocolProbe struct {
	BelugaEvalProtocol int `json:"beluga_eval_protocol"`
}

// metricFactory returns a concrete eval.Metric for a given config-level
// name. The CLI owns the name → constructor mapping because registering
// every metric variant in the framework eval package would couple the
// framework layer to CLI-visible metric names.
type metricFactory func(cfg *CLIConfig) (eval.Metric, error)

// builtInMetrics is the closed set of metric names the CLI recognises
// today. Provider-backed metrics (braintrust_*, deepeval_*, ragas_*)
// land via [CLIConfig.EvalProvider] in a follow-up.
var builtInMetrics = map[string]metricFactory{
	"exact_match": func(_ *CLIConfig) (eval.Metric, error) {
		return metrics.NewExactMatch(), nil
	},
	"latency": func(_ *CLIConfig) (eval.Metric, error) {
		return metrics.NewLatency(), nil
	},
}

// ResolveMetrics turns the config's metric names into concrete
// [eval.Metric] implementations. Unknown names produce a single error
// listing every valid name so the user sees the full menu at once.
func ResolveMetrics(cfg *CLIConfig) ([]eval.Metric, error) {
	out := make([]eval.Metric, 0, len(cfg.Metrics))
	var unknown []string
	for _, name := range cfg.Metrics {
		factory, ok := builtInMetrics[name]
		if !ok {
			unknown = append(unknown, name)
			continue
		}
		m, err := factory(cfg)
		if err != nil {
			return nil, fmt.Errorf("metric %q: %w", name, err)
		}
		out = append(out, m)
	}
	if len(unknown) > 0 {
		valid := make([]string, 0, len(builtInMetrics))
		for name := range builtInMetrics {
			valid = append(valid, name)
		}
		return nil, fmt.Errorf("unknown metric(s) %v (valid: %v)", unknown, valid)
	}
	return out, nil
}

// Report is the CLI-level artefact shape. It mirrors [eval.EvalReport]
// with the run-level identifier needed to join traces + metrics + JSON
// artefact in downstream aggregation tools (S4 brief §Decision
// summary, specialist-observability-expert §Q5).
type Report struct {
	RunID       string                 `json:"run_id"`
	DatasetName string                 `json:"dataset"`
	DatasetPath string                 `json:"dataset_path"`
	StartedAt   time.Time              `json:"started_at"`
	Duration    time.Duration          `json:"duration"`
	Samples     []SampleReport         `json:"samples"`
	Aggregate   map[string]float64     `json:"aggregate"`
	Errors      []string               `json:"errors,omitempty"`
	Skipped     int                    `json:"skipped,omitempty"`
	DryRun      bool                   `json:"dry_run,omitempty"`
	Extras      map[string]interface{} `json:"-"`
}

// SampleReport is a per-row view of one sample's scores and outcome.
type SampleReport struct {
	Index    int                `json:"index"`
	RowID    string             `json:"row_id"`
	Input    string             `json:"input"`
	Output   string             `json:"output"`
	Expected string             `json:"expected,omitempty"`
	Scores   map[string]float64 `json:"scores"`
	Error    string             `json:"error,omitempty"`
}

// builderFunc is indirected for testing so unit tests can stub the
// binary build without invoking the Go toolchain.
var builderFunc = devloop.BuildBinary

// execCommand is indirected for testing so unit tests can stub
// per-row subprocess execution.
var execCommand = func(ctx context.Context, name string, args ...string) *exec.Cmd {
	// #nosec G204 -- name is always an absolute path produced by
	// devloop.BuildBinary under os.TempDir(). args is empty — the
	// child receives its sample exclusively via the BELUGA_EVAL_SAMPLE_JSON
	// env var, never via argv. No shell is involved.
	return exec.CommandContext(ctx, name, args...) //nolint:gosec // G204: see nosec justification above
}

// Run is the CLI entry point. It loads the dataset, resolves metrics,
// builds the user binary once, execs it per row, parses the populated
// [eval.EvalSample] back from stdout, and hands the populated samples
// plus metrics to the framework eval runner for OTel-instrumented
// scoring. A run with --dry-run skips the build + exec path entirely
// and returns a report marked with DryRun=true.
func Run(ctx context.Context, cfg *CLIConfig, stdout, stderr io.Writer) (*Report, error) {
	if cfg == nil {
		return nil, errors.New("nil config")
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	datasetPath := filepath.Clean(cfg.Dataset)
	ds, err := eval.LoadDataset(datasetPath)
	if err != nil {
		return nil, fmt.Errorf("load dataset %s: %w", datasetPath, err)
	}

	samples := applyMaxRows(ds.Samples, cfg.MaxRows)
	skipped := len(ds.Samples) - len(samples)

	runID := newRunID()
	started := time.Now()

	if cfg.DryRun {
		report := &Report{
			RunID:       runID,
			DatasetName: ds.Name,
			DatasetPath: datasetPath,
			StartedAt:   started,
			Samples:     []SampleReport{},
			Aggregate:   map[string]float64{},
			Skipped:     skipped,
			DryRun:      true,
		}
		fmt.Fprintf(stdout, "dry-run: %d rows queued (capped from %d), row_timeout=%s\n",
			len(samples), len(ds.Samples), cfg.RowTimeout)
		report.Duration = time.Since(started)
		return report, nil
	}

	ms, err := ResolveMetrics(cfg)
	if err != nil {
		return nil, err
	}

	build, err := builderFunc(ctx, cfg.ProjectRoot, 0, stderr, stderr)
	if err != nil {
		return nil, fmt.Errorf("build user binary: %w", err)
	}
	defer removeBinary(build.OutputPath, stderr)

	populated, execErrs := execAllRows(ctx, cfg, build.OutputPath, samples, runID, stderr)

	runner := eval.NewRunner(
		eval.WithDataset(populated),
		eval.WithMetrics(ms...),
		eval.WithDatasetName(datasetName(ds, datasetPath)),
		eval.WithParallel(1), // scoring is in-process and cheap; exec was the parallel hot path
	)
	evalReport, err := runner.Run(ctx)
	if err != nil {
		return nil, fmt.Errorf("eval runner: %w", err)
	}

	report := toCLIReport(evalReport, runID, ds, datasetPath, started, skipped)
	for i, e := range execErrs {
		if e == "" {
			continue
		}
		if i < len(report.Samples) {
			report.Samples[i].Error = e
		}
		report.Errors = append(report.Errors, e)
	}
	return report, nil
}

// execAllRows runs every sample through the user binary serially (mock
// default) or in a worker pool (real-provider runs). Each invocation
// lives under [CLIConfig.RowTimeout]; missing populated output is
// reported as a per-row error without aborting the whole run.
func execAllRows(ctx context.Context, cfg *CLIConfig, binary string, samples []eval.EvalSample, runID string, stderr io.Writer) ([]eval.EvalSample, []string) {
	populated := make([]eval.EvalSample, len(samples))
	errs := make([]string, len(samples))

	sem := make(chan struct{}, cfg.Parallel)
	done := make(chan int, len(samples))

	for i := range samples {
		sem <- struct{}{}
		go func(idx int) {
			defer func() { <-sem }()
			defer func() { done <- idx }()

			rowCtx, cancel := context.WithTimeout(ctx, cfg.RowTimeout)
			defer cancel()

			rowID := newRunID()
			out, err := execOneRow(rowCtx, binary, cfg, samples[idx], runID, rowID, stderr)
			if err != nil {
				populated[idx] = samples[idx]
				errs[idx] = err.Error()
				return
			}
			populated[idx] = out
		}(i)
	}
	for range samples {
		<-done
	}
	return populated, errs
}

// execOneRow execs the user binary once with the sample JSON in the
// environment, validates the protocol probe on the first stdout line,
// and returns the populated sample decoded from the second line. Wall-
// clock latency is recorded into out.Metadata["latency_ms"] unless the
// child already set it — this is the canonical source of truth for the
// built-in latency metric and decouples it from child-reported timing.
func execOneRow(ctx context.Context, binary string, cfg *CLIConfig, sample eval.EvalSample, runID, rowID string, stderr io.Writer) (eval.EvalSample, error) {
	payload, err := json.Marshal(sample)
	if err != nil {
		return sample, fmt.Errorf("marshal sample: %w", err)
	}

	loaded, err := devloop.LoadProjectEnv(cfg.ProjectRoot)
	if err != nil {
		return sample, fmt.Errorf("load project env: %w", err)
	}
	extras := []string{
		"BELUGA_ENV=eval",
		"BELUGA_EVAL_RUN_ID=" + runID,
		"BELUGA_EVAL_ROW_ID=" + rowID,
		"BELUGA_EVAL_SAMPLE_JSON=" + string(payload),
	}
	env := devloop.MergeEnv(os.Environ(), loaded, extras)

	cmd := execCommand(ctx, binary)
	cmd.Dir = cfg.ProjectRoot
	cmd.Env = env
	cmd.Stderr = stderr
	// WaitDelay bounds cmd.Wait() when grandchildren (e.g., `sleep` inside
	// a shell script) keep the stdout pipe open after ctx.Done kills the
	// direct child. Without it, the reader below would block indefinitely
	// waiting on EOF that never comes.
	cmd.WaitDelay = 2 * time.Second
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return sample, fmt.Errorf("stdout pipe: %w", err)
	}
	started := time.Now()
	if err := cmd.Start(); err != nil {
		return sample, fmt.Errorf("start: %w", err)
	}

	type readOutcome struct {
		sample eval.EvalSample
		err    error
	}
	results := make(chan readOutcome, 1)
	go func() {
		out, err := readProtocolAndSample(stdoutPipe)
		results <- readOutcome{out, err}
	}()

	var rr readOutcome
	select {
	case rr = <-results:
	case <-ctx.Done():
		// Force the reader to unblock by closing the stdout pipe — any
		// grandchild holding the fd will simply get EPIPE on its next
		// write. cmd.WaitDelay below ensures Wait() still returns.
		_ = stdoutPipe.Close()
		rr = <-results
		if rr.err == nil {
			rr.err = ctx.Err()
		}
	}

	waitErr := cmd.Wait()
	elapsed := time.Since(started)

	switch {
	case rr.err != nil:
		return sample, rr.err
	case waitErr != nil:
		return sample, fmt.Errorf("child exited: %w", waitErr)
	}

	out := rr.sample
	if out.Metadata == nil {
		out.Metadata = map[string]any{}
	}
	if _, ok := out.Metadata["latency_ms"]; !ok {
		out.Metadata["latency_ms"] = float64(elapsed.Milliseconds())
	}
	return out, nil
}

// readProtocolAndSample consumes the child's stdout: first line MUST
// be the protocol probe; the next non-empty line MUST be the populated
// sample JSON. Any deviation is returned as a typed error so callers
// can distinguish "project not scaffolded for eval" from "child
// crashed mid-response".
func readProtocolAndSample(r io.Reader) (eval.EvalSample, error) {
	var empty eval.EvalSample
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	if !sc.Scan() {
		if err := sc.Err(); err != nil {
			return empty, fmt.Errorf("read protocol probe: %w", err)
		}
		return empty, fmt.Errorf("no output from child — project may not have the BELUGA_ENV=eval branch; re-scaffold or add manually")
	}
	var probe protocolProbe
	if err := json.Unmarshal(sc.Bytes(), &probe); err != nil || probe.BelugaEvalProtocol != ProtocolVersion {
		return empty, fmt.Errorf("child did not emit eval protocol probe on first line (expected %q, got %q)",
			fmt.Sprintf(`{"beluga_eval_protocol":%d}`, ProtocolVersion), sc.Text())
	}
	var payload eval.EvalSample
	for sc.Scan() {
		line := sc.Bytes()
		if len(line) == 0 {
			continue
		}
		if err := json.Unmarshal(line, &payload); err != nil {
			return empty, fmt.Errorf("decode populated sample: %w", err)
		}
		return payload, nil
	}
	if err := sc.Err(); err != nil {
		return empty, fmt.Errorf("read populated sample: %w", err)
	}
	return empty, errors.New("child emitted protocol probe but no populated sample")
}

// applyMaxRows returns the first n samples when n > 0, else the full
// slice. A negative n is rejected upstream in [CLIConfig.Validate].
func applyMaxRows(samples []eval.EvalSample, n int) []eval.EvalSample {
	if n <= 0 || n >= len(samples) {
		return samples
	}
	return samples[:n]
}

// newRunID produces a 16-hex-char (64-bit) identifier for run/row
// correlation. crypto/rand is used per the framework security rule
// against math/rand in identifier paths.
func newRunID() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("err-%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b[:])
}

// datasetName returns the dataset's declared Name, falling back to the
// filename (without extension) when Name is unset. Used as both the
// beluga.eval.dataset OTel label and the report's dataset identifier.
func datasetName(ds *eval.Dataset, path string) string {
	if ds.Name != "" {
		return ds.Name
	}
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	return base[:len(base)-len(ext)]
}

// toCLIReport adapts the framework-layer report into the CLI artefact
// shape. The CLI adds run_id and source-dataset metadata that the
// framework layer deliberately does not know about.
func toCLIReport(rep *eval.EvalReport, runID string, ds *eval.Dataset, path string, started time.Time, skipped int) *Report {
	report := &Report{
		RunID:       runID,
		DatasetName: datasetName(ds, path),
		DatasetPath: path,
		StartedAt:   started,
		Duration:    rep.Duration,
		Aggregate:   rep.Metrics,
		Samples:     make([]SampleReport, len(rep.Samples)),
		Skipped:     skipped,
	}
	for i, s := range rep.Samples {
		sr := SampleReport{
			Index:    i,
			RowID:    fmt.Sprintf("%s-%04d", runID, i),
			Input:    s.Sample.Input,
			Output:   s.Sample.Output,
			Expected: s.Sample.ExpectedOutput,
			Scores:   s.Scores,
		}
		if s.Error != nil {
			sr.Error = s.Error.Error()
			report.Errors = append(report.Errors, s.Error.Error())
		}
		report.Samples[i] = sr
	}
	return report
}

func removeBinary(path string, stderr io.Writer) {
	if path == "" {
		return
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		fmt.Fprintf(stderr, "warning: remove %s: %v\n", path, err)
	}
}
