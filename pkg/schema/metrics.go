package schema

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
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

	// A2A Communication metrics
	agentMessagesSent      metric.Int64Counter
	agentMessagesReceived  metric.Int64Counter
	agentRequestsSent      metric.Int64Counter
	agentRequestsReceived  metric.Int64Counter
	agentResponsesSent     metric.Int64Counter
	agentResponsesReceived metric.Int64Counter
	a2aCommunicationErrors metric.Int64Counter

	// Event metrics
	eventsPublished       metric.Int64Counter
	eventsConsumed        metric.Int64Counter
	agentLifecycleEvents  metric.Int64Counter
	taskEvents            metric.Int64Counter
	workflowEvents        metric.Int64Counter
	eventProcessingErrors metric.Int64Counter

	// Validation metrics
	schemaValidations        metric.Int64Counter
	schemaValidationErrors   metric.Int64Counter
	messageValidations       metric.Int64Counter
	messageValidationErrors  metric.Int64Counter
	documentValidations      metric.Int64Counter
	documentValidationErrors metric.Int64Counter

	// Factory metrics
	factoryCreations metric.Int64Counter
	factoryErrors    metric.Int64Counter

	// Tracer for span creation
	tracer trace.Tracer
}

// NewMetrics creates a new Metrics instance with OpenTelemetry instruments.
func NewMetrics(meter metric.Meter, tracer trace.Tracer) (*Metrics, error) {
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

	// A2A Communication metrics
	agentMessagesSent, err := meter.Int64Counter(
		"schema_agent_messages_sent_total",
		metric.WithDescription("Total number of agent messages sent"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	agentMessagesReceived, err := meter.Int64Counter(
		"schema_agent_messages_received_total",
		metric.WithDescription("Total number of agent messages received"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	agentRequestsSent, err := meter.Int64Counter(
		"schema_agent_requests_sent_total",
		metric.WithDescription("Total number of agent requests sent"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	agentRequestsReceived, err := meter.Int64Counter(
		"schema_agent_requests_received_total",
		metric.WithDescription("Total number of agent requests received"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	agentResponsesSent, err := meter.Int64Counter(
		"schema_agent_responses_sent_total",
		metric.WithDescription("Total number of agent responses sent"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	agentResponsesReceived, err := meter.Int64Counter(
		"schema_agent_responses_received_total",
		metric.WithDescription("Total number of agent responses received"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	a2aCommunicationErrors, err := meter.Int64Counter(
		"schema_a2a_communication_errors_total",
		metric.WithDescription("Total number of A2A communication errors"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	// Event metrics
	eventsPublished, err := meter.Int64Counter(
		"schema_events_published_total",
		metric.WithDescription("Total number of events published"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	eventsConsumed, err := meter.Int64Counter(
		"schema_events_consumed_total",
		metric.WithDescription("Total number of events consumed"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	agentLifecycleEvents, err := meter.Int64Counter(
		"schema_agent_lifecycle_events_total",
		metric.WithDescription("Total number of agent lifecycle events"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	taskEvents, err := meter.Int64Counter(
		"schema_task_events_total",
		metric.WithDescription("Total number of task events"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	workflowEvents, err := meter.Int64Counter(
		"schema_workflow_events_total",
		metric.WithDescription("Total number of workflow events"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	eventProcessingErrors, err := meter.Int64Counter(
		"schema_event_processing_errors_total",
		metric.WithDescription("Total number of event processing errors"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	// Validation metrics
	schemaValidations, err := meter.Int64Counter(
		"schema_validations_total",
		metric.WithDescription("Total number of schema validations"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	schemaValidationErrors, err := meter.Int64Counter(
		"schema_validation_errors_total",
		metric.WithDescription("Total number of schema validation errors"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	messageValidations, err := meter.Int64Counter(
		"schema_message_validations_total",
		metric.WithDescription("Total number of message validations"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	messageValidationErrors, err := meter.Int64Counter(
		"schema_message_validation_errors_total",
		metric.WithDescription("Total number of message validation errors"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	documentValidations, err := meter.Int64Counter(
		"schema_document_validations_total",
		metric.WithDescription("Total number of document validations"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	documentValidationErrors, err := meter.Int64Counter(
		"schema_document_validation_errors_total",
		metric.WithDescription("Total number of document validation errors"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	// Factory metrics
	factoryCreations, err := meter.Int64Counter(
		"schema_factory_creations_total",
		metric.WithDescription("Total number of objects created via factories"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	factoryErrors, err := meter.Int64Counter(
		"schema_factory_errors_total",
		metric.WithDescription("Total number of factory creation errors"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, err
	}

	if tracer == nil {
		tracer = otel.Tracer("github.com/lookatitude/beluga-ai/pkg/schema")
	}

	return &Metrics{
		messagesCreated:          messagesCreated,
		messageErrors:            messageErrors,
		documentsCreated:         documentsCreated,
		documentErrors:           documentErrors,
		historyOperations:        historyOperations,
		historySize:              historySize,
		agentActions:             agentActions,
		agentObservations:        agentObservations,
		stepsCreated:             stepsCreated,
		generationsCreated:       generationsCreated,
		llmResponsesCreated:      llmResponsesCreated,
		configValidations:        configValidations,
		configErrors:             configErrors,
		agentMessagesSent:        agentMessagesSent,
		agentMessagesReceived:    agentMessagesReceived,
		agentRequestsSent:        agentRequestsSent,
		agentRequestsReceived:    agentRequestsReceived,
		agentResponsesSent:       agentResponsesSent,
		agentResponsesReceived:   agentResponsesReceived,
		a2aCommunicationErrors:   a2aCommunicationErrors,
		eventsPublished:          eventsPublished,
		eventsConsumed:           eventsConsumed,
		agentLifecycleEvents:     agentLifecycleEvents,
		taskEvents:               taskEvents,
		workflowEvents:           workflowEvents,
		eventProcessingErrors:    eventProcessingErrors,
		schemaValidations:        schemaValidations,
		schemaValidationErrors:   schemaValidationErrors,
		messageValidations:       messageValidations,
		messageValidationErrors:  messageValidationErrors,
		documentValidations:      documentValidations,
		documentValidationErrors: documentValidationErrors,
		factoryCreations:         factoryCreations,
		factoryErrors:            factoryErrors,
		tracer:                   tracer,
	}, nil
}

// NoOpMetrics returns a metrics instance that does nothing.
// Useful for testing or when metrics are disabled.
func NoOpMetrics() *Metrics {
	return &Metrics{
		tracer: trace.NewNoopTracerProvider().Tracer("schema"),
	}
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

// A2A Communication metrics recording methods

// RecordAgentMessageSent records an agent message being sent.
func (m *Metrics) RecordAgentMessageSent(ctx context.Context, messageType, fromAgentID string) {
	if m == nil || m.agentMessagesSent == nil {
		return
	}
	m.agentMessagesSent.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("message_type", messageType),
			attribute.String("from_agent_id", fromAgentID),
		),
	)
}

// RecordAgentMessageReceived records an agent message being received.
func (m *Metrics) RecordAgentMessageReceived(ctx context.Context, messageType, toAgentID string) {
	if m == nil || m.agentMessagesReceived == nil {
		return
	}
	m.agentMessagesReceived.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("message_type", messageType),
			attribute.String("to_agent_id", toAgentID),
		),
	)
}

// RecordAgentRequestSent records an agent request being sent.
func (m *Metrics) RecordAgentRequestSent(ctx context.Context, action, fromAgentID string) {
	if m == nil || m.agentRequestsSent == nil {
		return
	}
	m.agentRequestsSent.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("action", action),
			attribute.String("from_agent_id", fromAgentID),
		),
	)
}

// RecordAgentRequestReceived records an agent request being received.
func (m *Metrics) RecordAgentRequestReceived(ctx context.Context, action, toAgentID string) {
	if m == nil || m.agentRequestsReceived == nil {
		return
	}
	m.agentRequestsReceived.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("action", action),
			attribute.String("to_agent_id", toAgentID),
		),
	)
}

// RecordAgentResponseSent records an agent response being sent.
func (m *Metrics) RecordAgentResponseSent(ctx context.Context, status, toAgentID string) {
	if m == nil || m.agentResponsesSent == nil {
		return
	}
	m.agentResponsesSent.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("status", status),
			attribute.String("to_agent_id", toAgentID),
		),
	)
}

// RecordAgentResponseReceived records an agent response being received.
func (m *Metrics) RecordAgentResponseReceived(ctx context.Context, status, fromAgentID string) {
	if m == nil || m.agentResponsesReceived == nil {
		return
	}
	m.agentResponsesReceived.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("status", status),
			attribute.String("from_agent_id", fromAgentID),
		),
	)
}

// RecordA2ACommunicationError records an A2A communication error.
func (m *Metrics) RecordA2ACommunicationError(ctx context.Context, operation string, err error) {
	if m == nil || m.a2aCommunicationErrors == nil {
		return
	}
	m.a2aCommunicationErrors.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("operation", operation),
			attribute.String("error_type", err.Error()),
		),
	)
}

// Event metrics recording methods

// RecordEventPublished records an event being published.
func (m *Metrics) RecordEventPublished(ctx context.Context, eventType, source string) {
	if m == nil || m.eventsPublished == nil {
		return
	}
	m.eventsPublished.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("event_type", eventType),
			attribute.String("source", source),
		),
	)
}

// RecordEventConsumed records an event being consumed.
func (m *Metrics) RecordEventConsumed(ctx context.Context, eventType, consumer string) {
	if m == nil || m.eventsConsumed == nil {
		return
	}
	m.eventsConsumed.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("event_type", eventType),
			attribute.String("consumer", consumer),
		),
	)
}

// RecordAgentLifecycleEvent records an agent lifecycle event.
func (m *Metrics) RecordAgentLifecycleEvent(ctx context.Context, agentID, eventType string) {
	if m == nil || m.agentLifecycleEvents == nil {
		return
	}
	m.agentLifecycleEvents.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("agent_id", agentID),
			attribute.String("event_type", eventType),
		),
	)
}

// RecordTaskEvent records a task event.
func (m *Metrics) RecordTaskEvent(ctx context.Context, taskID, eventType string) {
	if m == nil || m.taskEvents == nil {
		return
	}
	m.taskEvents.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("task_id", taskID),
			attribute.String("event_type", eventType),
		),
	)
}

// RecordWorkflowEvent records a workflow event.
func (m *Metrics) RecordWorkflowEvent(ctx context.Context, workflowID, eventType string) {
	if m == nil || m.workflowEvents == nil {
		return
	}
	m.workflowEvents.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("workflow_id", workflowID),
			attribute.String("event_type", eventType),
		),
	)
}

// RecordEventProcessingError records an event processing error.
func (m *Metrics) RecordEventProcessingError(ctx context.Context, eventType string, err error) {
	if m == nil || m.eventProcessingErrors == nil {
		return
	}
	m.eventProcessingErrors.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("event_type", eventType),
			attribute.String("error_type", err.Error()),
		),
	)
}

// Validation metrics recording methods

// RecordSchemaValidation records a schema validation.
func (m *Metrics) RecordSchemaValidation(ctx context.Context, validationType string) {
	if m == nil || m.schemaValidations == nil {
		return
	}
	m.schemaValidations.Add(ctx, 1,
		metric.WithAttributes(attribute.String("validation_type", validationType)),
	)
}

// RecordSchemaValidationError records a schema validation error.
func (m *Metrics) RecordSchemaValidationError(ctx context.Context, validationType string, err error) {
	if m == nil || m.schemaValidationErrors == nil {
		return
	}
	m.schemaValidationErrors.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("validation_type", validationType),
			attribute.String("error_type", err.Error()),
		),
	)
}

// RecordMessageValidation records a message validation.
func (m *Metrics) RecordMessageValidation(ctx context.Context, msgType MessageType) {
	if m == nil || m.messageValidations == nil {
		return
	}
	m.messageValidations.Add(ctx, 1,
		metric.WithAttributes(attribute.String("message_type", string(msgType))),
	)
}

// RecordMessageValidationError records a message validation error.
func (m *Metrics) RecordMessageValidationError(ctx context.Context, msgType MessageType, err error) {
	if m == nil || m.messageValidationErrors == nil {
		return
	}
	m.messageValidationErrors.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("message_type", string(msgType)),
			attribute.String("error_type", err.Error()),
		),
	)
}

// RecordDocumentValidation records a document validation.
func (m *Metrics) RecordDocumentValidation(ctx context.Context) {
	if m == nil || m.documentValidations == nil {
		return
	}
	m.documentValidations.Add(ctx, 1)
}

// RecordDocumentValidationError records a document validation error.
func (m *Metrics) RecordDocumentValidationError(ctx context.Context, err error) {
	if m == nil || m.documentValidationErrors == nil {
		return
	}
	m.documentValidationErrors.Add(ctx, 1,
		metric.WithAttributes(attribute.String("error_type", err.Error())),
	)
}

// Factory metrics recording methods

// RecordFactoryCreation records a factory creation.
func (m *Metrics) RecordFactoryCreation(ctx context.Context, objectType string) {
	if m == nil || m.factoryCreations == nil {
		return
	}
	m.factoryCreations.Add(ctx, 1,
		metric.WithAttributes(attribute.String("object_type", objectType)),
	)
}

// RecordFactoryError records a factory creation error.
func (m *Metrics) RecordFactoryError(ctx context.Context, objectType string, err error) {
	if m == nil || m.factoryErrors == nil {
		return
	}
	m.factoryErrors.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("object_type", objectType),
			attribute.String("error_type", err.Error()),
		),
	)
}

// Global metrics instance (can be set by the application).
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

// Global helper functions for A2A Communication metrics

// RecordAgentMessageSent records an agent message sent if global metrics is set.
func RecordAgentMessageSent(ctx context.Context, messageType, fromAgentID string) {
	if globalMetrics != nil {
		globalMetrics.RecordAgentMessageSent(ctx, messageType, fromAgentID)
	}
}

// RecordAgentMessageReceived records an agent message received if global metrics is set.
func RecordAgentMessageReceived(ctx context.Context, messageType, toAgentID string) {
	if globalMetrics != nil {
		globalMetrics.RecordAgentMessageReceived(ctx, messageType, toAgentID)
	}
}

// RecordAgentRequestSent records an agent request sent if global metrics is set.
func RecordAgentRequestSent(ctx context.Context, action, fromAgentID string) {
	if globalMetrics != nil {
		globalMetrics.RecordAgentRequestSent(ctx, action, fromAgentID)
	}
}

// RecordAgentRequestReceived records an agent request received if global metrics is set.
func RecordAgentRequestReceived(ctx context.Context, action, toAgentID string) {
	if globalMetrics != nil {
		globalMetrics.RecordAgentRequestReceived(ctx, action, toAgentID)
	}
}

// RecordAgentResponseSent records an agent response sent if global metrics is set.
func RecordAgentResponseSent(ctx context.Context, status, toAgentID string) {
	if globalMetrics != nil {
		globalMetrics.RecordAgentResponseSent(ctx, status, toAgentID)
	}
}

// RecordAgentResponseReceived records an agent response received if global metrics is set.
func RecordAgentResponseReceived(ctx context.Context, status, fromAgentID string) {
	if globalMetrics != nil {
		globalMetrics.RecordAgentResponseReceived(ctx, status, fromAgentID)
	}
}

// RecordA2ACommunicationError records an A2A communication error if global metrics is set.
func RecordA2ACommunicationError(ctx context.Context, operation string, err error) {
	if globalMetrics != nil {
		globalMetrics.RecordA2ACommunicationError(ctx, operation, err)
	}
}

// Global helper functions for Event metrics

// RecordEventPublished records an event published if global metrics is set.
func RecordEventPublished(ctx context.Context, eventType, source string) {
	if globalMetrics != nil {
		globalMetrics.RecordEventPublished(ctx, eventType, source)
	}
}

// RecordEventConsumed records an event consumed if global metrics is set.
func RecordEventConsumed(ctx context.Context, eventType, consumer string) {
	if globalMetrics != nil {
		globalMetrics.RecordEventConsumed(ctx, eventType, consumer)
	}
}

// RecordAgentLifecycleEvent records an agent lifecycle event if global metrics is set.
func RecordAgentLifecycleEvent(ctx context.Context, agentID, eventType string) {
	if globalMetrics != nil {
		globalMetrics.RecordAgentLifecycleEvent(ctx, agentID, eventType)
	}
}

// RecordTaskEvent records a task event if global metrics is set.
func RecordTaskEvent(ctx context.Context, taskID, eventType string) {
	if globalMetrics != nil {
		globalMetrics.RecordTaskEvent(ctx, taskID, eventType)
	}
}

// RecordWorkflowEvent records a workflow event if global metrics is set.
func RecordWorkflowEvent(ctx context.Context, workflowID, eventType string) {
	if globalMetrics != nil {
		globalMetrics.RecordWorkflowEvent(ctx, workflowID, eventType)
	}
}

// RecordEventProcessingError records an event processing error if global metrics is set.
func RecordEventProcessingError(ctx context.Context, eventType string, err error) {
	if globalMetrics != nil {
		globalMetrics.RecordEventProcessingError(ctx, eventType, err)
	}
}

// Global helper functions for Validation metrics

// RecordSchemaValidation records a schema validation if global metrics is set.
func RecordSchemaValidation(ctx context.Context, validationType string) {
	if globalMetrics != nil {
		globalMetrics.RecordSchemaValidation(ctx, validationType)
	}
}

// RecordSchemaValidationError records a schema validation error if global metrics is set.
func RecordSchemaValidationError(ctx context.Context, validationType string, err error) {
	if globalMetrics != nil {
		globalMetrics.RecordSchemaValidationError(ctx, validationType, err)
	}
}

// RecordMessageValidation records a message validation if global metrics is set.
func RecordMessageValidation(ctx context.Context, msgType MessageType) {
	if globalMetrics != nil {
		globalMetrics.RecordMessageValidation(ctx, msgType)
	}
}

// RecordMessageValidationError records a message validation error if global metrics is set.
func RecordMessageValidationError(ctx context.Context, msgType MessageType, err error) {
	if globalMetrics != nil {
		globalMetrics.RecordMessageValidationError(ctx, msgType, err)
	}
}

// RecordDocumentValidation records a document validation if global metrics is set.
func RecordDocumentValidation(ctx context.Context) {
	if globalMetrics != nil {
		globalMetrics.RecordDocumentValidation(ctx)
	}
}

// RecordDocumentValidationError records a document validation error if global metrics is set.
func RecordDocumentValidationError(ctx context.Context, err error) {
	if globalMetrics != nil {
		globalMetrics.RecordDocumentValidationError(ctx, err)
	}
}

// Global helper functions for Factory metrics

// RecordFactoryCreation records a factory creation if global metrics is set.
func RecordFactoryCreation(ctx context.Context, objectType string) {
	if globalMetrics != nil {
		globalMetrics.RecordFactoryCreation(ctx, objectType)
	}
}

// RecordFactoryError records a factory error if global metrics is set.
func RecordFactoryError(ctx context.Context, objectType string, err error) {
	if globalMetrics != nil {
		globalMetrics.RecordFactoryError(ctx, objectType, err)
	}
}
