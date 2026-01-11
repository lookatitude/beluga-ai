package textsplitters

import (
	"github.com/lookatitude/beluga-ai/pkg/textsplitters/iface"
	_ "github.com/lookatitude/beluga-ai/pkg/textsplitters/providers/markdown"
	_ "github.com/lookatitude/beluga-ai/pkg/textsplitters/providers/recursive"
)

// init registers built-in splitters with the global registry.
func init() {
	registry := GetRegistry()

	// Register recursive splitter
	registry.Register("recursive", func(config map[string]any) (iface.TextSplitter, error) {
		var opts []RecursiveOption

		if chunkSize, ok := config["chunk_size"].(int); ok {
			opts = append(opts, WithRecursiveChunkSize(chunkSize))
		}
		if chunkOverlap, ok := config["chunk_overlap"].(int); ok {
			opts = append(opts, WithRecursiveChunkOverlap(chunkOverlap))
		}
		if separators, ok := config["separators"].([]string); ok {
			opts = append(opts, WithSeparators(separators...))
		}

		return NewRecursiveCharacterTextSplitter(opts...)
	})

	// Register markdown splitter
	registry.Register("markdown", func(config map[string]any) (iface.TextSplitter, error) {
		var opts []MarkdownOption

		if chunkSize, ok := config["chunk_size"].(int); ok {
			opts = append(opts, WithMarkdownChunkSize(chunkSize))
		}
		if chunkOverlap, ok := config["chunk_overlap"].(int); ok {
			opts = append(opts, WithMarkdownChunkOverlap(chunkOverlap))
		}
		if headers, ok := config["headers_to_split_on"].([]string); ok {
			opts = append(opts, WithHeadersToSplitOn(headers...))
		}

		return NewMarkdownTextSplitter(opts...)
	})
}
