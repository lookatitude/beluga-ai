// Package base provides the base implementation for agents.
// It implements common agent functionality that can be embedded by specific agent types.
package base

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/agents/iface"
	"github.com/lookatitude/beluga-ai/pkg/agents/tools"
	"github.com/lookatitude/beluga-ai/pkg/core"
	llmsiface "github.com/lookatitude/beluga-ai/pkg/llms/iface"
	"github.com/lookatitude/beluga-ai/pkg/schema"
)

// options holds the configuration options for an agent.
type options struct {
	maxRetries    int
	retryDelay    time.Duration
	enableMetrics bool
	enableTracing bool
	eventHandlers map[string][]func(interface{}) error
}

// BaseAgent provides common functionality for all agents.
// It implements the CompositeAgent interface through composition and embedding.
type BaseAgent struct {
	// Core agent properties
	name   string
	config schema.AgentConfig
	llm    llmsiface.LLM
	tools  []tools.Tool
	memory interface{} // Memory interface (to be defined)

	// Lifecycle management
	state          iface.AgentState
	createdAt      time.Time
	lastActiveTime time.Time
	mutex          sync.RWMutex
	ctx            context.Context
	cancelFunc     context.CancelFunc

	// Operational settings
	maxRetries int
	retryDelay time.Duration
	errorCount int

	// Event handling
	eventHandlers map[string][]iface.EventHandler

	// Observability
	metrics iface.MetricsRecorder
}

