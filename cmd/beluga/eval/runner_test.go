package eval

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lookatitude/beluga-ai/v2/cmd/beluga/devloop"
	"github.com/lookatitude/beluga-ai/v2/eval"
)

// writeDataset serialises the given Dataset to a tempdir JSON file and
// returns its path. Centralises the boilerplate so per-test setup stays
// legible.
func writeDataset(t *testing.T, dir string, ds eval.Dataset) string {
	t.Helper()
	path := filepath.Join(dir, "dataset.json")
	require.NoError(t, ds.Save(path))
	return path
}

// writeFakeBinary writes an executable shell/batch script at path that
// emits the supplied stdout lines in order. On non-posix platforms the
// tests skip since the CLI runner is exercised end-to-end in
// eval-integration CI; the Linux-only surface matches S4 §Q5.
func writeFakeBinary(t *testing.T, path string, stdoutLines []string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("fake-binary runner test is posix-only")
	}
	var body strings.Builder
	body.WriteString("#!/bin/sh\n")
	for _, line := range stdoutLines {
		body.WriteString("printf '%s\\n' ")
		body.WriteString(shellQuote(line))
		body.WriteString("\n")
	}
	require.NoError(t, os.WriteFile(path, []byte(body.String()), 0o700)) //nolint:gosec // G306: test-only fake binary
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

// stubBuilder returns a builderFunc that yields a pre-created binary
// at fixedPath instead of invoking the Go toolchain.
func stubBuilder(fixedPath string) func(ctx context.Context, projectRoot string, seq int, stdout, stderr io.Writer) (*devloop.BuildResult, error) {
	return func(_ context.Context, _ string, _ int, _, _ io.Writer) (*devloop.BuildResult, error) {
		return &devloop.BuildResult{OutputPath: fixedPath}, nil
	}
}

func installBuilder(t *testing.T, fn func(ctx context.Context, projectRoot string, seq int, stdout, stderr io.Writer) (*devloop.BuildResult, error)) {
	t.Helper()
	prev := builderFunc
	builderFunc = fn
	t.Cleanup(func() { builderFunc = prev })
}

func installExec(t *testing.T, fn func(ctx context.Context, name string, args ...string) *exec.Cmd) {
	t.Helper()
	prev := execCommand
	execCommand = fn
	t.Cleanup(func() { execCommand = prev })
}

// --- tests ---

func TestResolveMetrics_BuiltIns(t *testing.T) {
	cfg := &CLIConfig{Metrics: []string{"exact_match", "latency"}}
	got, err := ResolveMetrics(cfg)
	require.NoError(t, err)
	require.Len(t, got, 2)
	names := []string{got[0].Name(), got[1].Name()}
	assert.ElementsMatch(t, []string{"exact_match", "latency"}, names)
}

func TestResolveMetrics_UnknownNameListsValid(t *testing.T) {
	cfg := &CLIConfig{Metrics: []string{"not_a_metric"}}
	_, err := ResolveMetrics(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not_a_metric")
	assert.Contains(t, err.Error(), "exact_match")
}

func TestRun_RequiresDataset(t *testing.T) {
	cfg := &CLIConfig{}
	require.NoError(t, cfg.ApplyDefaults())
	_, err := Run(context.Background(), cfg, io.Discard, io.Discard)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "dataset")
}

func TestRun_DryRun_NoBuildOrExec(t *testing.T) {
	dir := t.TempDir()
	ds := eval.Dataset{
		Name: "smoke",
		Samples: []eval.EvalSample{
			{Input: "q1", ExpectedOutput: "a1"},
			{Input: "q2", ExpectedOutput: "a2"},
		},
	}
	path := writeDataset(t, dir, ds)

	// Both builder and execCommand should remain untouched: install
	// panicky stubs so any accidental call fails the test loudly.
	installBuilder(t, func(context.Context, string, int, io.Writer, io.Writer) (*devloop.BuildResult, error) {
		t.Fatal("builder should not run in --dry-run")
		return nil, nil
	})
	installExec(t, func(context.Context, string, ...string) *exec.Cmd {
		t.Fatal("exec should not run in --dry-run")
		return nil
	})

	cfg := &CLIConfig{
		Dataset:     path,
		DryRun:      true,
		ProjectRoot: dir,
	}
	require.NoError(t, cfg.ApplyDefaults())

	var stdout bytes.Buffer
	report, err := Run(context.Background(), cfg, &stdout, io.Discard)
	require.NoError(t, err)
	require.NotNil(t, report)
	assert.True(t, report.DryRun)
	assert.Equal(t, "smoke", report.DatasetName)
	assert.Empty(t, report.Samples)
	assert.Contains(t, stdout.String(), "dry-run")
}

