package o11y

import (
	"context"
	"time"
)

// TraceExporter is implemented by backends that capture detailed LLM call data
// for analysis, debugging, or cost tracking. Examples include Langfuse, Arize
// Phoenix, and custom analytics stores.
type TraceExporter interface {
	// ExportLLMCall sends a completed LLM call record to the backend.
	ExportLLMCall(ctx context.Context, data LLMCallData) error
}

// LLMCallData captures the full details of a single LLM invocation for
// export to observability backends.
type LLMCallData struct {
	// Model is the model identifier that served the request (e.g. "gpt-4o").
	Model string

	// Provider is the upstream system (e.g. "openai", "anthropic").
	Provider string

	// InputTokens is the number of prompt tokens consumed.
	InputTokens int

	// OutputTokens is the number of completion tokens produced.
	OutputTokens int

	// Duration is the wall-clock time of the LLM call.
	Duration time.Duration

	// Cost is the estimated monetary cost in USD.
	Cost float64

	// Messages is the input conversation sent to the model, serialised as
	// a slice of generic maps for backend-agnostic export.
	Messages []map[string]any

	// Response is the model's output, serialised as a generic map.
	Response map[string]any

	// Error is non-empty when the LLM call failed.
	Error string

	// Metadata carries additional key-value data such as trace IDs,
	// session IDs, or user-defined labels.
	Metadata map[string]any
}

// MultiExporter fans out LLM call data to multiple TraceExporters. If any
// exporter returns an error, the first error encountered is returned but all
// exporters are still called.
type MultiExporter struct {
	exporters []TraceExporter
}

// NewMultiExporter creates a MultiExporter that writes to all given exporters.
func NewMultiExporter(exporters ...TraceExporter) *MultiExporter {
	return &MultiExporter{exporters: exporters}
}

// ExportLLMCall sends data to every registered exporter. All exporters are
// called even if one returns an error; the first error is returned.
func (m *MultiExporter) ExportLLMCall(ctx context.Context, data LLMCallData) error {
	var firstErr error
	for _, exp := range m.exporters {
		if err := exp.ExportLLMCall(ctx, data); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}
