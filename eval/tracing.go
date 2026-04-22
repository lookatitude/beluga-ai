package eval

import (
	"context"
	"sync"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"

	"github.com/lookatitude/beluga-ai/v2/o11y"
)

const (
	attrEvalMetricName   = "gen_ai.evaluation.name"
	attrEvalScoreValue   = "gen_ai.evaluation.score.value"
	attrBelugaMetricName = "beluga.eval.metric_name"
	attrBelugaDataset    = "beluga.eval.dataset"
	attrBelugaRowIndex   = "beluga.eval.row_index"
	attrBelugaSampleCt   = "beluga.eval.sample_count"
	attrBelugaMetricCt   = "beluga.eval.metric_count"

	evalResultEventName   = "gen_ai.evaluation.result"
	metricScoreInstrument = "beluga.eval.metric.score"

	runSpanName = "eval.run"
	rowSpanName = "eval.row"
)

var (
	evalMeter           = otel.Meter("github.com/lookatitude/beluga-ai/v2/eval")
	evalMetricScoreMu   sync.Mutex
	evalMetricScoreInst metric.Float64Histogram
)

func metricScoreHistogram() metric.Float64Histogram {
	evalMetricScoreMu.Lock()
	defer evalMetricScoreMu.Unlock()
	if evalMetricScoreInst != nil {
		return evalMetricScoreInst
	}
	h, err := evalMeter.Float64Histogram(
		metricScoreInstrument,
		metric.WithDescription("Per-metric evaluation score in [0, 1]"),
	)
	if err == nil {
		evalMetricScoreInst = h
	}
	return evalMetricScoreInst
}

// recordMetricScore records a successful metric score on the beluga.eval.metric.score
// Histogram with only metric_name + dataset label dimensions — never row_id or
// row_index — per the brief's cardinality constraint.
func recordMetricScore(ctx context.Context, metricName, datasetName string, score float64) {
	h := metricScoreHistogram()
	if h == nil {
		return
	}
	h.Record(ctx, score,
		metric.WithAttributes(
			attribute.String(attrBelugaMetricName, metricName),
			attribute.String(attrBelugaDataset, datasetName),
		),
	)
}

// recordEvalResult emits a gen_ai.evaluation.result event on the span currently
// carried by ctx, per the OTel GenAI evaluation semantic conventions.
func recordEvalResult(ctx context.Context, metricName string, score float64) {
	span := trace.SpanFromContext(ctx)
	if !span.SpanContext().IsValid() {
		return
	}
	span.AddEvent(evalResultEventName, trace.WithAttributes(
		attribute.String(attrEvalMetricName, metricName),
		attribute.Float64(attrEvalScoreValue, score),
	))
}

// startRunSpan opens the top-level eval.run span at runner entry.
func startRunSpan(ctx context.Context, dataset string, samples, metrics int) (context.Context, o11y.Span) {
	return o11y.StartSpan(ctx, runSpanName, o11y.Attrs{
		o11y.AttrOperationName: "eval",
		attrBelugaDataset:      dataset,
		attrBelugaSampleCt:     samples,
		attrBelugaMetricCt:     metrics,
	})
}

// startRowSpan opens the per-row eval.row span inside the eval.run span.
func startRowSpan(ctx context.Context, dataset string, rowIndex int) (context.Context, o11y.Span) {
	return o11y.StartSpan(ctx, rowSpanName, o11y.Attrs{
		o11y.AttrOperationName: "eval",
		attrBelugaDataset:      dataset,
		attrBelugaRowIndex:     rowIndex,
	})
}
