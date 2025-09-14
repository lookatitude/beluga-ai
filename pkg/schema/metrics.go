package schema

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// Metrics holds all the metrics for the schema package.
type Metrics struct {
	// Message metrics
	messagesCreated metric.Int64Counter
	messageErrors   metric.Int64Counter

	// Document metrics
	documentsCreated metric.Int64Counter
	documentErrors   metric.Int64Counter

	// Chat history metrics
	historyOperations metric.Int64Counter
	historySize       metric.Int64Gauge

	// Agent I/O metrics
	agentActions      metric.Int64Counter
	agentObservations metric.Int64Counter
	stepsCreated      metric.Int64Counter

	// Generation metrics
	generationsCreated  metric.Int64Counter
	llmResponsesCreated metric.Int64Counter

	// Configuration metrics
	configValidations metric.Int64Counter
	configErrors      metric.Int64Counter
}

// NewMetrics creates a new Metrics instance with OpenTelemetry instruments.
func NewMetrics(meter metric.Meter) (*Metrics, error) {
	messagesCreated, err := meter.Int64Counter(
		"schema_messages_created_total",
		metric.WithDescription("Total number of messages created"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	messageErrors, err := meter.Int64Counter(
		"schema_message_errors_total",
		metric.WithDescription("Total number of message creation errors"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	documentsCreated, err := meter.Int64Counter(
		"schema_documents_created_total",
		metric.WithDescription("Total number of documents created"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	documentErrors, err := meter.Int64Counter(
		"schema_document_errors_total",
		metric.WithDescription("Total number of document creation errors"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	historyOperations, err := meter.Int64Counter(
		"schema_history_operations_total",
		metric.WithDescription("Total number of chat history operations"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	historySize, err := meter.Int64Gauge(
		"schema_history_size",
		metric.WithDescription("Current size of chat history"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	agentActions, err := meter.Int64Counter(
		"schema_agent_actions_created_total",
		metric.WithDescription("Total number of agent actions created"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	agentObservations, err := meter.Int64Counter(
		"schema_agent_observations_created_total",
		metric.WithDescription("Total number of agent observations created"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	stepsCreated, err := meter.Int64Counter(
		"schema_steps_created_total",
		metric.WithDescription("Total number of steps created"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	generationsCreated, err := meter.Int64Counter(
		"schema_generations_created_total",
		metric.WithDescription("Total number of generations created"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	llmResponsesCreated, err := meter.Int64Counter(
		"schema_llm_responses_created_total",
		metric.WithDescription("Total number of LLM responses created"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	configValidations, err := meter.Int64Counter(
		"schema_config_validations_total",
		metric.WithDescription("Total number of configuration validations"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	configErrors, err := meter.Int64Counter(
		"schema_config_errors_total",
		metric.WithDescription("Total number of configuration validation errors"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	return &Metrics{
		messagesCreated:     messagesCreated,
		messageErrors:       messageErrors,
		documentsCreated:    documentsCreated,
		documentErrors:      documentErrors,
		historyOperations:   historyOperations,
		historySize:         historySize,
		agentActions:        agentActions,
		agentObservations:   agentObservations,
		stepsCreated:        stepsCreated,
		generationsCreated:  generationsCreated,
		llmResponsesCreated: llmResponsesCreated,
		configValidations:   configValidations,
		configErrors:        configErrors,
	}, nil
}

// NoOpMetrics returns a metrics instance that does nothing.
// Useful for testing or when metrics are disabled.
func NoOpMetrics() *Metrics {
	return &Metrics{}
}

// RecordMessageCreated records a message creation event.
func (m *Metrics) RecordMessageCreated(ctx context.Context, msgType MessageType) {
	if m == nil || m.messagesCreated == nil {
		return
	}
	m.messagesCreated.Add(ctx, 1,
		metric.WithAttributes(attribute.String("message_type", string(msgType))),
	)
}

// RecordMessageError records a message creation error.
func (m *Metrics) RecordMessageError(ctx context.Context, msgType MessageType, err error) {
	if m == nil || m.messageErrors == nil {
		return
	}
	m.messageErrors.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("message_type", string(msgType)),
			attribute.String("error_type", err.Error()),
		),
	)
}

// RecordDocumentCreated records a document creation event.
func (m *Metrics) RecordDocumentCreated(ctx context.Context) {
	if m == nil || m.documentsCreated == nil {
		return
	}
	m.documentsCreated.Add(ctx, 1)
}

// RecordDocumentError records a document creation error.
func (m *Metrics) RecordDocumentError(ctx context.Context, err error) {
	if m == nil || m.documentErrors == nil {
		return
	}
	m.documentErrors.Add(ctx, 1,
		metric.WithAttributes(attribute.String("error_type", err.Error())),
	)
}

// RecordHistoryOperation records a chat history operation.
func (m *Metrics) RecordHistoryOperation(ctx context.Context, operation string) {
	if m == nil || m.historyOperations == nil {
		return
	}
	m.historyOperations.Add(ctx, 1,
		metric.WithAttributes(attribute.String("operation", operation)),
	)
}

// RecordHistorySize records the current size of chat history.
func (m *Metrics) RecordHistorySize(ctx context.Context, size int) {
	if m == nil || m.historySize == nil {
		return
	}
	m.historySize.Record(ctx, int64(size))
}

// RecordAgentActionCreated records an agent action creation.
func (m *Metrics) RecordAgentActionCreated(ctx context.Context, tool string) {
	if m == nil || m.agentActions == nil {
		return
	}
	m.agentActions.Add(ctx, 1,
		metric.WithAttributes(attribute.String("tool", tool)),
	)
}

// RecordAgentObservationCreated records an agent observation creation.
func (m *Metrics) RecordAgentObservationCreated(ctx context.Context) {
	if m == nil || m.agentObservations == nil {
		return
	}
	m.agentObservations.Add(ctx, 1)
}

// RecordStepCreated records a step creation.
func (m *Metrics) RecordStepCreated(ctx context.Context) {
	if m == nil || m.stepsCreated == nil {
		return
	}
	m.stepsCreated.Add(ctx, 1)
}

// RecordGenerationCreated records a generation creation.
func (m *Metrics) RecordGenerationCreated(ctx context.Context) {
	if m == nil || m.generationsCreated == nil {
		return
	}
	m.generationsCreated.Add(ctx, 1)
}

// RecordLLMResponseCreated records an LLM response creation.
func (m *Metrics) RecordLLMResponseCreated(ctx context.Context) {
	if m == nil || m.llmResponsesCreated == nil {
		return
	}
	m.llmResponsesCreated.Add(ctx, 1)
}

// RecordConfigValidation records a configuration validation.
func (m *Metrics) RecordConfigValidation(ctx context.Context, configType string) {
	if m == nil || m.configValidations == nil {
		return
	}
	m.configValidations.Add(ctx, 1,
		metric.WithAttributes(attribute.String("config_type", configType)),
	)
}

// RecordConfigError records a configuration validation error.
func (m *Metrics) RecordConfigError(ctx context.Context, configType string, err error) {
	if m == nil || m.configErrors == nil {
		return
	}
	m.configErrors.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("config_type", configType),
			attribute.String("error_type", err.Error()),
		),
	)
}

// Global metrics instance (can be set by the application)
var globalMetrics *Metrics

// SetGlobalMetrics sets the global metrics instance.
func SetGlobalMetrics(m *Metrics) {
	globalMetrics = m
}

// GetGlobalMetrics returns the global metrics instance.
func GetGlobalMetrics() *Metrics {
	return globalMetrics
}

// Helper functions for recording metrics if global metrics is set

// RecordMessageCreated records a message creation if global metrics is set.
func RecordMessageCreated(ctx context.Context, msgType MessageType) {
	if globalMetrics != nil {
		globalMetrics.RecordMessageCreated(ctx, msgType)
	}
}

// RecordMessageError records a message error if global metrics is set.
func RecordMessageError(ctx context.Context, msgType MessageType, err error) {
	if globalMetrics != nil {
		globalMetrics.RecordMessageError(ctx, msgType, err)
	}
}

// RecordDocumentCreated records a document creation if global metrics is set.
func RecordDocumentCreated(ctx context.Context) {
	if globalMetrics != nil {
		globalMetrics.RecordDocumentCreated(ctx)
	}
}

// RecordDocumentError records a document error if global metrics is set.
func RecordDocumentError(ctx context.Context, err error) {
	if globalMetrics != nil {
		globalMetrics.RecordDocumentError(ctx, err)
	}
}

// RecordHistoryOperation records a history operation if global metrics is set.
func RecordHistoryOperation(ctx context.Context, operation string) {
	if globalMetrics != nil {
		globalMetrics.RecordHistoryOperation(ctx, operation)
	}
}

// RecordHistorySize records history size if global metrics is set.
func RecordHistorySize(ctx context.Context, size int) {
	if globalMetrics != nil {
		globalMetrics.RecordHistorySize(ctx, size)
	}
}

// RecordAgentActionCreated records an agent action if global metrics is set.
func RecordAgentActionCreated(ctx context.Context, tool string) {
	if globalMetrics != nil {
		globalMetrics.RecordAgentActionCreated(ctx, tool)
	}
}

// RecordAgentObservationCreated records an agent observation if global metrics is set.
func RecordAgentObservationCreated(ctx context.Context) {
	if globalMetrics != nil {
		globalMetrics.RecordAgentObservationCreated(ctx)
	}
}

// RecordStepCreated records a step creation if global metrics is set.
func RecordStepCreated(ctx context.Context) {
	if globalMetrics != nil {
		globalMetrics.RecordStepCreated(ctx)
	}
}

// RecordGenerationCreated records a generation creation if global metrics is set.
func RecordGenerationCreated(ctx context.Context) {
	if globalMetrics != nil {
		globalMetrics.RecordGenerationCreated(ctx)
	}
}

// RecordLLMResponseCreated records an LLM response creation if global metrics is set.
func RecordLLMResponseCreated(ctx context.Context) {
	if globalMetrics != nil {
		globalMetrics.RecordLLMResponseCreated(ctx)
	}
}

// RecordConfigValidation records a config validation if global metrics is set.
func RecordConfigValidation(ctx context.Context, configType string) {
	if globalMetrics != nil {
		globalMetrics.RecordConfigValidation(ctx, configType)
	}
}

// RecordConfigError records a config error if global metrics is set.
func RecordConfigError(ctx context.Context, configType string, err error) {
	if globalMetrics != nil {
		globalMetrics.RecordConfigError(ctx, configType, err)
	}
}
