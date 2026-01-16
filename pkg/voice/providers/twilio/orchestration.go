package twilio

import (
	"context"
	"fmt"
	"sync"

	"github.com/lookatitude/beluga-ai/pkg/core"
	orchestrationiface "github.com/lookatitude/beluga-ai/pkg/orchestration/iface"
	vbiface "github.com/lookatitude/beluga-ai/pkg/voice/backend/iface"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// EventHandler handles webhook events for event-driven workflows.
type EventHandler func(ctx context.Context, event *WebhookEvent) error

// OrchestrationManager manages workflow orchestration for Twilio voice events.
// It provides event-driven workflows using internal event handlers and scheduler for delayed operations.
type OrchestrationManager struct {
	orchestrator  orchestrationiface.Orchestrator
	workflows     map[string]orchestrationiface.Graph
	eventHandlers map[string][]EventHandler // Map event type to handlers
	mu            sync.RWMutex
	backend       *TwilioBackend
	ctx           context.Context
	cancel        context.CancelFunc
}

// NewOrchestrationManager creates a new orchestration manager.
func NewOrchestrationManager(backend *TwilioBackend, orchestrator orchestrationiface.Orchestrator) (*OrchestrationManager, error) {
	ctx, cancel := context.WithCancel(context.Background())

	manager := &OrchestrationManager{
		orchestrator:  orchestrator,
		workflows:     make(map[string]orchestrationiface.Graph),
		eventHandlers: make(map[string][]EventHandler),
		backend:       backend,
		ctx:           ctx,
		cancel:        cancel,
	}

	// Setup event-driven flows
	if err := manager.setupEventDrivenFlows(ctx); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to setup event-driven flows: %w", err)
	}

	// Create default call flow workflow: Inbound → Agent → Stream
	if err := manager.createDefaultCallFlowWorkflow(); err != nil {
		cancel()
		return nil, err
	}

	return manager, nil
}

// Stop stops the orchestration manager gracefully.
func (m *OrchestrationManager) Stop(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.cancel != nil {
		m.cancel()
	}

	return nil
}

// setupEventDrivenFlows sets up event-driven workflows using event handlers.
func (m *OrchestrationManager) setupEventDrivenFlows(ctx context.Context) error {
	// Register handlers for call events
	m.RegisterEventHandler("call.answered", m.TriggerCallFlowWorkflow)
	m.RegisterEventHandler("call.ended", m.TriggerCallEndedWorkflow)
	m.RegisterEventHandler("call.failed", m.TriggerCallFailedWorkflow)

	return nil
}

// RegisterEventHandler registers a handler for a specific event type.
func (m *OrchestrationManager) RegisterEventHandler(eventType string, handler EventHandler) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.eventHandlers == nil {
		m.eventHandlers = make(map[string][]EventHandler)
	}
	m.eventHandlers[eventType] = append(m.eventHandlers[eventType], handler)
}

// PublishEvent publishes a webhook event to registered event handlers.
func (m *OrchestrationManager) PublishEvent(ctx context.Context, event *WebhookEvent) error {
	m.mu.RLock()
	handlers, exists := m.eventHandlers[event.EventType]
	m.mu.RUnlock()

	if !exists {
		// No handlers for this event type - not an error
		return nil
	}

	// Execute all handlers for this event type
	for _, handler := range handlers {
		if err := handler(ctx, event); err != nil {
			return fmt.Errorf("error in event handler for %s: %w", event.EventType, err)
		}
	}

	return nil
}

// TriggerCallEndedWorkflow triggers the call ended workflow.
func (m *OrchestrationManager) TriggerCallEndedWorkflow(ctx context.Context, event *WebhookEvent) error {
	ctx, span := m.startSpan(ctx, "OrchestrationManager.TriggerCallEndedWorkflow")
	defer span.End()

	span.SetAttributes(attribute.String("event_type", event.EventType))

	// Extract CallSID from event data
	callSID := extractCallSID(event)

	// Cleanup session when call ends
	if callSID != "" {
		session, err := m.backend.GetSession(ctx, callSID)
		if err == nil && session != nil {
			if err := session.Stop(ctx); err != nil {
				span.RecordError(err)
			}
		}
	}

	span.SetStatus(codes.Ok, "call ended workflow completed")
	return nil
}

// TriggerCallFailedWorkflow triggers the call failed workflow.
func (m *OrchestrationManager) TriggerCallFailedWorkflow(ctx context.Context, event *WebhookEvent) error {
	ctx, span := m.startSpan(ctx, "OrchestrationManager.TriggerCallFailedWorkflow")
	defer span.End()

	span.SetAttributes(attribute.String("event_type", event.EventType))

	// Extract CallSID from event data
	callSID := extractCallSID(event)

	// Handle call failure - cleanup and log
	if callSID != "" {
		session, err := m.backend.GetSession(ctx, callSID)
		if err == nil && session != nil {
			if err := session.Stop(ctx); err != nil {
				span.RecordError(err)
			}
		}
	}

	span.SetStatus(codes.Ok, "call failed workflow completed")
	return nil
}

