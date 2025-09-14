// Package memory provides interfaces and implementations for managing conversation history.
// It follows the Beluga AI Framework design patterns with proper separation of concerns,
// configuration management, observability, and error handling.
//
// The package supports multiple memory types:
// - Buffer memory for storing all messages
// - Window buffer memory for keeping a fixed number of recent interactions
// - Summary memory for condensing conversations using LLMs
// - Summary buffer memory combining buffer and summarization
// - Vector store memory for semantic retrieval
package memory

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/lookatitude/beluga-ai/pkg/core"
	embeddingsiface "github.com/lookatitude/beluga-ai/pkg/embeddings/iface"
	"github.com/lookatitude/beluga-ai/pkg/memory/iface"
	"github.com/lookatitude/beluga-ai/pkg/memory/internal/buffer"
	"github.com/lookatitude/beluga-ai/pkg/memory/internal/summary"
	"github.com/lookatitude/beluga-ai/pkg/memory/internal/vectorstore"
	"github.com/lookatitude/beluga-ai/pkg/memory/internal/window"
	"github.com/lookatitude/beluga-ai/pkg/memory/providers"
	"github.com/lookatitude/beluga-ai/pkg/schema"
	"github.com/lookatitude/beluga-ai/pkg/vectorstores"
)

func init() {
	// Initialize global tracer and logger
	SetGlobalTracer()
}

// Ensure the main interfaces are imported for external use
type Memory = iface.Memory
type ChatMessageHistory = iface.ChatMessageHistory

// Factory defines the interface for creating Memory instances.
type Factory interface {
	CreateMemory(ctx context.Context, config Config) (Memory, error)
}

// DefaultFactory is the default implementation of the Factory interface.
type DefaultFactory struct{}

// NewFactory creates a new memory factory.
func NewFactory() Factory {
	return &DefaultFactory{}
}

// CreateMemory creates a memory instance based on the provided configuration.
// It validates the configuration and instantiates the appropriate memory implementation.
func (f *DefaultFactory) CreateMemory(ctx context.Context, config Config) (Memory, error) {
	start := time.Now()
	tracer := GetGlobalTracer()
	metrics := GetGlobalMetrics()

	// Start tracing span
	ctx, span := tracer.StartSpan(ctx, "create_memory", config.Type, config.MemoryKey)
	defer span.End()

	// Validate configuration
	validate := validator.New()
	if err := validate.Struct(config); err != nil {
		duration := time.Since(start)
		if metrics != nil {
			metrics.RecordError(ctx, "create_memory", config.Type, "invalid_config")
			metrics.RecordOperationDuration(ctx, "create_memory", config.Type, duration)
			metrics.RecordOperation(ctx, "create_memory", config.Type, false)
		}
		LogError(ctx, err, "create_memory", config.Type, config.MemoryKey)
		tracer.RecordSpanError(span, err)
		return nil, fmt.Errorf("invalid memory configuration: %w", err)
	}

	if !config.Enabled {
		// Return a no-op memory implementation
		duration := time.Since(start)
		if metrics != nil {
			metrics.RecordOperationDuration(ctx, "create_memory", config.Type, duration)
			metrics.RecordOperation(ctx, "create_memory", config.Type, true)
		}
		LogMemoryLifecycle(ctx, "created_noop_memory", config.Type, config.MemoryKey)
		return &NoOpMemory{}, nil
	}

	// Create memory based on type
	var memory Memory
	var err error

	switch config.Type {
	case MemoryTypeBuffer:
		memory, err = f.createBufferMemory(ctx, config)
	case MemoryTypeBufferWindow:
		memory, err = f.createBufferWindowMemory(ctx, config)
	case MemoryTypeSummary:
		memory, err = f.createSummaryMemory(ctx, config)
	case MemoryTypeSummaryBuffer:
		memory, err = f.createSummaryBufferMemory(ctx, config)
	case MemoryTypeVectorStore:
		memory, err = f.createVectorStoreMemory(ctx, config)
	case MemoryTypeVectorStoreRetriever:
		memory, err = f.createVectorStoreRetrieverMemory(ctx, config)
	default:
		err = fmt.Errorf("unsupported memory type: %s", config.Type)
	}

	duration := time.Since(start)

	if err != nil {
		if metrics != nil {
			metrics.RecordError(ctx, "create_memory", config.Type, "creation_error")
			metrics.RecordOperationDuration(ctx, "create_memory", config.Type, duration)
			metrics.RecordOperation(ctx, "create_memory", config.Type, false)
		}
		LogError(ctx, err, "create_memory", config.Type, config.MemoryKey)
		tracer.RecordSpanError(span, err)
		return nil, err
	}

	// Record success metrics and logging
	if metrics != nil {
		metrics.RecordOperationDuration(ctx, "create_memory", config.Type, duration)
		metrics.RecordOperation(ctx, "create_memory", config.Type, true)
		metrics.RecordActiveMemory(ctx, config.Type, 1)
	}
	LogMemoryLifecycle(ctx, "created_memory", config.Type, config.MemoryKey,
		slog.String("memory_variables", fmt.Sprintf("%v", memory.MemoryVariables())))

	return memory, nil
}

