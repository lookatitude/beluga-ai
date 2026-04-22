package eval_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	evalcmd "github.com/lookatitude/beluga-ai/v2/cmd/beluga/eval"
)

func TestLoadConfig_MissingFile_ReturnsDefaultsWhenImplicit(t *testing.T) {
	dir := t.TempDir()
	cfg, err := evalcmd.LoadConfig(dir, "")
	require.NoError(t, err)
	require.NotNil(t, cfg)
	assert.Equal(t, dir, cfg.ProjectRoot)
	assert.Empty(t, cfg.ConfigPath)
	assert.Empty(t, cfg.Metrics, "defaults not applied until ApplyDefaults")
}

func TestLoadConfig_MissingFile_ErrorsWhenExplicit(t *testing.T) {
	dir := t.TempDir()
	_, err := evalcmd.LoadConfig(dir, filepath.Join(dir, "nope.yaml"))
	require.Error(t, err, "explicit --config path must exist")
}

func TestLoadConfig_ParsesYAML(t *testing.T) {
	dir := t.TempDir()
	belugaDir := filepath.Join(dir, ".beluga")
	require.NoError(t, os.MkdirAll(belugaDir, 0o755))
	yamlBody := `
metrics:
  - exact_match
  - latency
row_timeout: 45s
parallel: 2
max_rows: 10
max_cost: 1.5
eval_provider: braintrust
judge_model: gpt-4o
`
	require.NoError(t, os.WriteFile(filepath.Join(belugaDir, "eval.yaml"), []byte(yamlBody), 0o600))

	cfg, err := evalcmd.LoadConfig(dir, "")
	require.NoError(t, err)
	assert.Equal(t, []string{"exact_match", "latency"}, cfg.Metrics)
	assert.Equal(t, "45s", cfg.RowTimeoutRaw)
	assert.Equal(t, 2, cfg.Parallel)
	assert.Equal(t, 10, cfg.MaxRows)
	assert.InDelta(t, 1.5, cfg.MaxCost, 1e-9)
	assert.Equal(t, "braintrust", cfg.EvalProvider)
	assert.Equal(t, "gpt-4o", cfg.JudgeModel)
	assert.NotEmpty(t, cfg.ConfigPath)
}

func TestApplyDefaults_FillsMandatoryFields(t *testing.T) {
	cfg := &evalcmd.CLIConfig{}
	require.NoError(t, cfg.ApplyDefaults())

	assert.Equal(t, evalcmd.DefaultRowTimeout, cfg.RowTimeout)
	assert.Equal(t, 1, cfg.Parallel)
	assert.Equal(t, []string{"exact_match", "latency"}, cfg.Metrics)
	assert.True(t, filepath.IsAbs(cfg.ProjectRoot), "ProjectRoot must be absolute")
	assert.True(t, filepath.IsAbs(cfg.ReportPath), "ReportPath must be absolute")
	assert.Equal(t, evalcmd.DefaultReportFilename, filepath.Base(cfg.ReportPath))
}

func TestApplyDefaults_ParsesRowTimeoutRaw(t *testing.T) {
	cfg := &evalcmd.CLIConfig{RowTimeoutRaw: "45s"}
	require.NoError(t, cfg.ApplyDefaults())
	assert.Equal(t, 45*time.Second, cfg.RowTimeout)
}

func TestApplyDefaults_InvalidRowTimeoutRaw(t *testing.T) {
	cfg := &evalcmd.CLIConfig{RowTimeoutRaw: "nope"}
	err := cfg.ApplyDefaults()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "row_timeout")
}

func TestApplyDefaults_FlagOverrideDoesNotReplaceExplicitRowTimeout(t *testing.T) {
	cfg := &evalcmd.CLIConfig{RowTimeout: 10 * time.Second}
	require.NoError(t, cfg.ApplyDefaults())
	assert.Equal(t, 10*time.Second, cfg.RowTimeout, "explicit flag value survives ApplyDefaults")
}

func TestValidate_RequiresDataset(t *testing.T) {
	cfg := &evalcmd.CLIConfig{}
	require.NoError(t, cfg.ApplyDefaults())
	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "dataset")
}

func TestValidate_UnknownFormat(t *testing.T) {
	cfg := &evalcmd.CLIConfig{Dataset: "ds.json", Format: "sarif"}
	require.NoError(t, cfg.ApplyDefaults())
	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "format")
}

func TestValidate_JUnitAccepted(t *testing.T) {
	cfg := &evalcmd.CLIConfig{Dataset: "ds.json", Format: "junit"}
	require.NoError(t, cfg.ApplyDefaults())
	require.NoError(t, cfg.Validate())
	assert.True(t, cfg.WantsJUnit())
}

func TestValidate_NegativeValues(t *testing.T) {
	tests := map[string]func(*evalcmd.CLIConfig){
		"negative max_rows": func(c *evalcmd.CLIConfig) { c.MaxRows = -1 },
		"negative max_cost": func(c *evalcmd.CLIConfig) { c.MaxCost = -0.01 },
		"zero parallel after apply": func(c *evalcmd.CLIConfig) {
			c.Parallel = 0 // will be coerced to 1; Validate should not trip
		},
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			cfg := &evalcmd.CLIConfig{Dataset: "ds.json"}
			mutate(cfg)
			require.NoError(t, cfg.ApplyDefaults())
			err := cfg.Validate()
			if name == "zero parallel after apply" {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
		})
	}
}

func TestJUnitReportPath_Siblings(t *testing.T) {
	cfg := &evalcmd.CLIConfig{ReportPath: "/tmp/eval-report.json"}
	assert.Equal(t, "/tmp/eval-report.junit.xml", cfg.JUnitReportPath())
}

func TestJUnitReportPath_EmptyWhenNoReport(t *testing.T) {
	cfg := &evalcmd.CLIConfig{}
	assert.Empty(t, cfg.JUnitReportPath())
}