func TestRun_ExecOncePerRow_PopulatesSamples(t *testing.T) {
	dir := t.TempDir()
	binaryPath := filepath.Join(dir, "fake-beluga-app")

	ds := eval.Dataset{
		Name: "exec-test",
		Samples: []eval.EvalSample{
			{Input: "capital of France?", ExpectedOutput: "Paris"},
			{Input: "capital of Spain?", ExpectedOutput: "Madrid"},
		},
	}
	path := writeDataset(t, dir, ds)

	// Fake binary emits the protocol probe and a populated sample
	// whose Output matches the ExpectedOutput so exact_match scores
	// 1.0 for every row.
	populatedParis := eval.EvalSample{Input: "capital of France?", ExpectedOutput: "Paris", Output: "Paris"}
	populatedMadrid := eval.EvalSample{Input: "capital of Spain?", ExpectedOutput: "Madrid", Output: "Madrid"}
	parisJSON, err := json.Marshal(populatedParis)
	require.NoError(t, err)
	madridJSON, err := json.Marshal(populatedMadrid)
	require.NoError(t, err)

	installBuilder(t, stubBuilder(binaryPath))
	// Swap the response on each call so row 0 → Paris, row 1 → Madrid.
	calls := 0
	responses := [][]byte{parisJSON, madridJSON}
	installExec(t, func(ctx context.Context, name string, args ...string) *exec.Cmd {
		writeFakeBinary(t, binaryPath, []string{
			`{"beluga_eval_protocol":1}`,
			string(responses[calls%len(responses)]),
		})
		calls++
		// #nosec G204 -- test-only binary written above in fixed tempdir.
		return exec.CommandContext(ctx, name, args...)
	})

	cfg := &CLIConfig{
		Dataset:     path,
		ProjectRoot: dir,
	}
	require.NoError(t, cfg.ApplyDefaults())

	report, err := Run(context.Background(), cfg, io.Discard, io.Discard)
	require.NoError(t, err)
	require.NotNil(t, report)
	require.Len(t, report.Samples, 2, "one row per sample")

	for _, sr := range report.Samples {
		assert.Empty(t, sr.Error, "no per-row errors")
		assert.Equal(t, 1.0, sr.Scores["exact_match"], "exact_match scores 1.0 when Output=Expected")
	}
	assert.Equal(t, 1.0, report.Aggregate["exact_match"])
}

func TestRun_ChildMissingProtocol_RowErrored(t *testing.T) {
	dir := t.TempDir()
	binaryPath := filepath.Join(dir, "fake-beluga-app")

	ds := eval.Dataset{Samples: []eval.EvalSample{{Input: "q", ExpectedOutput: "a"}}}
	path := writeDataset(t, dir, ds)

	installBuilder(t, stubBuilder(binaryPath))
	installExec(t, func(ctx context.Context, name string, args ...string) *exec.Cmd {
		writeFakeBinary(t, binaryPath, []string{"not a probe"})
		// #nosec G204 -- test-only binary written above.
		return exec.CommandContext(ctx, name, args...)
	})

	cfg := &CLIConfig{
		Dataset:     path,
		ProjectRoot: dir,
	}
	require.NoError(t, cfg.ApplyDefaults())

	report, err := Run(context.Background(), cfg, io.Discard, io.Discard)
	require.NoError(t, err, "row failures are per-row, not run-level")
	require.Len(t, report.Samples, 1)
	assert.NotEmpty(t, report.Samples[0].Error)
	assert.Contains(t, strings.Join(report.Errors, "|"), "eval protocol probe")
}

