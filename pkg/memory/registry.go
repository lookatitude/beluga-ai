// Package memory provides a standardized registry pattern for memory creation.
// This follows the Beluga AI Framework design patterns with consistent factory interfaces.
package memory

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/lookatitude/beluga-ai/pkg/memory/iface"
	"github.com/lookatitude/beluga-ai/pkg/memory/internal/buffer"
	"github.com/lookatitude/beluga-ai/pkg/memory/internal/window"
)

// MemoryFactory defines the interface for creating Memory instances.
// This enables dependency injection and makes testing easier.
type MemoryFactory interface {
	// CreateMemory creates a new Memory instance with the given configuration.
	// The config parameter contains memory-type-specific settings.
	CreateMemory(ctx context.Context, config Config) (iface.Memory, error)
}

// MemoryRegistry is the global registry for creating memory instances.
// It maintains a registry of available memory types and their creation functions.
type MemoryRegistry struct {
	creators map[string]func(ctx context.Context, config Config) (iface.Memory, error)
	mu       sync.RWMutex
}

// NewMemoryRegistry creates a new MemoryRegistry instance.
func NewMemoryRegistry() *MemoryRegistry {
	return &MemoryRegistry{
		creators: make(map[string]func(ctx context.Context, config Config) (iface.Memory, error)),
	}
}

// Register registers a new memory type with the registry.
func (r *MemoryRegistry) Register(memoryType string, creator func(ctx context.Context, config Config) (iface.Memory, error)) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.creators[memoryType] = creator
}

// Create creates a new memory instance using the registered memory type.
func (r *MemoryRegistry) Create(ctx context.Context, memoryType string, config Config) (iface.Memory, error) {
	r.mu.RLock()
	creator, exists := r.creators[memoryType]
	r.mu.RUnlock()

	if !exists {
		return nil, NewMemoryErrorWithMessage(
			"create_memory",
			ErrCodeTypeMismatch,
			fmt.Sprintf("memory type '%s' not registered", memoryType),
			fmt.Errorf("memory type '%s' not registered", memoryType),
		)
	}
	return creator(ctx, config)
}

// ListMemoryTypes returns a list of all registered memory type names.
func (r *MemoryRegistry) ListMemoryTypes() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.creators))
	for name := range r.creators {
		names = append(names, name)
	}
	return names
}

// Global registry instance for easy access.
var globalMemoryRegistry = NewMemoryRegistry()

// RegisterMemoryType registers a memory type with the global registry.
func RegisterMemoryType(memoryType string, creator func(ctx context.Context, config Config) (iface.Memory, error)) {
	globalMemoryRegistry.Register(memoryType, creator)
}

// CreateMemory creates a memory using the global registry.
func CreateMemory(ctx context.Context, memoryType string, config Config) (iface.Memory, error) {
	return globalMemoryRegistry.Create(ctx, memoryType, config)
}

// ListAvailableMemoryTypes returns all available memory types from the global registry.
func ListAvailableMemoryTypes() []string {
	return globalMemoryRegistry.ListMemoryTypes()
}

// GetGlobalMemoryRegistry returns the global registry instance for advanced usage.
func GetGlobalMemoryRegistry() *MemoryRegistry {
	return globalMemoryRegistry
}

// init registers the built-in memory types.
func init() {
	// Register built-in memory types
	RegisterMemoryType(string(MemoryTypeBuffer), createBufferMemory)
	RegisterMemoryType(string(MemoryTypeBufferWindow), createBufferWindowMemory)
	RegisterMemoryType(string(MemoryTypeSummary), createSummaryMemory)
	RegisterMemoryType(string(MemoryTypeSummaryBuffer), createSummaryBufferMemory)
	RegisterMemoryType(string(MemoryTypeVectorStore), createVectorStoreMemory)
	RegisterMemoryType(string(MemoryTypeVectorStoreRetriever), createVectorStoreRetrieverMemory)
}

// Built-in memory type creators (moved from DefaultFactory).
func createBufferMemory(ctx context.Context, config Config) (iface.Memory, error) {
	history := NewBaseChatMessageHistory()
	memory := NewChatMessageBufferMemory(history).(*buffer.ChatMessageBufferMemory)

	// Apply configuration
	memory.MemoryKey = config.MemoryKey
	memory.InputKey = config.InputKey
	memory.OutputKey = config.OutputKey
	memory.ReturnMessages = config.ReturnMessages
	memory.HumanPrefix = config.HumanPrefix
	memory.AIPrefix = config.AIPrefix

	return memory, nil
}

func createBufferWindowMemory(ctx context.Context, config Config) (iface.Memory, error) {
	history := NewBaseChatMessageHistory()
	memory := NewConversationBufferWindowMemory(history, config.WindowSize, config.MemoryKey, config.ReturnMessages).(*window.ConversationBufferWindowMemory)

	// Apply configuration
	memory.InputKey = config.InputKey
	memory.OutputKey = config.OutputKey
	memory.HumanPrefix = config.HumanPrefix
	memory.AiPrefix = config.AIPrefix

	return memory, nil
}

func createSummaryMemory(ctx context.Context, config Config) (iface.Memory, error) {
	// This requires an LLM to be provided via dependency injection
	return nil, errors.New("summary memory requires LLM dependency injection - use NewConversationSummaryMemory directly")
}

func createSummaryBufferMemory(ctx context.Context, config Config) (iface.Memory, error) {
	// This requires an LLM to be provided via dependency injection
	return nil, errors.New("summary buffer memory requires LLM dependency injection - use NewConversationSummaryBufferMemory directly")
}

func createVectorStoreMemory(ctx context.Context, config Config) (iface.Memory, error) {
	// This requires a retriever to be provided via dependency injection
	return nil, errors.New("vector store memory requires retriever dependency injection - use NewVectorStoreMemory directly")
}

func createVectorStoreRetrieverMemory(ctx context.Context, config Config) (iface.Memory, error) {
	// This requires embedder and vector store to be provided via dependency injection
	return nil, errors.New("vector store retriever memory requires embedder and vector store dependency injection - use NewVectorStoreRetrieverMemory directly")
}
