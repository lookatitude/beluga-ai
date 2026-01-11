package textsplitters

import (
	"github.com/lookatitude/beluga-ai/pkg/textsplitters/iface"
	"github.com/lookatitude/beluga-ai/pkg/textsplitters/providers/markdown"
	"github.com/lookatitude/beluga-ai/pkg/textsplitters/providers/recursive"
)

// NewRecursiveCharacterTextSplitter creates a splitter that recursively splits
// text using a hierarchy of separators (paragraph, line, word, character).
// This is the recommended splitter for most use cases as it respects natural text boundaries.
//
// The splitter tries separators in order:
//   - "\n\n" - Paragraph breaks (preferred)
//   - "\n" - Line breaks
//   - " " - Word boundaries
//   - "" - Character-level (fallback)
//
// Parameters:
//   - opts: Optional configuration functions (e.g., WithRecursiveChunkSize, WithSeparators)
//
// Returns:
//   - TextSplitter: A configured recursive splitter instance
//   - error: Returns SplitterError if configuration is invalid
//
// Example:
//
//	splitter, err := NewRecursiveCharacterTextSplitter(
//	    WithRecursiveChunkSize(1000),
//	    WithRecursiveChunkOverlap(200),
//	)
func NewRecursiveCharacterTextSplitter(opts ...RecursiveOption) (iface.TextSplitter, error) {
	cfg := DefaultRecursiveConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	// Convert to recursive package's config type
	recursiveCfg := &recursive.RecursiveConfig{
		ChunkSize:      cfg.ChunkSize,
		ChunkOverlap:   cfg.ChunkOverlap,
		LengthFunction: cfg.LengthFunction,
		Separators:     cfg.Separators,
	}

	return recursive.NewRecursiveCharacterTextSplitter(recursiveCfg)
}

// NewMarkdownTextSplitter creates a splitter that respects markdown structure.
// Splits preferentially at header boundaries while respecting chunk size limits.
// This splitter is ideal for markdown documents as it preserves semantic structure.
//
// Parameters:
//   - opts: Optional configuration functions (e.g., WithMarkdownChunkSize, WithHeadersToSplitOn)
//
// Returns:
//   - TextSplitter: A configured markdown splitter instance
//   - error: Returns SplitterError if configuration is invalid
//
// Example:
//
//	splitter, err := NewMarkdownTextSplitter(
//	    WithMarkdownChunkSize(500),
//	    WithHeadersToSplitOn("#", "##", "###"),
//	)
func NewMarkdownTextSplitter(opts ...MarkdownOption) (iface.TextSplitter, error) {
	cfg := DefaultMarkdownConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	// Convert to markdown package's config type
	markdownCfg := &markdown.MarkdownConfig{
		ChunkSize:       cfg.ChunkSize,
		ChunkOverlap:    cfg.ChunkOverlap,
		LengthFunction:  cfg.LengthFunction,
		HeadersToSplitOn: cfg.HeadersToSplitOn,
		ReturnEachLine:   cfg.ReturnEachLine,
	}

	return markdown.NewMarkdownTextSplitter(markdownCfg)
}
