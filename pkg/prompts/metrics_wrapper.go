// Package prompts provides a wrapper to make OTEL metrics compatible with the prompts interface
package prompts

import (
	"context"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/prompts/iface"
)

// MetricsWrapper wraps the OTEL Metrics to implement the iface.Metrics interface
type MetricsWrapper struct {
	otelMetrics *Metrics
}

// Template metrics implementation
func (m *MetricsWrapper) RecordTemplateCreated(templateType string) {
	if m.otelMetrics != nil {
		m.otelMetrics.RecordTemplateCreated(context.Background(), templateType)
	}
}

func (m *MetricsWrapper) RecordTemplateExecuted(templateName string, duration float64) {
	if m.otelMetrics != nil {
		m.otelMetrics.RecordTemplateExecuted(context.Background(), templateName,
			time.Duration(duration*float64(time.Second)), true)
	}
}

func (m *MetricsWrapper) RecordTemplateError(templateName string, errorType string) {
	if m.otelMetrics != nil {
		m.otelMetrics.RecordTemplateError(context.Background(), templateName, errorType)
	}
}

// Formatting metrics implementation
func (m *MetricsWrapper) RecordFormattingRequest(adapterType string, duration float64) {
	if m.otelMetrics != nil {
		m.otelMetrics.RecordFormattingRequest(context.Background(), adapterType,
			time.Duration(duration*float64(time.Second)), true)
	}
}

func (m *MetricsWrapper) RecordFormattingError(adapterType string, errorType string) {
	if m.otelMetrics != nil {
		m.otelMetrics.RecordFormattingError(context.Background(), adapterType, errorType)
	}
}

// Variable validation metrics implementation
func (m *MetricsWrapper) RecordValidationRequest() {
	if m.otelMetrics != nil {
		m.otelMetrics.RecordValidationRequest(context.Background(), "template_validation", true)
	}
}

func (m *MetricsWrapper) RecordValidationError(errorType string) {
	if m.otelMetrics != nil {
		m.otelMetrics.RecordValidationError(context.Background(), errorType)
	}
}

// Cache metrics implementation
func (m *MetricsWrapper) RecordCacheHit() {
	if m.otelMetrics != nil {
		m.otelMetrics.RecordCacheHit(context.Background(), "template_cache")
	}
}

func (m *MetricsWrapper) RecordCacheMiss() {
	if m.otelMetrics != nil {
		m.otelMetrics.RecordCacheMiss(context.Background(), "template_cache")
	}
}

func (m *MetricsWrapper) RecordCacheSize(size int64) {
	if m.otelMetrics != nil {
		m.otelMetrics.RecordCacheSize(context.Background(), size, "template_cache")
	}
}

// Adapter metrics implementation
func (m *MetricsWrapper) RecordAdapterRequest(adapterType string) {
	if m.otelMetrics != nil {
		m.otelMetrics.RecordAdapterRequest(context.Background(), adapterType, true)
	}
}

func (m *MetricsWrapper) RecordAdapterError(adapterType string, errorType string) {
	if m.otelMetrics != nil {
		m.otelMetrics.RecordAdapterError(context.Background(), adapterType, errorType)
	}
}

// Verify that MetricsWrapper implements iface.Metrics
var _ iface.Metrics = (*MetricsWrapper)(nil)