// createBufferMemory creates a buffer memory instance.
func (f *DefaultFactory) createBufferMemory(ctx context.Context, config Config) (Memory, error) {
	history := providers.NewBaseChatMessageHistory()
	memory := buffer.NewChatMessageBufferMemory(history)

	// Apply configuration
	memory.MemoryKey = config.MemoryKey
	memory.InputKey = config.InputKey
	memory.OutputKey = config.OutputKey
	memory.ReturnMessages = config.ReturnMessages
	memory.HumanPrefix = config.HumanPrefix
	memory.AIPrefix = config.AIPrefix

	return memory, nil
}

// createBufferWindowMemory creates a buffer window memory instance.
func (f *DefaultFactory) createBufferWindowMemory(ctx context.Context, config Config) (Memory, error) {
	history := providers.NewBaseChatMessageHistory()
	memory := window.NewConversationBufferWindowMemory(history, config.WindowSize, config.MemoryKey, config.ReturnMessages)

	// Apply configuration
	memory.InputKey = config.InputKey
	memory.OutputKey = config.OutputKey
	memory.HumanPrefix = config.HumanPrefix
	memory.AiPrefix = config.AIPrefix

	return memory, nil
}

// createSummaryMemory creates a summary memory instance.
// Note: This requires an LLM to be provided via dependency injection.
func (f *DefaultFactory) createSummaryMemory(ctx context.Context, config Config) (Memory, error) {
	// This is a placeholder - in practice, the LLM would need to be injected
	// through a more sophisticated factory pattern or configuration
	return nil, fmt.Errorf("summary memory requires LLM dependency injection - use NewConversationSummaryMemory directly")
}

// createSummaryBufferMemory creates a summary buffer memory instance.
// Note: This requires an LLM to be provided via dependency injection.
func (f *DefaultFactory) createSummaryBufferMemory(ctx context.Context, config Config) (Memory, error) {
	// This is a placeholder - in practice, the LLM would need to be injected
	return nil, fmt.Errorf("summary buffer memory requires LLM dependency injection - use NewConversationSummaryBufferMemory directly")
}

// createVectorStoreMemory creates a vector store memory instance.
// Note: This requires a retriever to be provided via dependency injection.
func (f *DefaultFactory) createVectorStoreMemory(ctx context.Context, config Config) (Memory, error) {
	// This is a placeholder - in practice, the retriever would need to be injected
	return nil, fmt.Errorf("vector store memory requires retriever dependency injection - use NewVectorStoreMemory directly")
}

