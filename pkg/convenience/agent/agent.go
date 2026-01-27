// Package agent provides a simplified API for creating agents with memory and tools.
// It reduces boilerplate by providing a fluent builder pattern for agent configuration.
//
// Example usage:
//
//	agent, err := agent.NewBuilder().
//	    WithLLM(llm).
//	    WithBufferMemory(50).
//	    WithTool(calculator).
//	    WithSystemPrompt("You are helpful").
//	    Build(ctx)
//
//	result, err := agent.Run(ctx, "Calculate 2+2")
package agent

import (
	"context"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/core"
	llmsiface "github.com/lookatitude/beluga-ai/pkg/llms/iface"
	"github.com/lookatitude/beluga-ai/pkg/memory"
	memoryiface "github.com/lookatitude/beluga-ai/pkg/memory/iface"
)

// Builder provides a fluent interface for constructing agents.
type Builder struct {
	// Basic configuration
	systemPrompt string
	name         string
	maxTurns     int
	verbose      bool
	agentType    string

	// LLM configuration
	llm         llmsiface.LLM
	chatModel   llmsiface.ChatModel
	llmProvider string
	llmAPIKey   string
	llmModel    string

	// Memory configuration
	memory       memoryiface.Memory
	memoryType   string
	memorySize   int
	enableMemory bool

	// Tools configuration
	tools []core.Tool

	// Execution configuration
	timeout time.Duration

	// Metrics
	metrics *Metrics
}

// NewBuilder creates a new agent builder with sensible defaults.
func NewBuilder() *Builder {
	return &Builder{
		name:         "assistant",
		maxTurns:     10,
		agentType:    "react",
		memorySize:   50,
		enableMemory: false,
		timeout:      30 * time.Second,
	}
}

// WithSystemPrompt sets the system prompt for the agent.
func (b *Builder) WithSystemPrompt(prompt string) *Builder {
	b.systemPrompt = prompt
	return b
}

// WithName sets the agent name.
func (b *Builder) WithName(name string) *Builder {
	b.name = name
	return b
}

// WithMaxTurns sets the maximum number of turns.
func (b *Builder) WithMaxTurns(max int) *Builder {
	b.maxTurns = max
	return b
}

// WithVerbose enables verbose output.
func (b *Builder) WithVerbose(verbose bool) *Builder {
	b.verbose = verbose
	return b
}

// WithAgentType sets the agent type (e.g., "react", "tool_calling").
func (b *Builder) WithAgentType(agentType string) *Builder {
	b.agentType = agentType
	return b
}

// WithLLM sets a pre-configured LLM instance.
func (b *Builder) WithLLM(llm llmsiface.LLM) *Builder {
	b.llm = llm
	return b
}

// WithChatModel sets a pre-configured ChatModel instance.
func (b *Builder) WithChatModel(model llmsiface.ChatModel) *Builder {
	b.chatModel = model
	return b
}

// WithLLMProvider sets the LLM provider for automatic resolution.
// The provider name (e.g., "openai", "anthropic") and API key are used
// to create an LLM instance from the global registry.
func (b *Builder) WithLLMProvider(provider, apiKey string) *Builder {
	b.llmProvider = provider
	b.llmAPIKey = apiKey
	return b
}

// WithLLMModel sets the specific model name to use with the LLM provider.
func (b *Builder) WithLLMModel(model string) *Builder {
	b.llmModel = model
	return b
}

// WithBufferMemory enables buffer memory with a specified maximum message count.
// The memory will store up to maxMessages messages.
func (b *Builder) WithBufferMemory(maxMessages int) *Builder {
	b.enableMemory = true
	b.memoryType = "buffer"
	b.memorySize = maxMessages
	return b
}

// WithWindowMemory enables window buffer memory with a specified window size.
// The memory will keep the most recent k interactions.
func (b *Builder) WithWindowMemory(windowSize int) *Builder {
	b.enableMemory = true
	b.memoryType = "window"
	b.memorySize = windowSize
	return b
}

// WithMemory sets a pre-configured memory instance.
func (b *Builder) WithMemory(mem memoryiface.Memory) *Builder {
	b.memory = mem
	b.enableMemory = true
	return b
}

// WithTool adds a single tool to the agent.
func (b *Builder) WithTool(tool core.Tool) *Builder {
	b.tools = append(b.tools, tool)
	return b
}

