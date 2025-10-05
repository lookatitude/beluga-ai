package core

import (
	"context"

	"github.com/lookatitude/beluga-ai/pkg/core/iface"
	"github.com/lookatitude/beluga-ai/pkg/schema"
)

// T005: Re-export core interfaces from iface/ for backward compatibility
// This ensures existing imports continue to work while achieving constitutional compliance
type (
	// Core framework interfaces
	Option                = iface.Option
	OptionFunc            = iface.OptionFunc
	Runnable              = iface.Runnable
	HealthChecker         = iface.HealthChecker
	AdvancedHealthChecker = iface.AdvancedHealthChecker

	// Configuration interfaces
	ConfigValidator       = iface.ConfigValidator
	OptionApplier         = iface.OptionApplier
	ConfigurableComponent = iface.ConfigurableComponent
)

// Re-export factory functions for backward compatibility
var (
	// WithOption creates a new Option that sets a specific key-value pair in the configuration map.
	WithOption = func(key string, value any) Option {
		return iface.OptionFunc(func(config *map[string]any) {
			(*config)[key] = value
		})
	}
)

// Note: Core Runnable interface is now defined in iface/runnable.go
// Legacy framework-specific interfaces maintained below:

// Loader defines the interface for loading documents from various sources.
// Loaders are responsible for reading data from files, databases, APIs, etc.
// and converting it into a standardized format (schema.Document).
type Loader interface {
	// Load reads all data from the source and returns it as a slice of Documents.
	Load(ctx context.Context) ([]schema.Document, error)
	// LazyLoad provides an alternative way to load data, returning a channel that yields
	// documents one by one as they become available.
	// This is useful for large datasets or sources where loading everything at once is inefficient.
	// Errors encountered during loading should be sent on the channel.
	// The channel yields items of type schema.Document or error.
	LazyLoad(ctx context.Context) (<-chan any, error)
}

// Retriever defines a generic interface for fetching relevant documents based on a query string.
// While often backed by a VectorStore, retrievers can implement other strategies
// (e.g., keyword search, database lookups, hybrid approaches).
// Retrievers implement the Runnable interface, making them easily pluggable into chains.
type Retriever interface {
	Runnable // Input: string (query), Output: []schema.Document

	// GetRelevantDocuments retrieves documents considered relevant to the given query string.
	GetRelevantDocuments(ctx context.Context, query string) ([]schema.Document, error)
}

// Note: HealthChecker interface is now defined in iface/health.go and re-exported above
