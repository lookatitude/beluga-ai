// Package core provides tracing and metrics instrumentation for Runnable components.
package core

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// TracedRunnable wraps a Runnable with OpenTelemetry tracing and metrics.
type TracedRunnable struct {
	runnable      Runnable
	tracer        trace.Tracer
	metrics       *Metrics
	componentType string
	componentName string
}

// NewTracedRunnable creates a new TracedRunnable that wraps the given Runnable
// with tracing and metrics instrumentation.
func NewTracedRunnable(
	runnable Runnable,
	tracer trace.Tracer,
	metrics *Metrics,
	componentType string,
	componentName string,
) *TracedRunnable {
	if tracer == nil {
		tracer = trace.NewNoopTracerProvider().Tracer("")
	}
	if metrics == nil {
		metrics = NoOpMetrics()
	}

	return &TracedRunnable{
		runnable:      runnable,
		tracer:        tracer,
		metrics:       metrics,
		componentType: componentType,
		componentName: componentName,
	}
}

// Invoke executes the wrapped Runnable with tracing and metrics.
func (tr *TracedRunnable) Invoke(ctx context.Context, input any, options ...Option) (any, error) {
	ctx, span := tr.tracer.Start(ctx, fmt.Sprintf("%s.invoke", tr.componentType),
		trace.WithAttributes(
			attribute.String("component.type", tr.componentType),
			attribute.String("component.name", tr.componentName),
			attribute.String("operation", "invoke"),
		))
	defer span.End()

	startTime := time.Now()

	result, err := tr.runnable.Invoke(ctx, input, options...)

	duration := time.Since(startTime)

	tr.metrics.RecordRunnableInvoke(ctx, tr.componentType, duration, err)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	} else {
		span.SetStatus(codes.Ok, "")
	}

	return result, err
}

// Batch executes the wrapped Runnable with tracing and metrics.
func (tr *TracedRunnable) Batch(ctx context.Context, inputs []any, options ...Option) ([]any, error) {
	ctx, span := tr.tracer.Start(ctx, fmt.Sprintf("%s.batch", tr.componentType),
		trace.WithAttributes(
			attribute.String("component.type", tr.componentType),
			attribute.String("component.name", tr.componentName),
			attribute.String("operation", "batch"),
			attribute.Int("batch.size", len(inputs)),
		))
	defer span.End()

	startTime := time.Now()

	results, err := tr.runnable.Batch(ctx, inputs, options...)

	duration := time.Since(startTime)

	tr.metrics.RecordRunnableBatch(ctx, tr.componentType, len(inputs), duration, err)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	} else {
		span.SetStatus(codes.Ok, "")
	}

	return results, err
}

// Stream executes the wrapped Runnable with tracing and metrics.
func (tr *TracedRunnable) Stream(ctx context.Context, input any, options ...Option) (<-chan any, error) {
	ctx, span := tr.tracer.Start(ctx, fmt.Sprintf("%s.stream", tr.componentType),
		trace.WithAttributes(
			attribute.String("component.type", tr.componentType),
			attribute.String("component.name", tr.componentName),
			attribute.String("operation", "stream"),
		))
	defer span.End()

	startTime := time.Now()

	streamChan, err := tr.runnable.Stream(ctx, input, options...)

	if err != nil {
		duration := time.Since(startTime)
		tr.metrics.RecordRunnableStream(ctx, tr.componentType, duration, 0, err)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return streamChan, err
	}

	// Create a wrapper channel to count chunks and measure duration
	wrapperChan := make(chan any)
	chunkCount := 0

	go func() {
		defer close(wrapperChan)
		defer func() {
			duration := time.Since(startTime)
			tr.metrics.RecordRunnableStream(ctx, tr.componentType, duration, chunkCount, nil)
			span.SetStatus(codes.Ok, "")
		}()

		for {
			select {
			case <-ctx.Done():
				return
			case chunk, ok := <-streamChan:
				if !ok {
					return
				}
				chunkCount++
				select {
				case wrapperChan <- chunk:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return wrapperChan, nil
}

// RunnableWithTracing is a helper function that creates a TracedRunnable
// with the given component information.
func RunnableWithTracing(
	runnable Runnable,
	tracer trace.Tracer,
	metrics *Metrics,
	componentType string,
) Runnable {
	return NewTracedRunnable(runnable, tracer, metrics, componentType, "")
}

// RunnableWithTracingAndName is a helper function that creates a TracedRunnable
// with the given component information including a name.
func RunnableWithTracingAndName(
	runnable Runnable,
	tracer trace.Tracer,
	metrics *Metrics,
	componentType string,
	componentName string,
) Runnable {
	return NewTracedRunnable(runnable, tracer, metrics, componentType, componentName)
}