// WithTools adds multiple tools to the agent.
func (b *Builder) WithTools(tools []core.Tool) *Builder {
	b.tools = append(b.tools, tools...)
	return b
}

// WithTimeout sets the execution timeout for agent runs.
func (b *Builder) WithTimeout(timeout time.Duration) *Builder {
	b.timeout = timeout
	return b
}

// WithMetrics sets the metrics instance for the agent.
func (b *Builder) WithMetrics(metrics *Metrics) *Builder {
	b.metrics = metrics
	return b
}

// Build creates and returns a configured Agent instance.
// It validates the configuration and creates all necessary components.
//
// Returns an error if:
//   - No LLM is configured (either via WithLLM, WithChatModel, or WithLLMProvider)
//   - LLM creation fails when using provider-based configuration
//   - Memory creation fails when memory is enabled
func (b *Builder) Build(ctx context.Context) (Agent, error) {
	const op = "Build"

	// Get or create metrics
	metrics := b.metrics
	if metrics == nil {
		metrics = GetMetrics()
		if metrics == nil {
			metrics = NoOpMetrics()
		}
	}

	// Start tracing span
	ctx, span := metrics.StartBuildSpan(ctx, b.agentType)
	if span != nil {
		defer span.End()
	}

	// Validate LLM configuration
	if b.llm == nil && b.chatModel == nil && b.llmProvider == "" {
		metrics.RecordBuild(ctx, b.agentType, false)
		return nil, NewError(op, ErrCodeMissingLLM, ErrMissingLLM)
	}

	// Resolve LLM if using provider-based configuration
	var llm llmsiface.LLM
	var chatModel llmsiface.ChatModel

	if b.chatModel != nil {
		chatModel = b.chatModel
		llm = b.chatModel // ChatModel extends LLM
	} else if b.llm != nil {
		llm = b.llm
		// Check if LLM also implements ChatModel
		if cm, ok := b.llm.(llmsiface.ChatModel); ok {
			chatModel = cm
		}
	} else if b.llmProvider != "" {
		// Provider-based creation would go here
		// For now, return an error as we don't have registry integration yet
		metrics.RecordBuild(ctx, b.agentType, false)
		return nil, NewErrorWithMessage(op, ErrCodeLLMCreation,
			"provider-based LLM creation not yet implemented, use WithLLM or WithChatModel", nil)
	}

	// Create memory if enabled
	var mem memoryiface.Memory
	if b.enableMemory {
		if b.memory != nil {
			mem = b.memory
		} else {
			var err error
			switch b.memoryType {
			case "buffer":
				mem, err = memory.NewMemory(memory.MemoryTypeBuffer,
					memory.WithWindowSize(b.memorySize),
					memory.WithReturnMessages(true),
				)
			case "window":
				mem, err = memory.NewMemory(memory.MemoryTypeBufferWindow,
					memory.WithWindowSize(b.memorySize),
					memory.WithReturnMessages(true),
				)
			default:
				mem, err = memory.NewMemory(memory.MemoryTypeBuffer,
					memory.WithWindowSize(b.memorySize),
					memory.WithReturnMessages(true),
				)
			}
			if err != nil {
				metrics.RecordBuild(ctx, b.agentType, false)
				return nil, NewError(op, ErrCodeMemoryCreation, err)
			}
		}
	}

	// Create the convenience agent
	agent := &convenienceAgent{
		name:         b.name,
		systemPrompt: b.systemPrompt,
		llm:          llm,
		chatModel:    chatModel,
		memory:       mem,
		tools:        b.tools,
		maxTurns:     b.maxTurns,
		timeout:      b.timeout,
		verbose:      b.verbose,
		agentType:    b.agentType,
		metrics:      metrics,
	}

	metrics.RecordBuild(ctx, b.agentType, true)
	return agent, nil
}

// Getter methods for builder configuration

// GetSystemPrompt returns the configured system prompt.
func (b *Builder) GetSystemPrompt() string {
	return b.systemPrompt
}

// GetName returns the configured agent name.
func (b *Builder) GetName() string {
	return b.name
}

// GetMaxTurns returns the configured max turns.
func (b *Builder) GetMaxTurns() int {
	return b.maxTurns
}

// GetAgentType returns the configured agent type.
func (b *Builder) GetAgentType() string {
	return b.agentType
}
