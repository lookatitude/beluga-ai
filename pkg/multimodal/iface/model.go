// Package iface defines interfaces for multimodal model operations.
package iface

import (
	"context"

	"github.com/lookatitude/beluga-ai/pkg/multimodal/types"
)

// MultimodalModel defines the interface for processing multimodal inputs.
// Implementations provide unified access to multimodal model capabilities
// across different providers (OpenAI, Google, Anthropic, etc.).
type MultimodalModel interface {
	// Process processes a multimodal input and returns a multimodal output.
	// This is the primary method for non-streaming multimodal operations.
	Process(ctx context.Context, input *types.MultimodalInput) (*types.MultimodalOutput, error)

	// ProcessStream processes a multimodal input and streams results incrementally.
	// Returns a channel that receives output chunks as they become available.
	// The channel is closed when processing completes or an error occurs.
	ProcessStream(ctx context.Context, input *types.MultimodalInput) (<-chan *types.MultimodalOutput, error)

	// GetCapabilities returns the capabilities of this model for different modalities.
	GetCapabilities(ctx context.Context) (*types.ModalityCapabilities, error)

	// SupportsModality checks if this model supports a specific modality.
	// Modality should be one of: "text", "image", "audio", "video".
	SupportsModality(ctx context.Context, modality string) (bool, error)
}