// extractCallSID extracts CallSID from webhook event data.
func extractCallSID(event *WebhookEvent) string {
	if event.EventData == nil {
		return ""
	}

	// Try common CallSID field names
	if callSID, ok := event.EventData["CallSid"].(string); ok {
		return callSID
	}
	if callSID, ok := event.EventData["CallSID"].(string); ok {
		return callSID
	}
	if callSID, ok := event.EventData["call_sid"].(string); ok {
		return callSID
	}
	if resourceSID := event.ResourceSID; resourceSID != "" {
		return resourceSID
	}

	return ""
}

// createDefaultCallFlowWorkflow creates the default DAG workflow for call flows.
func (m *OrchestrationManager) createDefaultCallFlowWorkflow() error {
	// Define workflow nodes
	handleInboundNode := &runnableFunc{
		name: "handle_inbound",
		fn: func(ctx context.Context, input any, options ...core.Option) (any, error) {
			// Handle inbound call
			webhookData, ok := input.(map[string]string)
			if !ok {
				return nil, fmt.Errorf("invalid input type")
			}

			session, err := m.backend.HandleInboundCall(ctx, webhookData)
			if err != nil {
				return nil, err
			}

			return session, nil
		},
	}

	setupAgentNode := &runnableFunc{
		name: "setup_agent",
		fn: func(ctx context.Context, input any, options ...core.Option) (any, error) {
			// Setup agent for session
			session, ok := input.(vbiface.VoiceSession)
			if !ok {
				return nil, fmt.Errorf("invalid input type")
			}

			// Agent setup logic would go here
			return session, nil
		},
	}

	createStreamNode := &runnableFunc{
		name: "create_stream",
		fn: func(ctx context.Context, input any, options ...core.Option) (any, error) {
			// Create audio stream for session
			session, ok := input.(vbiface.VoiceSession)
			if !ok {
				return nil, fmt.Errorf("invalid input type")
			}

			stream, err := m.backend.StreamAudio(ctx, session.GetID())
			if err != nil {
				return nil, err
			}

			return stream, nil
		},
	}

	// Create DAG workflow: Inbound → Agent → Stream
	workflow, err := m.orchestrator.CreateGraph(
		func(config *orchestrationiface.GraphConfig) error {
			config.Name = "twilio_call_flow"
			config.Nodes = map[string]core.Runnable{
				"inbound": handleInboundNode,
				"agent":   setupAgentNode,
				"stream":  createStreamNode,
			}
			config.Edges = []orchestrationiface.GraphEdge{
				{Source: "inbound", Target: "agent"},
				{Source: "agent", Target: "stream"},
			}
			config.EntryPoints = []string{"inbound"}
			config.ExitPoints = []string{"stream"}
			return nil
		},
	)
	if err != nil {
		return err
	}

	m.mu.Lock()
	m.workflows["call_flow"] = workflow
	m.mu.Unlock()

	return nil
}

// TriggerCallFlowWorkflow triggers the call flow workflow from a call.answered event.
func (m *OrchestrationManager) TriggerCallFlowWorkflow(ctx context.Context, event *WebhookEvent) error {
	ctx, span := m.startSpan(ctx, "OrchestrationManager.TriggerCallFlowWorkflow")
	defer span.End()

	span.SetAttributes(attribute.String("event_type", event.EventType))

	m.mu.RLock()
	workflow, exists := m.workflows["call_flow"]
	m.mu.RUnlock()

	if !exists {
		err := fmt.Errorf("call flow workflow not found")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	// Convert event to workflow input
	input := convertEventDataToMap(event.EventData)

	// Invoke workflow
	_, err := workflow.Invoke(ctx, input)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	span.SetStatus(codes.Ok, "workflow triggered")
	return nil
}

// startSpan starts an OTEL span for tracing.
func (m *OrchestrationManager) startSpan(ctx context.Context, operation string) (context.Context, trace.Span) {
	// Use backend's tracer if available
	if m.backend.metrics != nil && m.backend.metrics.Tracer() != nil {
		return m.backend.metrics.Tracer().Start(ctx, operation)
	}
	return ctx, trace.SpanFromContext(ctx)
}

// runnableFunc implements core.Runnable for function-based runnables.
type runnableFunc struct {
	name string
	fn   func(ctx context.Context, input any, options ...core.Option) (any, error)
}

func (r *runnableFunc) Invoke(ctx context.Context, input any, options ...core.Option) (any, error) {
	return r.fn(ctx, input, options...)
}

func (r *runnableFunc) Batch(ctx context.Context, inputs []any, options ...core.Option) ([]any, error) {
	results := make([]any, len(inputs))
	for i, input := range inputs {
		result, err := r.fn(ctx, input, options...)
		if err != nil {
			return nil, err
		}
		results[i] = result
	}
	return results, nil
}

func (r *runnableFunc) Stream(ctx context.Context, input any, options ...core.Option) (<-chan any, error) {
	ch := make(chan any, 1)
	go func() {
		defer close(ch)
		result, err := r.fn(ctx, input, options...)
		if err != nil {
			ch <- err
			return
		}
		ch <- result
	}()
	return ch, nil
}
