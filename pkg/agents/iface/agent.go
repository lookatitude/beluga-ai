// Package iface defines the core interfaces for the agents package.
// It follows the Interface Segregation Principle by providing small, focused interfaces
// that serve specific purposes within the agent system.
package iface

import (
	"context"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/core"
	llmsiface "github.com/lookatitude/beluga-ai/pkg/llms/iface"
	"github.com/lookatitude/beluga-ai/pkg/schema"
	"go.opentelemetry.io/otel/trace"
)

// AgentAction represents an action to be taken by the agent.
// It encapsulates the tool to use, its input, and any associated logging.
type AgentAction struct {
	Tool      string // The name of the tool to use
	ToolInput any    // The input to the tool (can be string, map[string]any, etc.)
	Log       string // Additional logging information (e.g., the LLM's thought process)
}

// AgentFinish represents the final response from the agent.
// It contains the return values and any final logging information.
type AgentFinish struct {
	ReturnValues map[string]any // Final output values
	Log          string         // Additional logging information
}

// Planner defines the interface for planning the next action or determining completion.
// This follows the Single Responsibility Principle by focusing solely on decision-making.
type Planner interface {
	// Plan decides the next action or finish state based on intermediate steps and inputs.
	Plan(ctx context.Context, intermediateSteps []IntermediateStep, inputs map[string]any) (AgentAction, AgentFinish, error)
}

// IntermediateStep represents a single step in the agent's execution process.
type IntermediateStep struct {
	Action      AgentAction
	Observation string
}

// Agent defines the core interface for AI agents.
// It combines planning capabilities with tool and configuration management.
type Agent interface {
	Planner

	// InputVariables returns the expected input keys for the agent.
	InputVariables() []string

	// OutputVariables returns the keys for the agent's final output.
	OutputVariables() []string

	// GetTools returns the tools available to the agent.
	GetTools() []Tool

	// GetConfig returns the agent's configuration.
	GetConfig() schema.AgentConfig

	// GetLLM returns the LLM instance used by the agent.
	GetLLM() llmsiface.LLM

	// GetMetrics returns the metrics recorder for the agent.
	GetMetrics() MetricsRecorder
}

// Executor defines the interface for executing agent plans.
// This allows for different execution strategies (sequential, parallel, etc.).
type Executor interface {
	// ExecutePlan runs the given plan for the specified agent.
	ExecutePlan(ctx context.Context, agent Agent, plan []schema.Step) (schema.FinalAnswer, error)
}

// AgentFactory defines the interface for creating agent instances.
// It enables dependency injection and different agent creation strategies.
type AgentFactory interface {
	// CreateAgent creates a new agent instance based on the provided configuration.
	CreateAgent(ctx context.Context, config schema.AgentConfig) (Agent, error)
}

// LifecycleManager defines the interface for managing agent lifecycle.
// It handles initialization, execution, and cleanup operations.
type LifecycleManager interface {
	// Initialize sets up the agent with necessary configurations.
	Initialize(config map[string]any) error

	// Execute performs the main task of the agent.
	Execute() error

	// Shutdown gracefully stops the agent and cleans up resources.
	Shutdown() error

	// GetState returns the current state of the agent.
	GetState() AgentState

	// CheckHealth returns the health status of the agent.
	CheckHealth() map[string]any
}

// AgentState represents the current state of an agent.
type AgentState string

const (
	StateInitializing AgentState = "initializing"
	StateReady        AgentState = "ready"
	StateRunning      AgentState = "running"
	StatePaused       AgentState = "paused"
	StateError        AgentState = "error"
	StateShutdown     AgentState = "shutdown"
)

// EventHandler defines a function type for handling agent events.
type EventHandler func(payload any) error

// EventEmitter defines the interface for emitting and handling events.
type EventEmitter interface {
	// RegisterEventHandler registers a handler function for a specific event type.
	RegisterEventHandler(eventType string, handler EventHandler)

	// EmitEvent triggers all registered handlers for the given event type.
	EmitEvent(eventType string, payload any)
}

// Option represents a functional option for configuring agents.
type Option func(*Options)

// Options holds the configuration options for an agent.
type Options struct {
	Metrics         MetricsRecorder
	EventHandlers   map[string][]EventHandler
	MaxRetries      int
	RetryDelay      time.Duration
	Timeout         time.Duration
	MaxIterations   int
	EnableMetrics   bool
	EnableTracing   bool
	EnableSafety    bool
	StreamingConfig StreamingConfig
}

// HealthChecker defines the interface for health checking components.
type HealthChecker interface {
	// CheckHealth returns the health status information.
	CheckHealth() map[string]any
}

// MetricsRecorder defines the interface for recording metrics.
type MetricsRecorder interface {
	// StartAgentSpan starts a tracing span for agent operations.
	StartAgentSpan(ctx context.Context, agentName, operation string) (context.Context, trace.Span)

	// RecordAgentExecution records agent execution metrics.
	RecordAgentExecution(ctx context.Context, agentName, agentType string, duration time.Duration, success bool)

	// RecordPlanningCall records planning operation metrics.
	RecordPlanningCall(ctx context.Context, agentName string, duration time.Duration, success bool)

	// RecordExecutorRun records executor run metrics.
	RecordExecutorRun(ctx context.Context, executorType string, duration time.Duration, steps int, success bool)

	// RecordToolCall records tool call metrics.
	RecordToolCall(ctx context.Context, toolName string, duration time.Duration, success bool)

	// RecordStreamingOperation records streaming operation metrics (latency and duration).
	RecordStreamingOperation(ctx context.Context, agentName string, latency, duration time.Duration)

	// RecordStreamingChunk records that a streaming chunk was produced.
	RecordStreamingChunk(ctx context.Context, agentName string)
}

// SpanEnder defines an interface for span ending operations.
type SpanEnder interface {
	End(options ...trace.SpanEndOption)
}

// CompositeAgent combines the core Agent interface with lifecycle management,
// health checking capabilities, and Runnable interface through interface embedding.
type CompositeAgent interface {
	core.Runnable
	Agent
	LifecycleManager
	HealthChecker
	EventEmitter
}
