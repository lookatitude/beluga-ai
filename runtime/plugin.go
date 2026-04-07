// Package runtime provides the core runtime abstractions for agent execution,
// including the Plugin interface for cross-cutting concerns such as auditing,
// cost tracking, rate limiting, and retry logic.
package runtime

import (
	"context"

	"github.com/lookatitude/beluga-ai/agent"
	"github.com/lookatitude/beluga-ai/schema"
)

// Plugin is the extension interface for cross-cutting concerns applied to
// every agent turn. Plugins are invoked before and after each turn as well
// as on error, allowing them to observe, modify, or abort execution.
//
// All methods receive the active context and the current session. Context
// cancellation must be respected by all implementations.
type Plugin interface {
	// Name returns a unique identifier for this plugin.
	Name() string

	// BeforeTurn is called before the agent processes the input message.
	// Implementations may modify the message (e.g., add metadata) or return
	// an error to abort the turn (e.g., rate limit exceeded).
	BeforeTurn(ctx context.Context, session *Session, input schema.Message) (schema.Message, error)

	// AfterTurn is called after the agent produces its events.
	// Implementations may observe or modify the event stream (e.g., record
	// usage, enrich events with cost information).
	AfterTurn(ctx context.Context, session *Session, events []agent.Event) ([]agent.Event, error)

	// OnError is called when an error occurs during a turn.
	// Implementations may decide to suppress, wrap, or escalate the error.
	// Returning nil suppresses the error; returning a new error replaces it.
	OnError(ctx context.Context, err error) error
}
