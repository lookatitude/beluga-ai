package textsplitters

import (
	"fmt"

	"github.com/go-playground/validator/v10"
)

// SplitterConfig contains common configuration for all splitters.
type SplitterConfig struct {
	LengthFunction func(string) int `mapstructure:"-" yaml:"-"`
	ChunkSize      int              `mapstructure:"chunk_size" yaml:"chunk_size" env:"SPLITTER_CHUNK_SIZE" validate:"required,min=1"`
	ChunkOverlap   int              `mapstructure:"chunk_overlap" yaml:"chunk_overlap" env:"SPLITTER_CHUNK_OVERLAP" validate:"min=0,ltfield=ChunkSize"`
}

// DefaultSplitterConfig returns a SplitterConfig with sensible defaults.
func DefaultSplitterConfig() *SplitterConfig {
	return &SplitterConfig{
		ChunkSize:      1000,
		ChunkOverlap:   200,
		LengthFunction: func(s string) int { return len(s) },
	}
}

// Validate validates the SplitterConfig.
func (c *SplitterConfig) Validate() error {
	validate := validator.New()
	if err := validate.Struct(c); err != nil {
		return NewSplitterError("Validate", ErrCodeInvalidConfig, fmt.Sprintf("configuration validation failed: %v", err), err)
	}
	return nil
}

// RecursiveConfig contains configuration for RecursiveCharacterTextSplitter.
type RecursiveConfig struct {
	SplitterConfig `mapstructure:",squash"`

	// Separators defines the hierarchy of separators to try.
	// Default: ["\n\n", "\n", " ", ""]
	Separators []string `mapstructure:"separators" yaml:"separators" env:"SPLITTER_SEPARATORS"`
}

// DefaultRecursiveConfig returns a RecursiveConfig with sensible defaults.
func DefaultRecursiveConfig() *RecursiveConfig {
	return &RecursiveConfig{
		SplitterConfig: *DefaultSplitterConfig(),
		Separators:     []string{"\n\n", "\n", " ", ""},
	}
}

// Validate validates the RecursiveConfig.
func (c *RecursiveConfig) Validate() error {
	if err := c.SplitterConfig.Validate(); err != nil {
		return err
	}
	return nil
}

// MarkdownConfig contains configuration for MarkdownTextSplitter.
type MarkdownConfig struct {
	SplitterConfig `mapstructure:",squash"`

	// HeadersToSplitOn defines which headers trigger splits.
	// Default: ["#", "##", "###", "####", "#####", "######"]
	HeadersToSplitOn []string `mapstructure:"headers_to_split_on" yaml:"headers_to_split_on"`

	// ReturnEachLine returns each line as a separate chunk if true.
	ReturnEachLine bool `mapstructure:"return_each_line" yaml:"return_each_line"`
}

// DefaultMarkdownConfig returns a MarkdownConfig with sensible defaults.
func DefaultMarkdownConfig() *MarkdownConfig {
	return &MarkdownConfig{
		SplitterConfig:   *DefaultSplitterConfig(),
		HeadersToSplitOn: []string{"#", "##", "###", "####", "#####", "######"},
		ReturnEachLine:   false,
	}
}

// Validate validates the MarkdownConfig.
func (c *MarkdownConfig) Validate() error {
	if err := c.SplitterConfig.Validate(); err != nil {
		return err
	}
	return nil
}

// Option configures a splitter.
type Option func(*SplitterConfig)

// WithChunkSize sets the target chunk size.
func WithChunkSize(size int) Option {
	return func(c *SplitterConfig) {
		c.ChunkSize = size
	}
}

// WithChunkOverlap sets the overlap between consecutive chunks.
func WithChunkOverlap(overlap int) Option {
	return func(c *SplitterConfig) {
		c.ChunkOverlap = overlap
	}
}

// WithLengthFunction sets a custom length function.
// Use this for token-based splitting with LLM tokenizers.
func WithLengthFunction(fn func(string) int) Option {
	return func(c *SplitterConfig) {
		c.LengthFunction = fn
	}
}

// RecursiveOption configures a recursive splitter.
type RecursiveOption func(*RecursiveConfig)

// WithRecursiveChunkSize sets the chunk size for recursive splitter.
func WithRecursiveChunkSize(size int) RecursiveOption {
	return func(c *RecursiveConfig) {
		c.ChunkSize = size
	}
}

// WithRecursiveChunkOverlap sets the chunk overlap for recursive splitter.
func WithRecursiveChunkOverlap(overlap int) RecursiveOption {
	return func(c *RecursiveConfig) {
		c.ChunkOverlap = overlap
	}
}

// WithRecursiveLengthFunction sets the length function for recursive splitter.
func WithRecursiveLengthFunction(fn func(string) int) RecursiveOption {
	return func(c *RecursiveConfig) {
		c.LengthFunction = fn
	}
}

// WithSeparators sets the separator hierarchy.
func WithSeparators(seps ...string) RecursiveOption {
	return func(c *RecursiveConfig) {
		c.Separators = seps
	}
}

// MarkdownOption configures a markdown splitter.
type MarkdownOption func(*MarkdownConfig)

// WithMarkdownChunkSize sets the chunk size for markdown splitter.
func WithMarkdownChunkSize(size int) MarkdownOption {
	return func(c *MarkdownConfig) {
		c.ChunkSize = size
	}
}

// WithMarkdownChunkOverlap sets the chunk overlap for markdown splitter.
func WithMarkdownChunkOverlap(overlap int) MarkdownOption {
	return func(c *MarkdownConfig) {
		c.ChunkOverlap = overlap
	}
}

// WithHeadersToSplitOn sets which markdown headers trigger splits.
func WithHeadersToSplitOn(headers ...string) MarkdownOption {
	return func(c *MarkdownConfig) {
		c.HeadersToSplitOn = headers
	}
}
