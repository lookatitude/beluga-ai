package runtime

import (
	"context"

	"github.com/lookatitude/beluga-ai/v2/agent"
	"github.com/lookatitude/beluga-ai/v2/schema"
)

// Plugin intercepts agent execution at the Runner level.
// All methods are called for every turn. Implementations should be lightweight.
type Plugin interface {
	// Name returns a unique identifier for this plugin.
	Name() string

	// BeforeTurn runs before each agent invocation. It may modify the input
	// message before it reaches the agent.
	BeforeTurn(ctx context.Context, session *Session, input schema.Message) (schema.Message, error)

	// AfterTurn runs after each agent invocation. It may modify the events
	// produced by the agent before they are returned to the caller.
	AfterTurn(ctx context.Context, session *Session, events []agent.Event) ([]agent.Event, error)

	// OnError runs when an error occurs during agent execution.
	// It may wrap, suppress, or replace the error.
	OnError(ctx context.Context, err error) error
}

// PluginChain manages an ordered sequence of plugins and provides methods
// to run each hook phase across all plugins in registration order.
type PluginChain struct {
	plugins []Plugin
}

// NewPluginChain creates a PluginChain that will execute the provided
// plugins in the order they are given.
func NewPluginChain(plugins ...Plugin) *PluginChain {
	ps := make([]Plugin, len(plugins))
	copy(ps, plugins)
	return &PluginChain{plugins: ps}
}

// RunBeforeTurn executes each plugin's BeforeTurn in registration order,
// passing the (potentially modified) message from one plugin to the next.
// It stops and returns the first error encountered.
func (c *PluginChain) RunBeforeTurn(ctx context.Context, session *Session, input schema.Message) (schema.Message, error) {
	msg := input
	for _, p := range c.plugins {
		if err := ctx.Err(); err != nil {
			return msg, err
		}
		var err error
		msg, err = p.BeforeTurn(ctx, session, msg)
		if err != nil {
			return msg, err
		}
	}
	return msg, nil
}

// RunAfterTurn executes each plugin's AfterTurn in registration order,
// passing the (potentially modified) event slice from one plugin to the next.
// It stops and returns the first error encountered.
func (c *PluginChain) RunAfterTurn(ctx context.Context, session *Session, events []agent.Event) ([]agent.Event, error) {
	evts := events
	for _, p := range c.plugins {
		if err := ctx.Err(); err != nil {
			return evts, err
		}
		var err error
		evts, err = p.AfterTurn(ctx, session, evts)
		if err != nil {
			return evts, err
		}
	}
	return evts, nil
}

// RunOnError executes each plugin's OnError in registration order,
// replacing the error with whatever each plugin returns. A plugin may
// return nil to suppress the error for subsequent plugins.
func (c *PluginChain) RunOnError(ctx context.Context, err error) error {
	cur := err
	for _, p := range c.plugins {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
		cur = p.OnError(ctx, cur)
	}
	return cur
}
