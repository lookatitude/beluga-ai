// Package memory provides configuration management for memory implementations.
// It follows the framework's configuration patterns with struct tags for validation
// and functional options for runtime configuration.
package memory

import (
	"time"

	"github.com/lookatitude/beluga-ai/pkg/core"
)

// MemoryType represents the type of memory implementation to use.
type MemoryType string

const (
	// MemoryTypeBuffer represents buffer memory that stores all messages.
	MemoryTypeBuffer MemoryType = "buffer"

	// MemoryTypeBufferWindow represents window buffer memory with a fixed size.
	MemoryTypeBufferWindow MemoryType = "buffer_window"

	// MemoryTypeSummary represents summary memory that condenses conversations.
	MemoryTypeSummary MemoryType = "summary"

	// MemoryTypeSummaryBuffer represents summary buffer memory combining both approaches.
	MemoryTypeSummaryBuffer MemoryType = "summary_buffer"

	// MemoryTypeVectorStore represents vector store memory for semantic retrieval.
	MemoryTypeVectorStore MemoryType = "vector_store"

	// MemoryTypeVectorStoreRetriever represents vector store retriever memory.
	MemoryTypeVectorStoreRetriever MemoryType = "vector_store_retriever"
)

// Config holds the configuration for memory implementations.
// It uses struct tags for validation and mapping to configuration sources.
type Config struct {
	// Type specifies which memory implementation to use
	Type MemoryType `mapstructure:"type" yaml:"type" env:"MEMORY_TYPE" validate:"required,oneof=buffer buffer_window summary summary_buffer vector_store vector_store_retriever"`

	// MemoryKey is the key used for storing memory variables in prompts
	MemoryKey string `mapstructure:"memory_key" yaml:"memory_key" env:"MEMORY_KEY" default:"history"`

	// InputKey is the key for input values in SaveContext operations
	InputKey string `mapstructure:"input_key" yaml:"input_key" env:"INPUT_KEY" default:"input"`

	// OutputKey is the key for output values in SaveContext operations
	OutputKey string `mapstructure:"output_key" yaml:"output_key" env:"OUTPUT_KEY" default:"output"`

	// ReturnMessages indicates whether to return messages directly or formatted strings
	ReturnMessages bool `mapstructure:"return_messages" yaml:"return_messages" env:"RETURN_MESSAGES" default:"false"`

	// WindowSize is the number of interactions to keep for window-based memories
	WindowSize int `mapstructure:"window_size" yaml:"window_size" env:"WINDOW_SIZE" default:"5"`

	// MaxTokenLimit is the maximum token count before triggering summarization
	MaxTokenLimit int `mapstructure:"max_token_limit" yaml:"max_token_limit" env:"MAX_TOKEN_LIMIT" default:"2000"`

	// TopK is the number of documents to retrieve for vector store memories
	TopK int `mapstructure:"top_k" yaml:"top_k" env:"TOP_K" default:"4"`

	// HumanPrefix is the prefix for human messages in formatted output
	HumanPrefix string `mapstructure:"human_prefix" yaml:"human_prefix" env:"HUMAN_PREFIX" default:"Human"`

	// AIPrefix is the prefix for AI messages in formatted output
	AIPrefix string `mapstructure:"ai_prefix" yaml:"ai_prefix" env:"AI_PREFIX" default:"AI"`

	// Enabled indicates whether memory is enabled
	Enabled bool `mapstructure:"enabled" yaml:"enabled" env:"MEMORY_ENABLED" default:"true"`

	// Timeout is the timeout for memory operations
	Timeout time.Duration `mapstructure:"timeout" yaml:"timeout" env:"MEMORY_TIMEOUT" default:"30s"`
}

// BufferConfig holds configuration specific to buffer memory implementations.
type BufferConfig struct {
	// Base memory configuration
	Config `mapstructure:",squash"`

	// ChatHistory is the underlying message history storage
	ChatHistory ChatMessageHistory
}

// SummaryConfig holds configuration specific to summary memory implementations.
type SummaryConfig struct {
	// Base memory configuration
	Config `mapstructure:",squash"`

	// ChatHistory is the underlying message history storage
	ChatHistory ChatMessageHistory

	// LLM is the language model used for generating summaries
	LLM core.Runnable
}

// VectorStoreConfig holds configuration specific to vector store memory implementations.
type VectorStoreConfig struct {
	// Base memory configuration
	Config `mapstructure:",squash"`

	// Retriever is the retriever interface for vector store operations
	Retriever core.Retriever
}

// Option is a functional option for configuring memory implementations.
type Option func(*Config)

// WithMemoryKey sets the memory key.
func WithMemoryKey(key string) Option {
	return func(c *Config) {
		c.MemoryKey = key
	}
}

// WithInputKey sets the input key.
func WithInputKey(key string) Option {
	return func(c *Config) {
		c.InputKey = key
	}
}

// WithOutputKey sets the output key.
func WithOutputKey(key string) Option {
	return func(c *Config) {
		c.OutputKey = key
	}
}

// WithReturnMessages sets whether to return messages directly.
func WithReturnMessages(returnMessages bool) Option {
	return func(c *Config) {
		c.ReturnMessages = returnMessages
	}
}

// WithWindowSize sets the window size for window-based memories.
func WithWindowSize(size int) Option {
	return func(c *Config) {
		c.WindowSize = size
	}
}

// WithMaxTokenLimit sets the maximum token limit for summary memories.
func WithMaxTokenLimit(limit int) Option {
	return func(c *Config) {
		c.MaxTokenLimit = limit
	}
}

// WithTopK sets the number of documents to retrieve.
func WithTopK(k int) Option {
	return func(c *Config) {
		c.TopK = k
	}
}

// WithHumanPrefix sets the human message prefix.
func WithHumanPrefix(prefix string) Option {
	return func(c *Config) {
		c.HumanPrefix = prefix
	}
}

// WithAIPrefix sets the AI message prefix.
func WithAIPrefix(prefix string) Option {
	return func(c *Config) {
		c.AIPrefix = prefix
	}
}

// WithTimeout sets the timeout for operations.
func WithTimeout(timeout time.Duration) Option {
	return func(c *Config) {
		c.Timeout = timeout
	}
}
