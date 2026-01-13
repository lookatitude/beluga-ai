// Package agents provides AI agent implementations following the Beluga AI Framework design patterns.
//
// This package implements autonomous agents that can reason, plan, and execute actions using tools.
// It follows SOLID principles with dependency inversion, interface segregation, and composition over inheritance.
//
// Key Features:
//   - Multiple agent types (ReAct, custom implementations)
//   - Tool integration with registry system
//   - Observability with OpenTelemetry tracing and metrics
//   - Configurable execution with retry logic and timeouts
//   - Event-driven architecture for extensibility
//
// Basic Usage:
//
//	// Create a simple agent
//	llm := // ... initialize LLM
//	tools := // ... get tools
//	agent, err := agents.NewBaseAgent("my-agent", llm, tools)
//
//	// Execute with input
//	result, err := agent.Execute()
//
// Advanced Usage:
//
//	// Create agent with custom configuration
//	agent, err := agents.NewBaseAgent("my-agent", llm, tools,
//		agents.WithMaxRetries(5),
//		agents.WithTimeout(60*time.Second),
//		agents.WithEventHandler("execution_completed", func(payload interface{}) error {
//			log.Printf("Agent completed: %v", payload)
//			return nil
//		}),
//	)
//
//	// Use with executor
//	:= agents.NewAgentExecutor()
//	result, err := executor.ExecutePlan(ctx, agent, plan)
package agents

import (
	"context"
	"log/slog"

	"go.opentelemetry.io/otel/trace"

	"github.com/lookatitude/beluga-ai/pkg/agents/iface"
	"github.com/lookatitude/beluga-ai/pkg/agents/internal/base"
	"github.com/lookatitude/beluga-ai/pkg/agents/internal/executor"
	"github.com/lookatitude/beluga-ai/pkg/agents/providers/react"
	"github.com/lookatitude/beluga-ai/pkg/agents/tools"
	llmsiface "github.com/lookatitude/beluga-ai/pkg/llms/iface"
)

// AgentFactory provides a centralized factory for creating different types of agents.
// It implements dependency injection and provides consistent configuration across agent types.
type AgentFactory struct {
	config  *Config
	metrics *Metrics
}

// NewAgentFactory creates a new agent factory with the specified configuration.
func NewAgentFactory(config *Config) *AgentFactory {
	return &AgentFactory{
		config:  config,
		metrics: DefaultMetrics(),
	}
}

// NewAgentFactoryWithMetrics creates a new agent factory with custom metrics.
func NewAgentFactoryWithMetrics(config *Config, metrics *Metrics) *AgentFactory {
	return &AgentFactory{
		config:  config,
		metrics: metrics,
	}
}

// NewBaseAgent creates a new base agent with the specified configuration.
// It provides a foundation for building custom agent implementations.
//
// Parameters:
//   - name: Unique identifier for the agent
//   - llm: Language model instance for reasoning and generation
//   - agentTools: List of tools available to the agent
//   - opts: Optional configuration functions
//
// Returns:
//   - Agent instance implementing the CompositeAgent interface
//   - Error if initialization fails
//
// Example:
//
//	llm := openai.NewGPT4()
//	tools := []tools.Tool{calculator, webSearch}
//	agent, err := agents.NewBaseAgent("assistant", llm, tools,
//		agents.WithMaxRetries(3),
//		agents.WithTimeout(30*time.Second),
//	)
func NewBaseAgent(name string, llm llmsiface.LLM, agentTools []tools.Tool, opts ...iface.Option) (iface.CompositeAgent, error) {
	return base.NewBaseAgent(name, llm, agentTools, opts...)
}

