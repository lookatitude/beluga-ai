// Package iface defines the core interfaces for the documentloaders package.
package iface

import (
	"context"

	"github.com/lookatitude/beluga-ai/pkg/core"
	"github.com/lookatitude/beluga-ai/pkg/schema"
)

// DocumentLoader defines the interface for loading documents from various sources.
// It embeds core.Loader to ensure compatibility with existing RAG pipeline components.
// Implementations should be stateless and thread-safe.
type DocumentLoader interface {
	core.Loader

	// Load reads all documents from the configured source.
	// Returns all successfully loaded documents and an error if any failures occurred.
	// Implementations MUST respect context cancellation.
	// Implementations MUST populate Document.Metadata with at least "source" key.
	Load(ctx context.Context) ([]schema.Document, error)

	// LazyLoad provides an alternative way to load data, returning a channel that yields
	// documents one by one as they become available.
	// This is useful for large datasets or sources where loading everything at once is inefficient.
	// Errors encountered during loading should be sent on the channel.
	// The channel yields items of type schema.Document or error.
	LazyLoad(ctx context.Context) (<-chan any, error)
}
