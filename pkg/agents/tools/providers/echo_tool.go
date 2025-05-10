package providers

import (
	"context"

	"github.com/lookatitude/beluga-ai/pkg/agents/tools"
)

// EchoTool is a simple tool that echoes back its input.
type EchoTool struct {
	tools.BaseTool
}

// NewEchoTool creates a new EchoTool.
func NewEchoTool() *EchoTool {
	return &EchoTool{
		BaseTool: tools.BaseTool{
			Name:        "echo",
			Description: "A simple tool that echoes back its input. Useful for testing.",
			InputSchema: `{"type": "string", "description": "The text to echo back."}`,
		},
	}
}

// Execute echoes back the input string.
func (et *EchoTool) Execute(ctx context.Context, input string) (string, error) {
	return input, nil
}

// Ensure EchoTool implements the Tool interface.
var _ tools.Tool = (*EchoTool)(nil)

