// Package iface defines the core interfaces for the textsplitters package.
package iface

import (
	"context"

	"github.com/lookatitude/beluga-ai/pkg/retrievers/iface"
	"github.com/lookatitude/beluga-ai/pkg/schema"
)

// TextSplitter defines the interface for splitting text into chunks.
// It embeds retrievers.iface.Splitter to ensure compatibility with existing RAG pipeline components.
// Implementations should be stateless and thread-safe.
type TextSplitter interface {
	iface.Splitter

	// SplitText splits a single raw text string into a slice of smaller strings.
	// Returns at least one chunk for non-empty input.
	SplitText(ctx context.Context, text string) ([]string, error)

	// SplitDocuments takes a slice of existing documents and splits each one into smaller documents.
	// Each output document inherits the source document's metadata plus:
	// - "chunk_index": 0-based index of this chunk
	// - "chunk_total": total number of chunks from source document
	SplitDocuments(ctx context.Context, documents []schema.Document) ([]schema.Document, error)

	// CreateDocuments takes raw text strings and optional corresponding metadata,
	// splits the text, and creates new schema.Document objects for each chunk.
	CreateDocuments(ctx context.Context, texts []string, metadatas []map[string]any) ([]schema.Document, error)
}