// NewBaseAgent creates a new BaseAgent with the provided configuration.
func NewBaseAgent(name string, llm llmsiface.LLM, agentTools []tools.Tool, opts ...iface.Option) (*BaseAgent, error) {
	if llm == nil {
		return nil, fmt.Errorf("LLM cannot be nil")
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Apply default options
	options := &iface.Options{
		MaxRetries:    3,
		RetryDelay:    2 * time.Second,
		EnableMetrics: true,
		EnableTracing: true,
		EventHandlers: make(map[string][]iface.EventHandler),
	}

	// Apply user options
	for _, opt := range opts {
		opt(options)
	}

	agent := &BaseAgent{
		name:          name,
		llm:           llm,
		tools:         agentTools,
		config:        schema.AgentConfig{Name: name}, // Initialize config with agent name
		state:         iface.StateInitializing,
		createdAt:     time.Now(),
		ctx:           ctx,
		cancelFunc:    cancel,
		maxRetries:    options.MaxRetries,
		retryDelay:    options.RetryDelay,
		eventHandlers: make(map[string][]iface.EventHandler),
	}

	// Initialize metrics (use provided metrics or nil if not enabled)
	// TODO: Pass metrics as dependency injection parameter
	agent.metrics = options.Metrics
	// Register event handlers
	for eventType, handlers := range options.EventHandlers {
		for _, handler := range handlers {
			agent.RegisterEventHandler(eventType, handler)
		}
	}

	return agent, nil
}

// Core Agent interface implementation

// InputVariables returns the expected input variables for the agent.
// This is a placeholder and should be overridden by specific agent implementations.
func (a *BaseAgent) InputVariables() []string {
	return []string{"input"}
}

// OutputVariables returns the expected output variables from the agent.
// This is a placeholder and should be overridden by specific agent implementations.
func (a *BaseAgent) OutputVariables() []string {
	return []string{"output"}
}

// GetTools returns the list of tools available to the agent.
func (a *BaseAgent) GetTools() []tools.Tool {
	a.mutex.RLock()
	defer a.mutex.RUnlock()
	return a.tools
}

// GetConfig returns the agent's configuration.
func (a *BaseAgent) GetConfig() schema.AgentConfig {
	a.mutex.RLock()
	defer a.mutex.RUnlock()
	return a.config
}

// GetLLM returns the LLM instance used by the agent.
func (a *BaseAgent) GetLLM() llmsiface.LLM {
	return a.llm
}

// Plan is a placeholder implementation that should be overridden by specific agents.
func (a *BaseAgent) Plan(ctx context.Context, intermediateSteps []iface.IntermediateStep, inputs map[string]any) (iface.AgentAction, iface.AgentFinish, error) {
	// Start tracing span if metrics are available
	var planCtx context.Context = ctx
	var span iface.SpanEnder
	if a.metrics != nil {
		planCtx, span = a.metrics.StartAgentSpan(ctx, a.name, "plan")
		defer func() {
			if span != nil {
				span.End()
			}
		}()
	}

	start := time.Now()

	// Placeholder implementation - should be overridden by specific agents
	err := fmt.Errorf("Plan method not implemented in BaseAgent; must be overridden by specific agent type")

	if a.metrics != nil {
		a.metrics.RecordPlanningCall(planCtx, a.name, time.Since(start), false)
	}
	return iface.AgentAction{}, iface.AgentFinish{}, err
}

// Runnable interface implementation

// Invoke executes the agent with a single input and returns the result.
// This is the primary method for synchronous execution.
func (a *BaseAgent) Invoke(ctx context.Context, input any, options ...core.Option) (any, error) {
	// Start tracing span if metrics are available
	var invokeCtx context.Context = ctx
	var span iface.SpanEnder
	if a.metrics != nil {
		invokeCtx, span = a.metrics.StartAgentSpan(ctx, a.name, "invoke")
		defer func() {
			if span != nil {
				span.End()
			}
		}()
	}

	start := time.Now()

	// Convert input to the expected format
	inputs, ok := input.(map[string]any)
	if !ok {
		if a.metrics != nil {
			a.metrics.RecordAgentExecution(invokeCtx, a.name, "base", time.Since(start), false)
		}
		return nil, fmt.Errorf("input must be a map[string]any, got %T", input)
	}

	// Apply options to create execution context
	config := make(map[string]any)
	for _, opt := range options {
		opt.Apply(&config)
	}

	// Set up execution context with timeout if specified
	execCtx := invokeCtx
	if timeout, ok := config["timeout"].(time.Duration); ok {
		var cancel context.CancelFunc
		execCtx, cancel = context.WithTimeout(invokeCtx, timeout)
		defer cancel()
	}

	// Execute the agent
	result, err := a.executeWithInput(execCtx, inputs)
	duration := time.Since(start)

	if err != nil {
		if a.metrics != nil {
			a.metrics.RecordAgentExecution(invokeCtx, a.name, "base", duration, false)
		}
		return nil, fmt.Errorf("agent execution failed: %w", err)
	}

	if a.metrics != nil {
		a.metrics.RecordAgentExecution(invokeCtx, a.name, "base", duration, true)
	}
	return result, nil
}

// Batch executes the agent with multiple inputs and returns corresponding outputs.
// This implementation processes inputs sequentially for simplicity.
func (a *BaseAgent) Batch(ctx context.Context, inputs []any, options ...core.Option) ([]any, error) {
	results := make([]any, len(inputs))

	for i, input := range inputs {
		result, err := a.Invoke(ctx, input, options...)
		if err != nil {
			return nil, fmt.Errorf("batch execution failed at input %d: %w", i, err)
		}
		results[i] = result
	}

	return results, nil
}

// Stream executes the agent and returns a channel for streaming output.
// This is a basic implementation that sends the final result when ready.
func (a *BaseAgent) Stream(ctx context.Context, input any, options ...core.Option) (<-chan any, error) {
	resultChan := make(chan any, 1)

	go func() {
		defer close(resultChan)

		result, err := a.Invoke(ctx, input, options...)
		if err != nil {
			resultChan <- err
			return
		}

		resultChan <- result
	}()

	return resultChan, nil
}

// executeWithInput performs the actual agent execution with the provided inputs.
// This method should be overridden by specific agent implementations.
func (a *BaseAgent) executeWithInput(ctx context.Context, inputs map[string]any) (any, error) {
	// Default implementation - should be overridden by specific agents
	return nil, fmt.Errorf("executeWithInput not implemented in BaseAgent; must be overridden by specific agent type")
}

// Lifecycle management

// Initialize sets up the agent with necessary configurations.
func (a *BaseAgent) Initialize(config map[string]interface{}) error {
	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}

	a.mutex.Lock()
	// Update configuration (store in a separate field for now)
	// TODO: Update schema.AgentConfig to support settings
	a.config = schema.AgentConfig{
		Name:     a.name,
		Settings: config,
	}

	// Handle specific configuration options
	if maxRetries, ok := config["max_retries"].(int); ok {
		a.maxRetries = maxRetries
	}
	if retryDelay, ok := config["retry_delay"].(time.Duration); ok {
		a.retryDelay = retryDelay
	}

	// Update state while holding lock
	a.state = iface.StateReady
	a.lastActiveTime = time.Now()
	a.mutex.Unlock()

	// Emit events after releasing lock to avoid deadlock
	a.emitEvent("state_change", iface.StateReady)
	a.emitEvent("initialized", map[string]interface{}{
		"config": config,
		"time":   time.Now(),
	})

	return nil
}

