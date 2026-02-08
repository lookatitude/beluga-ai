package o11y

import (
	"context"
	"sync"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// meter holds the package-level OTel meter used by metric recording functions.
var meter metric.Meter

// Pre-registered GenAI instruments following OTel GenAI semantic conventions.
var (
	inputTokenCounter  metric.Int64Counter
	outputTokenCounter metric.Int64Counter
	operationDuration  metric.Float64Histogram
	costGauge          metric.Float64Counter

	meterOnce sync.Once
	meterErr  error
)

func init() {
	meter = otel.Meter("github.com/lookatitude/beluga-ai/o11y")
}

// initInstruments lazily creates the pre-defined metric instruments. This is
// deferred so callers can configure the meter provider before first use.
func initInstruments() error {
	meterOnce.Do(func() {
		var err error

		inputTokenCounter, err = meter.Int64Counter(
			"gen_ai.client.token.usage",
			metric.WithDescription("Number of tokens used by GenAI operations"),
			metric.WithUnit("{token}"),
		)
		if err != nil {
			meterErr = err
			return
		}

		outputTokenCounter, err = meter.Int64Counter(
			"gen_ai.client.token.usage.output",
			metric.WithDescription("Number of output tokens produced"),
			metric.WithUnit("{token}"),
		)
		if err != nil {
			meterErr = err
			return
		}

		operationDuration, err = meter.Float64Histogram(
			"gen_ai.client.operation.duration",
			metric.WithDescription("Duration of GenAI operations"),
			metric.WithUnit("ms"),
		)
		if err != nil {
			meterErr = err
			return
		}

		costGauge, err = meter.Float64Counter(
			"gen_ai.client.estimated_cost",
			metric.WithDescription("Estimated cost of GenAI operations"),
			metric.WithUnit("USD"),
		)
		if err != nil {
			meterErr = err
			return
		}
	})
	return meterErr
}

// InitMeter configures the package-level meter with the given service name.
// This should be called after setting up the OTel meter provider. If not called,
// the default global meter provider is used.
func InitMeter(serviceName string) error {
	meter = otel.Meter(
		"github.com/lookatitude/beluga-ai/o11y",
		metric.WithInstrumentationAttributes(
			attribute.String("service.name", serviceName),
		),
	)
	// Reset so instruments are re-created with the new meter.
	meterOnce = sync.Once{}
	meterErr = nil
	return initInstruments()
}

// TokenUsage records the number of input and output tokens consumed by a
// GenAI operation.
func TokenUsage(ctx context.Context, input, output int) {
	if err := initInstruments(); err != nil {
		return
	}
	inputTokenCounter.Add(ctx, int64(input),
		metric.WithAttributes(attribute.String("gen_ai.token.type", "input")),
	)
	outputTokenCounter.Add(ctx, int64(output),
		metric.WithAttributes(attribute.String("gen_ai.token.type", "output")),
	)
}

// OperationDuration records the duration of a GenAI operation in milliseconds.
func OperationDuration(ctx context.Context, durationMs float64) {
	if err := initInstruments(); err != nil {
		return
	}
	operationDuration.Record(ctx, durationMs)
}

// Cost records the estimated monetary cost of a GenAI operation in USD.
func Cost(ctx context.Context, cost float64) {
	if err := initInstruments(); err != nil {
		return
	}
	costGauge.Add(ctx, cost)
}

// Counter records an increment to a named counter metric.
func Counter(ctx context.Context, name string, value int64) {
	c, err := meter.Int64Counter(name)
	if err != nil {
		return
	}
	c.Add(ctx, value)
}

// Histogram records a value to a named histogram metric.
func Histogram(ctx context.Context, name string, value float64) {
	h, err := meter.Float64Histogram(name)
	if err != nil {
		return
	}
	h.Record(ctx, value)
}
