package twilio

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/core"
	"github.com/lookatitude/beluga-ai/pkg/messaging/iface"
	orchestrationiface "github.com/lookatitude/beluga-ai/pkg/orchestration/iface"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// OrchestrationManager manages workflow orchestration for Twilio messaging events.
type OrchestrationManager struct {
	orchestrator interface{}            // orchestrationiface.Orchestrator
	workflows    map[string]interface{} // orchestrationiface.Graph
	mu           sync.RWMutex
	provider     *TwilioProvider
}

// NewOrchestrationManager creates a new orchestration manager.
func NewOrchestrationManager(provider *TwilioProvider, orchestrator interface{}) (*OrchestrationManager, error) {
	manager := &OrchestrationManager{
		orchestrator: orchestrator,
		workflows:    make(map[string]interface{}),
		provider:     provider,
	}

	// Create default message processing workflow
	if err := manager.createDefaultMessageFlowWorkflow(); err != nil {
		return nil, err
	}

	return manager, nil
}

// createDefaultMessageFlowWorkflow creates the default DAG workflow for message processing.
func (m *OrchestrationManager) createDefaultMessageFlowWorkflow() error {
	// Define workflow nodes
	processMessageNode := &runnableFunc{
		name: "process_message",
		fn: func(ctx context.Context, input any, options ...core.Option) (any, error) {
			// Process incoming message
			message, ok := input.(*iface.Message)
			if !ok {
				return nil, fmt.Errorf("invalid input type")
			}

			// Get or create session (thread-safe)
			// First, check with read lock
			m.provider.mu.RLock()
			session, exists := m.provider.sessions[message.ConversationSID]
			m.provider.mu.RUnlock()

			if !exists {
				// Create new session - use write lock with double-check pattern
				// to avoid race condition where multiple goroutines create sessions
				m.provider.mu.Lock()
				// Double-check: another goroutine might have created it while we waited for lock
				session, exists = m.provider.sessions[message.ConversationSID]
				if !exists {
					var err error
					session, err = NewMessagingSession(message.ConversationSID, m.provider.config, m.provider)
					if err != nil {
						m.provider.mu.Unlock()
						return nil, err
					}

					m.provider.sessions[message.ConversationSID] = session
					m.provider.mu.Unlock()

					if err := session.Start(ctx); err != nil {
						return nil, err
					}
				} else {
					// Session was created by another goroutine, just use it
					m.provider.mu.Unlock()
				}
			}

			// Process message through session
			if err := session.ProcessMessage(ctx, message); err != nil {
				return nil, err
			}

			return message, nil
		},
	}

	generateResponseNode := &runnableFunc{
		name: "generate_response",
		fn: func(ctx context.Context, input any, options ...core.Option) (any, error) {
			// Generate agent response
			message, ok := input.(*iface.Message)
			if !ok {
				return nil, fmt.Errorf("invalid input type")
			}

			// Response generation is handled in session.ProcessMessage
			return message, nil
		},
	}

	sendResponseNode := &runnableFunc{
		name: "send_response",
		fn: func(ctx context.Context, input any, options ...core.Option) (any, error) {
			// Send response message
			message, ok := input.(*iface.Message)
			if !ok {
				return nil, fmt.Errorf("invalid input type")
			}

			// Response sending is handled in session.ProcessMessage
			return message, nil
		},
	}

	// Create DAG workflow: Process → Generate → Send
	// Note: Full implementation would cast orchestrator to orchestrationiface.Orchestrator
	orchestrator, _ := m.orchestrator.(orchestrationiface.Orchestrator)
	if orchestrator == nil {
		return fmt.Errorf("orchestrator not available")
	}
	workflow, err := orchestrator.CreateGraph(
		func(config *orchestrationiface.GraphConfig) error {
			config.Name = "twilio_message_flow"
			config.Nodes = map[string]core.Runnable{
				"process":  processMessageNode,
				"generate": generateResponseNode,
				"send":     sendResponseNode,
			}
			config.Edges = []orchestrationiface.GraphEdge{
				{Source: "process", Target: "generate"},
				{Source: "generate", Target: "send"},
			}
			config.EntryPoints = []string{"process"}
			config.ExitPoints = []string{"send"}
			return nil
		},
	)
	if err != nil {
		return err
	}

	m.mu.Lock()
	m.workflows["message_flow"] = workflow
	m.mu.Unlock()

	return nil
}

// TriggerMessageFlowWorkflow triggers the message flow workflow from a message.added event.
func (m *OrchestrationManager) TriggerMessageFlowWorkflow(ctx context.Context, event *iface.WebhookEvent) error {
	ctx, span := m.startSpan(ctx, "OrchestrationManager.TriggerMessageFlowWorkflow")
	defer span.End()

	span.SetAttributes(attribute.String("event_type", event.EventType))

	m.mu.RLock()
	workflowInterface, exists := m.workflows["message_flow"]
	m.mu.RUnlock()

	if !exists {
		err := fmt.Errorf("message flow workflow not found")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	workflow, ok := workflowInterface.(orchestrationiface.Graph)
	if !ok {
		err := fmt.Errorf("workflow type assertion failed")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	// Parse message from event
	message := &iface.Message{
		MessageSID:      getStringFromEvent(event.EventData, "MessageSid"),
		ConversationSID: getStringFromEvent(event.EventData, "ConversationSid"),
		Body:            getStringFromEvent(event.EventData, "Body"),
		Channel:         iface.Channel(getStringFromEvent(event.EventData, "Channel")),
		From:            getStringFromEvent(event.EventData, "Author"),
		DateCreated:     time.Now(),
	}

	// Invoke workflow
	_, err := workflow.Invoke(ctx, message)
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
	// Use provider's tracer if available
	if m.provider.metrics != nil && m.provider.metrics.Tracer() != nil {
		return m.provider.metrics.Tracer().Start(ctx, operation)
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