// Execute performs the main task of the agent with retry logic.
func (a *BaseAgent) Execute() error {
	a.mutex.Lock()
	a.state = iface.StateRunning
	a.lastActiveTime = time.Now()
	a.mutex.Unlock()

	// Emit state change event after releasing lock
	a.emitEvent("state_change", iface.StateRunning)

	start := time.Now()

	// Emit execution start event
	a.emitEvent("execution_started", map[string]interface{}{
		"time": time.Now(),
	})

	// Implement retry logic
	var err error
	for attempt := 0; attempt <= a.maxRetries; attempt++ {
		// Check context cancellation before each attempt
		select {
		case <-a.ctx.Done():
			a.mutex.Lock()
			a.state = iface.StateError
			a.lastActiveTime = time.Now()
			a.mutex.Unlock()

			// Emit events after releasing lock
			a.emitEvent("state_change", iface.StateError)
			a.emitEvent("execution_cancelled", map[string]interface{}{
				"attempt":    attempt,
				"total_time": time.Since(start),
			})
			if a.metrics != nil {
				a.metrics.RecordAgentExecution(a.ctx, a.name, "base", time.Since(start), false)
			}
			return fmt.Errorf("agent %s execution cancelled: %w", a.name, a.ctx.Err())
		default:
		}

		if attempt > 0 {
			a.emitEvent("retry", map[string]interface{}{
				"attempt": attempt,
				"delay":   a.retryDelay,
			})
			// Use context-aware sleep to allow cancellation during retry delay
			select {
			case <-time.After(a.retryDelay):
				// Continue with retry
			case <-a.ctx.Done():
				a.mutex.Lock()
				a.state = iface.StateError
				a.lastActiveTime = time.Now()
				a.mutex.Unlock()

				// Emit events after releasing lock
				a.emitEvent("state_change", iface.StateError)
				a.emitEvent("execution_cancelled", map[string]interface{}{
					"attempt":    attempt,
					"total_time": time.Since(start),
				})
				if a.metrics != nil {
					a.metrics.RecordAgentExecution(a.ctx, a.name, "base", time.Since(start), false)
				}
				return fmt.Errorf("agent %s execution cancelled during retry delay: %w", a.name, a.ctx.Err())
			}
		}

		err = a.doExecute()
		if err == nil {
			break
		}

		a.mutex.Lock()
		a.errorCount++
		a.mutex.Unlock()

		a.emitEvent("execution_error", map[string]interface{}{
			"attempt": attempt,
			"error":   err.Error(),
		})
	}

	a.mutex.Lock()
	if err != nil {
		a.state = iface.StateError
		a.lastActiveTime = time.Now()
		a.mutex.Unlock()

		// Emit events after releasing lock
		a.emitEvent("state_change", iface.StateError)
		a.emitEvent("execution_failed", map[string]interface{}{
			"error":      err.Error(),
			"attempts":   a.maxRetries + 1,
			"total_time": time.Since(start),
		})
		if a.metrics != nil {
			a.metrics.RecordAgentExecution(a.ctx, a.name, "base", time.Since(start), false)
		}
		return fmt.Errorf("agent %s execution failed after %d attempts: %w", a.name, a.maxRetries+1, err)
	}

	a.state = iface.StateReady
	a.lastActiveTime = time.Now()
	a.mutex.Unlock()

	// Emit events after releasing lock
	a.emitEvent("state_change", iface.StateReady)
	a.emitEvent("execution_completed", map[string]interface{}{
		"total_time": time.Since(start),
	})
	if a.metrics != nil {
		a.metrics.RecordAgentExecution(a.ctx, a.name, "base", time.Since(start), true)
	}

	return nil
}

