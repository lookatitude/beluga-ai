package eval

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/lookatitude/beluga-ai/v2/o11y"
)

// stubMetric scores every sample with a fixed value and name. Used to drive
// the OTel emission tests without touching real providers.
type stubMetric struct {
	name  string
	score float64
	err   error
}

func (s *stubMetric) Name() string { return s.name }
func (s *stubMetric) Score(_ context.Context, _ EvalSample) (float64, error) {
	return s.score, s.err
}

// installSpanRecorder wires an in-memory OTel span exporter for the duration
// of the test and returns the exporter for assertions.
func installSpanRecorder(t *testing.T) *tracetest.InMemoryExporter {
	t.Helper()
	exporter := tracetest.NewInMemoryExporter()
	shutdown, err := o11y.InitTracer("eval-test",
		o11y.WithSpanExporter(exporter),
		o11y.WithSyncExport(),
	)
	require.NoError(t, err)
	t.Cleanup(shutdown)
	return exporter
}

// installMetricReader swaps the eval package's meter for an in-memory reader
// and resets the histogram registration so the next record goes through the
// reader. Restores global state on test cleanup.
func installMetricReader(t *testing.T) *sdkmetric.ManualReader {
	t.Helper()
	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))

	prevMeter := evalMeter
	prevInst := evalMetricScoreInst

	evalMeter = provider.Meter("github.com/lookatitude/beluga-ai/v2/eval")
	evalMetricScoreInst = nil

	t.Cleanup(func() {
		evalMeter = prevMeter
		evalMetricScoreInst = prevInst
		_ = provider.Shutdown(context.Background())
	})
	return reader
}

func attrString(attrs []attribute.KeyValue, key string) (string, bool) {
	for _, a := range attrs {
		if string(a.Key) == key {
			return a.Value.AsString(), true
		}
	}
	return "", false
}

func attrInt64(attrs []attribute.KeyValue, key string) (int64, bool) {
	for _, a := range attrs {
		if string(a.Key) == key {
			return a.Value.AsInt64(), true
		}
	}
	return 0, false
}

func attrFloat64(attrs []attribute.KeyValue, key string) (float64, bool) {
	for _, a := range attrs {
		if string(a.Key) == key {
			return a.Value.AsFloat64(), true
		}
	}
	return 0, false
}

func TestRun_EmitsEvalRunAndEvalRowSpans(t *testing.T) {
	exporter := installSpanRecorder(t)

	runner := NewRunner(
		WithDataset([]EvalSample{{Input: "q1", Output: "a1"}, {Input: "q2", Output: "a2"}}),
		WithMetrics(&stubMetric{name: "exact_match", score: 1.0}),
		WithDatasetName("ds-otel"),
	)

	_, err := runner.Run(context.Background())
	require.NoError(t, err)

	spans := exporter.GetSpans()

	var runCount, rowCount int
	for _, s := range spans {
		switch s.Name {
		case runSpanName:
			runCount++
			op, ok := attrString(s.Attributes, o11y.AttrOperationName)
			assert.True(t, ok)
			assert.Equal(t, "eval", op)
			ds, _ := attrString(s.Attributes, attrBelugaDataset)
			assert.Equal(t, "ds-otel", ds)
			samples, _ := attrInt64(s.Attributes, attrBelugaSampleCt)
			assert.Equal(t, int64(2), samples)
			metrics, _ := attrInt64(s.Attributes, attrBelugaMetricCt)
			assert.Equal(t, int64(1), metrics)
		case rowSpanName:
			rowCount++
			op, _ := attrString(s.Attributes, o11y.AttrOperationName)
			assert.Equal(t, "eval", op)
			ds, _ := attrString(s.Attributes, attrBelugaDataset)
			assert.Equal(t, "ds-otel", ds)
		}
	}

	assert.Equal(t, 1, runCount, "exactly one eval.run span")
	assert.Equal(t, 2, rowCount, "one eval.row span per sample")
}

func TestRun_EmitsGenAIEvaluationResultEventsPerMetric(t *testing.T) {
	exporter := installSpanRecorder(t)

	runner := NewRunner(
		WithDataset([]EvalSample{{Input: "q", Output: "a"}}),
		WithMetrics(
			&stubMetric{name: "exact_match", score: 1.0},
			&stubMetric{name: "pass_rate", score: 0.5},
		),
		WithDatasetName("ds-events"),
	)
	_, err := runner.Run(context.Background())
	require.NoError(t, err)

	var rowSpan *tracetest.SpanStub
	spans := exporter.GetSpans()
	for i := range spans {
		if spans[i].Name == rowSpanName {
			rowSpan = &spans[i]
			break
		}
	}
	require.NotNil(t, rowSpan, "eval.row span must be emitted")

	require.Len(t, rowSpan.Events, 2, "one gen_ai.evaluation.result event per metric")

	got := map[string]float64{}
	for _, ev := range rowSpan.Events {
		assert.Equal(t, evalResultEventName, ev.Name)
		metricName, ok := attrString(ev.Attributes, attrEvalMetricName)
		require.True(t, ok, "event missing gen_ai.evaluation.name")
		score, ok := attrFloat64(ev.Attributes, attrEvalScoreValue)
		require.True(t, ok, "event missing gen_ai.evaluation.score.value")
		got[metricName] = score
	}
	assert.Equal(t, 1.0, got["exact_match"])
	assert.Equal(t, 0.5, got["pass_rate"])
}

