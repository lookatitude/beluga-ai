package iface

import (
	"context"
)

// AuthResult represents the result of an authentication operation.
type AuthResult struct {
	UserID     string         `json:"user_id"`
	Authorized bool           `json:"authorized"`
	Metadata   map[string]any `json:"metadata,omitempty"`
}

// AuthHook defines the interface for custom authentication hooks.
// Hooks allow providers to integrate custom authentication logic.
type AuthHook interface {
	// Authenticate authenticates a user based on a token and metadata.
	Authenticate(ctx context.Context, token string, metadata map[string]any) (*AuthResult, error)

	// Authorize checks if a user is authorized to perform an operation.
	Authorize(ctx context.Context, userID string, operation string) (bool, error)
}

// RateLimiter defines the interface for custom rate limiting.
// Hooks allow providers to integrate custom rate limiting logic.
type RateLimiter interface {
	// Allow checks if a request is allowed based on a key.
	Allow(ctx context.Context, key string) (bool, error)

	// Wait waits until a request is allowed based on a key.
	Wait(ctx context.Context, key string) error
}

// DataRetentionHook defines the interface for custom data retention policies.
// Hooks allow providers to integrate custom data retention logic.
type DataRetentionHook interface {
	// ShouldRetain determines if data should be retained based on the session.
	ShouldRetain(ctx context.Context, sessionID string, metadata map[string]any) (bool, error)

	// GetRetentionPeriod returns the retention period for data.
	GetRetentionPeriod(ctx context.Context, sessionID string) (int64, error)
}

// TelephonyHook defines the interface for telephony protocol integration (SIP, etc.).
// Hooks allow providers to integrate custom telephony routing and protocol handling (FR-015, User Story 5 acceptance scenario 2).
type TelephonyHook interface {
	// RouteCall routes an incoming call to the appropriate backend/provider.
	RouteCall(ctx context.Context, phoneNumber string, metadata map[string]any) (string, error) // Returns backend provider name

	// HandleSIP handles SIP protocol-specific operations.
	HandleSIP(ctx context.Context, message []byte, metadata map[string]any) ([]byte, error)

	// GetCallMetadata extracts metadata from a telephony call.
	GetCallMetadata(ctx context.Context, callID string) (map[string]any, error)
}