// CreateBaseAgent creates a base agent using the factory's configuration.
func (f *AgentFactory) CreateBaseAgent(ctx context.Context, name string, llm llmsiface.LLM, agentTools []tools.Tool, opts ...iface.Option) (iface.CompositeAgent, error) {
	// Merge factory config with provided options
	allOpts := []iface.Option{
		WithMaxRetries(f.config.DefaultMaxRetries),
		WithRetryDelay(f.config.DefaultRetryDelay),
		WithTimeout(f.config.DefaultTimeout),
		WithMetrics(f.config.EnableMetrics),
		WithTracing(f.config.EnableTracing),
	}

	// Add factory metrics if enabled
	if f.config.EnableMetrics && f.metrics != nil {
		allOpts = append(allOpts, func(o *iface.Options) {
			o.Metrics = f.metrics
		})
	}

	// Add user-provided options
	allOpts = append(allOpts, opts...)

	agent, err := base.NewBaseAgent(name, llm, agentTools, allOpts...)
	if err != nil {
		return nil, err
	}

	// Record agent creation metric
	f.metrics.RecordAgentCreation(ctx, "base")

	return agent, nil
}

// CreateReActAgent creates a ReAct agent using the factory's configuration.
func (f *AgentFactory) CreateReActAgent(ctx context.Context, name string, llm llmsiface.ChatModel, agentTools []tools.Tool, prompt any, opts ...iface.Option) (iface.CompositeAgent, error) {
	// Merge factory config with provided options
	allOpts := []iface.Option{
		WithMaxRetries(f.config.DefaultMaxRetries),
		WithRetryDelay(f.config.DefaultRetryDelay),
		WithTimeout(f.config.DefaultTimeout),
		WithMetrics(f.config.EnableMetrics),
		WithTracing(f.config.EnableTracing),
	}

	// Add user-provided options
	allOpts = append(allOpts, opts...)

	agent, err := react.NewReActAgent(name, llm, agentTools, prompt, allOpts...)
	if err != nil {
		return nil, err
	}

	// Record agent creation metric
	f.metrics.RecordAgentCreation(ctx, "react")

	return agent, nil
}

// NewReActAgent creates a new ReAct (Reasoning + Acting) agent.
// ReAct agents iteratively reason about problems and execute actions using tools.
//
// Parameters:
//   - name: Unique identifier for the agent
//   - llm: Chat model instance (required for ReAct pattern)
//   - agentTools: List of tools available to the agent
//   - prompt: Prompt template defining the agent's behavior
//   - opts: Optional configuration functions
//
// Returns:
//   - ReAct agent instance
//   - Error if initialization fails
//
// Example:
//
//	llm := openai.NewGPT4Chat()
//	tools := []tools.Tool{calculator, webSearch}
//	prompt := prompts.NewReActPrompt()
//	agent, err := agents.NewReActAgent("researcher", llm, tools, prompt)
func NewReActAgent(name string, llm llmsiface.ChatModel, agentTools []tools.Tool, prompt any, opts ...iface.Option) (iface.CompositeAgent, error) {
	return react.NewReActAgent(name, llm, agentTools, prompt, opts...)
}

// NewAgentExecutor creates a new agent executor.
// The executor handles the execution loop, tool calling, and result processing.
//
// Returns:
//   - New executor instance
//
// Example:
//
//	executor := agents.NewAgentExecutor(
//		agents.WithMaxIterations(10),
//		agents.WithReturnIntermediateSteps(true),
//	)
//	result, err := executor.ExecutePlan(ctx, agent, plan)
func NewAgentExecutor(opts ...executor.ExecutorOption) iface.Executor {
	return executor.NewAgentExecutor(opts...)
}

// Executor constructor functions

// NewExecutorWithMaxIterations creates a new executor with the specified maximum iterations.
func NewExecutorWithMaxIterations(max int) iface.Executor {
	return executor.NewAgentExecutor(executor.WithMaxIterations(max))
}

// NewExecutorWithReturnIntermediateSteps creates a new executor that returns intermediate steps.
func NewExecutorWithReturnIntermediateSteps(returnSteps bool) iface.Executor {
	return executor.NewAgentExecutor(executor.WithReturnIntermediateSteps(returnSteps))
}