// doExecute is the internal execution method that should be overridden by subclasses.
func (a *BaseAgent) doExecute() error {
	return fmt.Errorf("doExecute not implemented in BaseAgent; must be overridden by specific agent type")
}

// Shutdown gracefully stops the agent and cleans up resources.
func (a *BaseAgent) Shutdown() error {
	a.mutex.Lock()
	if a.state == iface.StateShutdown {
		a.mutex.Unlock()
		return nil
	}

	// Cancel context first to signal Execute() to stop
	a.cancelFunc()
	a.state = iface.StateShutdown
	a.lastActiveTime = time.Now()
	a.mutex.Unlock()

	// Give Execute() a brief moment to respond to context cancellation
	// This prevents race conditions where Execute() might still be emitting events
	time.Sleep(10 * time.Millisecond)

	// Emit events after releasing lock and allowing Execute() to finish
	a.emitEvent("shutdown_started", map[string]interface{}{
		"time": time.Now(),
	})
	a.emitEvent("state_change", iface.StateShutdown)
	a.emitEvent("shutdown_completed", map[string]interface{}{
		"time": time.Now(),
	})

	return nil
}

// GetState returns the current state of the agent.
func (a *BaseAgent) GetState() iface.AgentState {
	a.mutex.RLock()
	defer a.mutex.RUnlock()
	return a.state
}

// CheckHealth returns the health status of the agent.
func (a *BaseAgent) CheckHealth() map[string]interface{} {
	a.mutex.RLock()
	defer a.mutex.RUnlock()

	return map[string]interface{}{
		"name":             a.name,
		"state":            a.state,
		"up_time":          time.Since(a.createdAt).String(),
		"last_active_time": a.lastActiveTime,
		"error_count":      a.errorCount,
		"tools_count":      len(a.tools),
	}
}

// Event handling

// RegisterEventHandler registers a handler function for a specific event type.
func (a *BaseAgent) RegisterEventHandler(eventType string, handler iface.EventHandler) {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	if _, exists := a.eventHandlers[eventType]; !exists {
		a.eventHandlers[eventType] = make([]iface.EventHandler, 0)
	}

	a.eventHandlers[eventType] = append(a.eventHandlers[eventType], handler)
}

// EmitEvent triggers all registered handlers for the given event type.
func (a *BaseAgent) EmitEvent(eventType string, payload interface{}) {
	a.emitEvent(eventType, payload)
}

// GetMetrics returns the metrics recorder for the agent.
func (a *BaseAgent) GetMetrics() iface.MetricsRecorder {
	return a.metrics
}

// Private methods

// setState updates the agent's state and emits a state change event.
func (a *BaseAgent) setState(state iface.AgentState) {
	a.state = state
	a.lastActiveTime = time.Now()
	a.emitEvent("state_change", state)
}

// emitEvent calls all registered handlers for the given event type.
func (a *BaseAgent) emitEvent(eventType string, payload interface{}) {
	a.mutex.RLock()
	handlers := make([]iface.EventHandler, len(a.eventHandlers[eventType]))
	copy(handlers, a.eventHandlers[eventType])
	a.mutex.RUnlock()

	for _, handler := range handlers {
		if err := handler(payload); err != nil {
			// Log error but don't fail the operation
			// In a real implementation, you'd use structured logging here
			fmt.Printf("Event handler error for %s: %v\n", eventType, err)
		}
	}
}