// createVectorStoreRetrieverMemory creates a vector store retriever memory instance.
// Note: This requires embedder and vector store to be provided via dependency injection.
func (f *DefaultFactory) createVectorStoreRetrieverMemory(ctx context.Context, config Config) (Memory, error) {
	// This is a placeholder - in practice, the embedder and vector store would need to be injected
	return nil, fmt.Errorf("vector store retriever memory requires embedder and vector store dependency injection - use NewVectorStoreRetrieverMemory directly")
}

// NewMemory is a convenience function for creating memory instances with functional options.
func NewMemory(memoryType MemoryType, options ...Option) (Memory, error) {
	config := Config{
		Type:           memoryType,
		MemoryKey:      "history",
		InputKey:       "input",
		OutputKey:      "output",
		ReturnMessages: false,
		WindowSize:     5,
		MaxTokenLimit:  2000,
		TopK:           4,
		HumanPrefix:    "Human",
		AIPrefix:       "AI",
		Enabled:        true,
		Timeout:        30 * time.Second,
	}

	// Apply options
	for _, option := range options {
		option(&config)
	}

	factory := NewFactory()
	ctx := context.Background()
	return factory.CreateMemory(ctx, config)
}

// NoOpMemory is a no-op implementation of the Memory interface.
// It can be used when memory is disabled or for testing purposes.
type NoOpMemory struct{}

// MemoryVariables returns an empty slice for no-op memory.
func (m *NoOpMemory) MemoryVariables() []string {
	return []string{}
}

// LoadMemoryVariables returns an empty map for no-op memory.
func (m *NoOpMemory) LoadMemoryVariables(ctx context.Context, inputs map[string]any) (map[string]any, error) {
	return map[string]any{}, nil
}

// SaveContext does nothing for no-op memory.
func (m *NoOpMemory) SaveContext(ctx context.Context, inputs map[string]any, outputs map[string]any) error {
	return nil
}

// Clear does nothing for no-op memory.
func (m *NoOpMemory) Clear(ctx context.Context) error {
	return nil
}

// GetInputOutputKeys determines the input and output keys from the given maps.
// This utility function is exposed for use by memory implementations.
func GetInputOutputKeys(inputs map[string]any, outputs map[string]any) (string, string, error) {
	if len(inputs) == 0 {
		return "", "", fmt.Errorf("inputs map is empty")
	}
	if len(outputs) == 0 {
		return "", "", fmt.Errorf("outputs map is empty")
	}

	// Common input/output key names
	possibleInputKeys := []string{"input", "query", "question", "human_input", "user_input"}
	possibleOutputKeys := []string{"output", "result", "answer", "ai_output", "response"}

	// Try to find known input key
	var inputKey string
	for _, key := range possibleInputKeys {
		if _, ok := inputs[key]; ok {
			inputKey = key
			break
		}
	}

	// If no known input key, use the first key
	if inputKey == "" {
		for k := range inputs {
			inputKey = k
			break
		}
	}

	// Try to find known output key
	var outputKey string
	for _, key := range possibleOutputKeys {
		if _, ok := outputs[key]; ok {
			outputKey = key
			break
		}
	}

	// If no known output key, use the first key
	if outputKey == "" {
		for k := range outputs {
			outputKey = k
			break
		}
	}

	return inputKey, outputKey, nil
}

// GetBufferString formats messages into a text buffer with human/AI prefixes.
// This utility function is exposed for use by memory implementations.
func GetBufferString(messages []schema.Message, humanPrefix, aiPrefix string) string {
	var buffer strings.Builder

	for _, msg := range messages {
		switch msg.GetType() {
		case schema.RoleHuman:
			buffer.WriteString(fmt.Sprintf("%s: %s\n", humanPrefix, msg.GetContent()))
		case schema.RoleAssistant:
			buffer.WriteString(fmt.Sprintf("%s: %s\n", aiPrefix, msg.GetContent()))
		case schema.RoleSystem:
			buffer.WriteString(fmt.Sprintf("System: %s\n", msg.GetContent()))
		case schema.RoleTool:
			toolMsg, ok := msg.(*schema.ToolMessage)
			if ok {
				buffer.WriteString(fmt.Sprintf("Tool (%s): %s\n", toolMsg.ToolCallID, msg.GetContent()))
			} else {
				buffer.WriteString(fmt.Sprintf("Tool: %s\n", msg.GetContent()))
			}
		default:
			buffer.WriteString(fmt.Sprintf("%s: %s\n", msg.GetType(), msg.GetContent()))
		}
	}

	return buffer.String()
}

