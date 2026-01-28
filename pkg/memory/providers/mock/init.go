// Package mock provides mock implementations for testing memory components.
// This package exists to enable auto-registration via blank imports.
package mock

// Note: Registration is handled in pkg/memory/registry.go.
// Import this package with a blank import to ensure the provider is registered:
//
//     import _ "github.com/lookatitude/beluga-ai/pkg/memory/providers/mock"
