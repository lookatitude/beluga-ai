package eval

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func sampleReport(t *testing.T) *Report {
	t.Helper()
	return &Report{
		RunID:       "run0123456789ab",
		DatasetName: "smoke",
		DatasetPath: "/tmp/smoke.json",
		StartedAt:   time.Date(2026, 4, 21, 0, 0, 0, 0, time.UTC),
		Duration:    1500 * time.Millisecond,
		Samples: []SampleReport{
			{
				Index: 0, RowID: "run0123456789ab-0000",
				Input: "capital of France?", Output: "Paris", Expected: "Paris",
				Scores: map[string]float64{"exact_match": 1.0, "latency": 0.85},
			},
			{
				// Single-metric row: isFailedByExactMatch only fires
				// when exact_match is the sole metric recorded (see
				// render.go:isFailedByExactMatch). Mixed-metric rows
				// defer to the JSON report for pass/fail judgement.
				Index: 1, RowID: "run0123456789ab-0001",
				Input: "capital of Spain?", Output: "Lisbon", Expected: "Madrid",
				Scores: map[string]float64{"exact_match": 0.0},
			},
			{
				Index: 2, RowID: "run0123456789ab-0002",
				Input: "capital of Germany?", Output: "", Expected: "Berlin",
				Scores: map[string]float64{},
				Error:  "child exited: exit status 1",
			},
		},
		Aggregate: map[string]float64{"exact_match": 0.333, "latency": 0.556},
		Errors:    []string{"child exited: exit status 1"},
	}
}

func TestRenderText_DeterministicOrdering(t *testing.T) {
	var buf bytes.Buffer
	require.NoError(t, RenderText(&buf, sampleReport(t)))
	got := buf.String()

	// Header columns appear alphabetically (exact_match before latency).
	hdrIdx := strings.Index(got, "exact_match")
	latIdx := strings.Index(got, "latency")
	require.Positive(t, hdrIdx, "exact_match must be in header")
	require.Positive(t, latIdx, "latency must be in header")
	assert.Less(t, hdrIdx, latIdx, "exact_match column before latency")

	// Aggregate footer lists each metric once, in alpha order.
	assert.Contains(t, got, "aggregate:")
	emIdx := strings.Index(got, "exact_match: 0.3330")
	laIdx := strings.Index(got, "latency: 0.5560")
	require.Positive(t, emIdx)
	require.Positive(t, laIdx)
	assert.Less(t, emIdx, laIdx)

	assert.Contains(t, got, "run_id:  run0123456789ab")
	assert.Contains(t, got, "samples: 3")
	assert.Contains(t, got, "errors: 1")
}

func TestRenderText_DryRun(t *testing.T) {
	rep := &Report{
		RunID:   "dry1234567890ab",
		DryRun:  true,
		Skipped: 2,
		Samples: []SampleReport{},
	}
	var buf bytes.Buffer
	require.NoError(t, RenderText(&buf, rep))
	assert.Contains(t, buf.String(), "dry-run")
	assert.Contains(t, buf.String(), "dry1234567890ab")
}

func TestRenderText_NilReport(t *testing.T) {
	var buf bytes.Buffer
	require.Error(t, RenderText(&buf, nil))
}

func TestRenderJSON_RoundTrips(t *testing.T) {
	original := sampleReport(t)
	var buf bytes.Buffer
	require.NoError(t, RenderJSON(&buf, original))

	var back Report
	require.NoError(t, json.Unmarshal(buf.Bytes(), &back))
	assert.Equal(t, original.RunID, back.RunID)
	assert.Equal(t, original.DatasetName, back.DatasetName)
	assert.Len(t, back.Samples, len(original.Samples))
	assert.Equal(t, original.Aggregate["exact_match"], back.Aggregate["exact_match"])
}

func TestRenderJSON_NilReport(t *testing.T) {
	var buf bytes.Buffer
	require.Error(t, RenderJSON(&buf, nil))
}

func TestRenderJUnit_StructureAndCounts(t *testing.T) {
	var buf bytes.Buffer
	require.NoError(t, RenderJUnit(&buf, sampleReport(t)))
	out := buf.String()

	assert.True(t, strings.HasPrefix(out, xml.Header),
		"JUnit output begins with XML declaration")
	assert.Contains(t, out, `<testsuites name="smoke"`)
	assert.Contains(t, out, `tests="3"`)
	assert.Contains(t, out, `failures="1"`, "one exact_match=0 row → one <failure>")
	assert.Contains(t, out, `errors="1"`, "one exec-error row → one <error>-classified testcase")
	assert.Contains(t, out, `<testcase classname="smoke" name="run0123456789ab-0000"`)
	assert.Contains(t, out, `type="exact_match_fail"`, "row 1 fails exact_match")
	assert.Contains(t, out, `type="exec_error"`, "row 2 recorded an exec error")
	assert.Contains(t, out, `<property name="exact_match" value="1.0000">`)
	assert.Contains(t, out, `<property name="latency" value="0.8500">`)
}

func TestRenderJUnit_NilReport(t *testing.T) {
	var buf bytes.Buffer
	require.Error(t, RenderJUnit(&buf, nil))
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		name string
		in   string
		max  int
		want string
	}{
		{"under", "abc", 10, "abc"},
		{"equal", "abcde", 5, "abcde"},
		{"over", "abcdefghij", 6, "abc..."},
		{"max3", "abcdef", 3, "abc"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, truncate(tt.in, tt.max))
		})
	}
}