// Convenience functions for creating specific memory types

// NewBaseChatMessageHistory creates a new base chat message history with functional options.
func NewBaseChatMessageHistory(options ...providers.BaseHistoryOption) iface.ChatMessageHistory {
	return providers.NewBaseChatMessageHistory(options...)
}

// WithMaxHistorySize sets the maximum number of messages for the base history.
func WithMaxHistorySize(maxSize int) providers.BaseHistoryOption {
	return providers.WithMaxHistorySize(maxSize)
}

// NewCompositeChatMessageHistory creates a new composite chat message history with functional options.
func NewCompositeChatMessageHistory(primary iface.ChatMessageHistory, options ...providers.CompositeHistoryOption) iface.ChatMessageHistory {
	return providers.NewCompositeChatMessageHistory(primary, options...)
}

// WithSecondaryHistory sets a secondary history for the composite history.
func WithSecondaryHistory(secondary iface.ChatMessageHistory) providers.CompositeHistoryOption {
	return providers.WithSecondaryHistory(secondary)
}

// WithMaxSize sets the maximum number of messages for the composite history.
func WithMaxSize(maxSize int) providers.CompositeHistoryOption {
	return providers.WithMaxSize(maxSize)
}

// WithOnAddHook sets an add hook for the composite history.
func WithOnAddHook(hook func(context.Context, schema.Message) error) providers.CompositeHistoryOption {
	return providers.WithOnAddHook(hook)
}

// WithOnGetHook sets a get hook for the composite history.
func WithOnGetHook(hook func(context.Context, []schema.Message) ([]schema.Message, error)) providers.CompositeHistoryOption {
	return providers.WithOnGetHook(hook)
}

// NewChatMessageBufferMemory creates a new chat message buffer memory.
func NewChatMessageBufferMemory(history iface.ChatMessageHistory) iface.Memory {
	return buffer.NewChatMessageBufferMemory(history)
}

// NewConversationBufferWindowMemory creates a new conversation buffer window memory.
func NewConversationBufferWindowMemory(history iface.ChatMessageHistory, k int, memoryKey string, returnMessages bool) iface.Memory {
	return window.NewConversationBufferWindowMemory(history, k, memoryKey, returnMessages)
}

// NewConversationSummaryMemory creates a new conversation summary memory.
func NewConversationSummaryMemory(history iface.ChatMessageHistory, llm core.Runnable, memoryKey string) iface.Memory {
	return summary.NewConversationSummaryMemory(history, llm, memoryKey)
}

// NewConversationSummaryBufferMemory creates a new conversation summary buffer memory.
func NewConversationSummaryBufferMemory(history iface.ChatMessageHistory, llm core.Runnable, memoryKey string, maxTokenLimit int) iface.Memory {
	return summary.NewConversationSummaryBufferMemory(history, llm, memoryKey, maxTokenLimit)
}

// NewVectorStoreMemory creates a new vector store memory.
func NewVectorStoreMemory(retriever core.Retriever, memoryKey string, returnDocs bool, k int) iface.Memory {
	return vectorstore.NewVectorStoreMemory(retriever, memoryKey, returnDocs, k)
}

// NewVectorStoreRetrieverMemory creates a new vector store retriever memory.
func NewVectorStoreRetrieverMemory(embedder embeddingsiface.Embedder, vectorStore vectorstores.VectorStore, options ...vectorstore.VectorStoreMemoryOption) iface.Memory {
	return vectorstore.NewVectorStoreRetrieverMemory(embedder, vectorStore, options...)
}