func TestRun_MaxRows_Truncates(t *testing.T) {
	dir := t.TempDir()
	binaryPath := filepath.Join(dir, "fake-beluga-app")

	ds := eval.Dataset{Samples: make([]eval.EvalSample, 5)}
	for i := range ds.Samples {
		ds.Samples[i] = eval.EvalSample{Input: "q", ExpectedOutput: "a", Output: "a"}
	}
	path := writeDataset(t, dir, ds)

	installBuilder(t, stubBuilder(binaryPath))
	populated := eval.EvalSample{Input: "q", ExpectedOutput: "a", Output: "a"}
	populatedJSON, err := json.Marshal(populated)
	require.NoError(t, err)
	installExec(t, func(ctx context.Context, name string, args ...string) *exec.Cmd {
		writeFakeBinary(t, binaryPath, []string{
			`{"beluga_eval_protocol":1}`,
			string(populatedJSON),
		})
		// #nosec G204 -- test-only binary written above.
		return exec.CommandContext(ctx, name, args...)
	})

	cfg := &CLIConfig{
		Dataset:     path,
		ProjectRoot: dir,
		MaxRows:     2,
	}
	require.NoError(t, cfg.ApplyDefaults())

	report, err := Run(context.Background(), cfg, io.Discard, io.Discard)
	require.NoError(t, err)
	assert.Len(t, report.Samples, 2, "max_rows caps the run")
	assert.Equal(t, 3, report.Skipped, "remaining rows counted as skipped")
}

func TestReadProtocolAndSample_HappyPath(t *testing.T) {
	sample := eval.EvalSample{Input: "q", Output: "a"}
	payload, err := json.Marshal(sample)
	require.NoError(t, err)
	buf := bytes.NewBufferString(`{"beluga_eval_protocol":1}` + "\n" + string(payload) + "\n")
	got, err := readProtocolAndSample(buf)
	require.NoError(t, err)
	assert.Equal(t, sample.Output, got.Output)
}

func TestReadProtocolAndSample_MissingProbe(t *testing.T) {
	buf := bytes.NewBufferString(`{"not":"a probe"}` + "\n")
	_, err := readProtocolAndSample(buf)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "eval protocol probe")
}

func TestReadProtocolAndSample_NoSampleAfterProbe(t *testing.T) {
	buf := bytes.NewBufferString(`{"beluga_eval_protocol":1}` + "\n")
	_, err := readProtocolAndSample(buf)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "populated sample")
}

func TestApplyMaxRows_Table(t *testing.T) {
	tests := []struct {
		name string
		n    int
		in   int
		want int
	}{
		{"zero → full", 0, 5, 5},
		{"negative → full", -1, 5, 5},
		{"cap smaller", 2, 5, 2},
		{"cap equal", 5, 5, 5},
		{"cap larger", 10, 5, 5},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			samples := make([]eval.EvalSample, tt.in)
			got := applyMaxRows(samples, tt.n)
			assert.Len(t, got, tt.want)
		})
	}
}

func TestRun_RespectsRowTimeout(t *testing.T) {
	dir := t.TempDir()
	binaryPath := filepath.Join(dir, "slow-beluga-app")

	ds := eval.Dataset{Samples: []eval.EvalSample{{Input: "q", ExpectedOutput: "a"}}}
	path := writeDataset(t, dir, ds)

	installBuilder(t, stubBuilder(binaryPath))
	// Fake binary blocks forever — RowTimeout must kill it.
	writeFakeBinary(t, binaryPath, []string{
		`{"beluga_eval_protocol":1}`,
	})
	// Append a trailing `sleep 60` so the process hangs after the
	// probe, exceeding RowTimeout.
	require.NoError(t, os.WriteFile(binaryPath, []byte("#!/bin/sh\nprintf '{\"beluga_eval_protocol\":1}\\n'\nsleep 60\n"), 0o700)) //nolint:gosec // G306: test-only fake binary
	installExec(t, func(ctx context.Context, name string, args ...string) *exec.Cmd {
		// #nosec G204 -- test-only binary written above.
		return exec.CommandContext(ctx, name, args...)
	})

	cfg := &CLIConfig{
		Dataset:     path,
		ProjectRoot: dir,
		RowTimeout:  500 * time.Millisecond,
	}
	require.NoError(t, cfg.ApplyDefaults())

	start := time.Now()
	report, err := Run(context.Background(), cfg, io.Discard, io.Discard)
	elapsed := time.Since(start)
	require.NoError(t, err)
	require.Len(t, report.Samples, 1)
	assert.NotEmpty(t, report.Samples[0].Error, "timed-out row must record an error")
	assert.Less(t, elapsed, 30*time.Second, "row timeout must bound wall-clock")
}

func TestNewRunID_HexAndStable(t *testing.T) {
	got := newRunID()
	assert.Len(t, got, 16)
	for _, r := range got {
		assert.True(t, (r >= '0' && r <= '9') || (r >= 'a' && r <= 'f'), "hex digits only")
	}
}
