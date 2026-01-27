// Package iface defines interfaces for voice backend operations.
//
// Deprecated: This package has been moved to pkg/voicebackend/iface.
// Please update your imports to use github.com/lookatitude/beluga-ai/pkg/voicebackend/iface.
// This package will be removed in v2.0.
package iface

import (
	"context"
)

// VoiceBackend defines the interface for voice backend instances.
// A voice backend manages real-time voice pipelines for AI agents.
type VoiceBackend interface {
	// Start starts the voice backend.
	Start(ctx context.Context) error

	// Stop stops the voice backend gracefully, completing in-flight conversations.
	Stop(ctx context.Context) error

	// CreateSession creates a new voice session.
	CreateSession(ctx context.Context, config *SessionConfig) (VoiceSession, error)

	// GetSession retrieves a voice session by ID.
	GetSession(ctx context.Context, sessionID string) (VoiceSession, error)

	// ListSessions returns all active voice sessions.
	ListSessions(ctx context.Context) ([]VoiceSession, error)

	// CloseSession closes a voice session.
	CloseSession(ctx context.Context, sessionID string) error

	// HealthCheck checks the health status of the backend.
	HealthCheck(ctx context.Context) (*HealthStatus, error)

	// GetConnectionState returns the current connection state.
	GetConnectionState() ConnectionState

	// GetActiveSessionCount returns the number of active sessions.
	GetActiveSessionCount() int

	// GetConfig returns the backend configuration.
	GetConfig() *Config

	// UpdateConfig updates the backend configuration with validation.
	UpdateConfig(ctx context.Context, config *Config) error
}