func TestRun_RecordsBelugaEvalMetricScoreHistogram(t *testing.T) {
	installSpanRecorder(t)
	reader := installMetricReader(t)

	runner := NewRunner(
		WithDataset([]EvalSample{{Input: "q1"}, {Input: "q2"}, {Input: "q3"}}),
		WithMetrics(
			&stubMetric{name: "exact_match", score: 1.0},
			&stubMetric{name: "pass_rate", score: 0.5},
		),
		WithDatasetName("ds-histo"),
	)
	_, err := runner.Run(context.Background())
	require.NoError(t, err)

	rm := metricdata.ResourceMetrics{}
	require.NoError(t, reader.Collect(context.Background(), &rm))

	histo := findHistogram(rm, metricScoreInstrument)
	require.NotNil(t, histo, "beluga.eval.metric.score histogram must be recorded")

	h, ok := histo.Data.(metricdata.Histogram[float64])
	require.True(t, ok, "expected Float64 histogram data")

	require.Len(t, h.DataPoints, 2, "exactly two (metric_name, dataset) label combinations for two metrics on one dataset")

	for _, dp := range h.DataPoints {
		dsAttr, ok := dp.Attributes.Value(attribute.Key(attrBelugaDataset))
		require.True(t, ok)
		metricAttr, ok := dp.Attributes.Value(attribute.Key(attrBelugaMetricName))
		require.True(t, ok)
		assert.Equal(t, "ds-histo", dsAttr.AsString())
		assert.Contains(t, []string{"exact_match", "pass_rate"}, metricAttr.AsString())
		assert.Equal(t, uint64(3), dp.Count, "three samples recorded per (metric, dataset) bucket")
	}
}

func TestRun_HistogramCardinality_StaysBoundedAtDatasetAndMetricOnly(t *testing.T) {
	// Success criterion: cardinality test asserts <10 unique label combinations
	// for a 1000-row fixture with two metrics. Expect exactly 2.
	installSpanRecorder(t)
	reader := installMetricReader(t)

	samples := make([]EvalSample, 1000)
	for i := range samples {
		samples[i] = EvalSample{Input: fmt.Sprintf("q%d", i)}
	}

	runner := NewRunner(
		WithDataset(samples),
		WithMetrics(
			&stubMetric{name: "exact_match", score: 1.0},
			&stubMetric{name: "pass_rate", score: 1.0},
		),
		WithDatasetName("ds-cardinality"),
		WithParallel(4),
	)
	_, err := runner.Run(context.Background())
	require.NoError(t, err)

	rm := metricdata.ResourceMetrics{}
	require.NoError(t, reader.Collect(context.Background(), &rm))

	histo := findHistogram(rm, metricScoreInstrument)
	require.NotNil(t, histo)
	h := histo.Data.(metricdata.Histogram[float64])
	assert.Less(t, len(h.DataPoints), 10, "histogram cardinality must stay bounded: <10 combinations for 1000 rows × 2 metrics")
	assert.Equal(t, 2, len(h.DataPoints), "expected exactly 2 (metric, dataset) combinations")
}

func TestRun_SetsErrorStatusOnSampleFailure(t *testing.T) {
	exporter := installSpanRecorder(t)

	runner := NewRunner(
		WithDataset([]EvalSample{{Input: "q"}}),
		WithMetrics(&stubMetric{name: "exact_match", err: errors.New("forced metric failure")}),
		WithDatasetName("ds-err"),
	)
	_, err := runner.Run(context.Background())
	require.NoError(t, err)

	var sawRowErr, sawRunErr bool
	for _, s := range exporter.GetSpans() {
		if s.Name == rowSpanName && s.Status.Code.String() == "Error" {
			sawRowErr = true
		}
		if s.Name == runSpanName && s.Status.Code.String() == "Error" {
			sawRunErr = true
		}
	}
	assert.True(t, sawRowErr, "eval.row span must record Error status on metric failure")
	assert.True(t, sawRunErr, "eval.run span must propagate Error status when any sample fails")
}

// findHistogram returns a pointer to the histogram metric with the given name
// from the ResourceMetrics, or nil if not present.
func findHistogram(rm metricdata.ResourceMetrics, name string) *metricdata.Metrics {
	for i := range rm.ScopeMetrics {
		sm := &rm.ScopeMetrics[i]
		for j := range sm.Metrics {
			if sm.Metrics[j].Name == name {
				return &sm.Metrics[j]
			}
		}
	}
	return nil
}
