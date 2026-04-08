package simulation

import (
	"context"
)

// Observation represents the current state of the environment as perceived
// by the agent or user.
type Observation struct {
	// Text is a textual description of the current state.
	Text string

	// Data holds structured environment data (e.g., form fields, page content).
	Data map[string]any

	// Done indicates whether the episode has ended.
	Done bool
}

// SimEnvironment is the interface for resettable simulation environments.
// Implementations model the world that agents interact with during
// simulation-based testing.
type SimEnvironment interface {
	// Reset initializes or resets the environment to its starting state
	// and returns the initial observation.
	Reset(ctx context.Context) (*Observation, error)

	// Step applies an action to the environment and returns the resulting
	// observation. The action is a free-form string interpreted by the
	// environment.
	Step(ctx context.Context, action string) (*Observation, error)

	// Observe returns the current observation without modifying state.
	Observe(ctx context.Context) (*Observation, error)

	// Close releases any resources held by the environment.
	Close() error
}