// NewExecutorWithHandleParsingErrors creates a new executor with the specified error handling behavior.
func NewExecutorWithHandleParsingErrors(handle bool) iface.Executor {
	return executor.NewAgentExecutor(executor.WithHandleParsingErrors(handle))
}

// NewToolRegistry creates a new tool registry for managing agent tools.
// The registry provides centralized tool discovery and management.
//
// Returns:
//   - New tool registry instance
//
// Example:
//
//	registry := agents.NewToolRegistry()
//	registry.RegisterTool(calculator)
//	tool, err := registry.GetTool("calculator")
func NewToolRegistry() tools.Registry {
	return tools.NewInMemoryToolRegistry()
}

// NewDefaultConfig creates a new configuration instance with default values.
// This provides sensible defaults for most use cases while allowing customization.
//
// Returns:
//   - Configuration instance with defaults
//
// Example:
//
//	config := agents.NewDefaultConfig()
//	config.DefaultMaxRetries = 5
//	agent, err := agents.NewBaseAgent("my-agent", llm, tools,
//		agents.WithConfig(config),
//	)
func NewDefaultConfig() *Config {
	return DefaultConfig()
}

// ValidateConfig validates an agent configuration.
// This ensures the configuration is complete and contains valid values.
//
// Parameters:
//   - config: Configuration to validate
//
// Returns:
//   - Error if validation fails, nil otherwise
//
// Example:
//
//	config := agents.NewDefaultConfig()
//	if err := agents.ValidateConfig(config); err != nil {
//		log.Fatal("Invalid config:", err)
//	}
func ValidateConfig(config *Config) error {
	return config.Validate()
}

// HealthCheck performs a health check on an agent.
// This can be used for monitoring and ensuring agent availability.
//
// Parameters:
//   - agent: Agent to check
//
// Returns:
//   - Health status information
//
// Example:
//
//	status := agents.HealthCheck(agent)
//	if status["state"] == "error" {
//		log.Warn("Agent is in error state")
//	}
func HealthCheck(agent iface.CompositeAgent) map[string]any {
	return agent.CheckHealth()
}

// ListAgentStates returns all possible agent states.
// This is useful for UI components or status monitoring.
//
// Returns:
//   - Slice of all agent states
func ListAgentStates() []iface.AgentState {
	return []iface.AgentState{
		iface.StateInitializing,
		iface.StateReady,
		iface.StateRunning,
		iface.StatePaused,
		iface.StateError,
		iface.StateShutdown,
	}
}

// GetAgentStateString returns a human-readable string for an agent state.
//
// Parameters:
//   - state: Agent state to convert
//
// Returns:
//   - Human-readable state string
func GetAgentStateString(state iface.AgentState) string {
	switch state {
	case iface.StateInitializing:
		return "Initializing"
	case iface.StateReady:
		return "Ready"
	case iface.StateRunning:
		return "Running"
	case iface.StatePaused:
		return "Paused"
	case iface.StateError:
		return "Error"
	case iface.StateShutdown:
		return "Shutdown"
	default:
		return "Unknown"
	}
}

// Option functions for configuring agents are defined in config.go

// logWithOTELContext extracts OTEL trace/span IDs from context and logs with structured logging.
func logWithOTELContext(ctx context.Context, level slog.Level, msg string, attrs ...any) {
	// Extract OTEL context
	spanCtx := trace.SpanContextFromContext(ctx)
	if spanCtx.IsValid() {
		otelAttrs := []any{
			"trace_id", spanCtx.TraceID().String(),
			"span_id", spanCtx.SpanID().String(),
		}
		attrs = append(otelAttrs, attrs...)
	}

	// Use slog for structured logging
	logger := slog.Default()
	logger.Log(ctx, level, msg, attrs...)
}

// Compile-time checks to ensure implementations satisfy interfaces.
var (
	_ iface.CompositeAgent = (*base.BaseAgent)(nil)
	_ iface.Executor       = (*executor.AgentExecutor)(nil)
)
