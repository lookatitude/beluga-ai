// Package multiquery provides registration for the multiquery retriever provider.
// The actual implementation is in the main retrievers package for backward compatibility.
// This package exists to enable auto-registration via blank imports.
package multiquery

// Note: Registration is handled in pkg/retrievers/registry_providers.go to avoid import cycles.
// Import this package with a blank import to ensure the provider is registered:
//
//     import _ "github.com/lookatitude/beluga-ai/pkg/retrievers/providers/multiquery"
