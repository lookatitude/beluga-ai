// Package window provides registration for the window memory provider.
// The actual implementation is in the internal/window package.
// This package exists to enable auto-registration via blank imports.
package window

// Note: Registration is handled in pkg/memory/registry.go.
// Import this package with a blank import to ensure the provider is registered:
//
//     import _ "github.com/lookatitude/beluga-ai/pkg/memory/providers/window"
