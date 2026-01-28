// Package retrievers provides custom error types for the retrievers package.
// This file re-exports error types from iface for backward compatibility.
package retrievers

import (
	"github.com/lookatitude/beluga-ai/pkg/retrievers/iface"
)

// Error codes for programmatic error handling.
// These are re-exported from iface for backward compatibility.
const (
	ErrCodeInvalidConfig         = iface.ErrCodeInvalidConfig
	ErrCodeInvalidInput          = iface.ErrCodeInvalidInput
	ErrCodeRetrievalFailed       = iface.ErrCodeRetrievalFailed
	ErrCodeEmbeddingFailed       = iface.ErrCodeEmbeddingFailed
	ErrCodeVectorStoreError      = iface.ErrCodeVectorStoreError
	ErrCodeTimeout               = iface.ErrCodeTimeout
	ErrCodeRateLimit             = iface.ErrCodeRateLimit
	ErrCodeNetworkError          = iface.ErrCodeNetworkError
	ErrCodeQueryGenerationFailed = iface.ErrCodeQueryGenerationFailed
	ErrCodeProviderNotFound      = iface.ErrCodeProviderNotFound
)

// Type aliases for backward compatibility.
type (
	// RetrieverError represents an error that occurred during retrieval operations.
	RetrieverError = iface.RetrieverError

	// ValidationError represents a configuration validation error.
	ValidationError = iface.ValidationError

	// TimeoutError represents a timeout error.
	TimeoutError = iface.TimeoutError
)

// Constructor function aliases for backward compatibility.
var (
	NewRetrieverError            = iface.NewRetrieverError
	NewRetrieverErrorWithMessage = iface.NewRetrieverErrorWithMessage
	NewProviderNotFoundError     = iface.NewProviderNotFoundError
	NewTimeoutError              = iface.NewTimeoutError
)
