// Package httputil provides shared HTTP server lifecycle helpers used by the
// server adapter implementations.
//
// It handles graceful startup, shutdown, and readiness signaling for
// net/http servers across the Beluga framework's Layer 4 (Protocol)
// adapters.
//
// This package is internal and must not be imported outside of this module.
package httputil
