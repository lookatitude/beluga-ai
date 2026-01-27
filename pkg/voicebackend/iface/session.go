package iface

import (
	"context"

	"github.com/lookatitude/beluga-ai/pkg/agents/iface"
)

// VoiceSession defines the interface for voice conversation sessions within a backend.
// A voice session manages a single conversation between a user and an AI agent.
type VoiceSession interface {
	// Start starts the voice session.
	Start(ctx context.Context) error

	// Stop stops the voice session.
	Stop(ctx context.Context) error

	// ProcessAudio processes incoming audio data through the pipeline.
	ProcessAudio(ctx context.Context, audio []byte) error

	// SendAudio sends audio data to the user.
	SendAudio(ctx context.Context, audio []byte) error

	// ReceiveAudio returns a channel for receiving audio from the user.
	ReceiveAudio() <-chan []byte

	// SetAgentCallback sets the agent callback function for processing transcripts.
	SetAgentCallback(callback func(context.Context, string) (string, error)) error

	// SetAgentInstance sets the agent instance for processing transcripts.
	SetAgentInstance(agent iface.Agent) error

	// GetState returns the current pipeline state.
	GetState() PipelineState

	// GetPersistenceStatus returns the persistence status of the session.
	GetPersistenceStatus() PersistenceStatus

	// UpdateMetadata updates the session metadata.
	UpdateMetadata(metadata map[string]any) error

	// GetID returns the session identifier.
	GetID() string
}
