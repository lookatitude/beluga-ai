package providers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/lookatitude/beluga-ai/pkg/agents/tools"
	"github.com/lookatitude/beluga-ai/pkg/config/iface"
)

// EchoTool is a simple tool that echoes back its input.
type EchoTool struct {
	tools.BaseTool
}

// NewEchoTool creates a new EchoTool.
// The config parameter is added to match the expected signature for tool provider constructors,
// though EchoTool itself might not use it directly.
func NewEchoTool(cfg iface.ToolConfig) (*EchoTool, error) {
	// Define the schema as a map
	inputSchema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"input": map[string]any{
				"type":        "string",
				"description": "The text to echo back.",
			},
		},
		"required": []string{"input"},
	}

	tool := &EchoTool{
		BaseTool: tools.BaseTool{},
	}
	tool.SetName(cfg.Name)
	tool.SetDescription(cfg.Description)
	tool.SetInputSchema(inputSchema)
	return tool, nil
}

// Execute echoes back the input, which is expected to be in the input map under the key "input".
func (et *EchoTool) Execute(ctx context.Context, input any) (any, error) {
	inputMap, ok := input.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("input must be a map[string]interface{}, got %T", input)
	}
	inputText, ok := inputMap["input"].(string)
	if !ok {
		// Attempt to marshal the input to string if it's not directly a string, or provide a more informative error.
		inputBytes, err := json.Marshal(input)
		if err != nil {
			return nil, fmt.Errorf("invalid input format for EchoTool: expected a map with a string field 'input', but got something unmarshalable: %v", input)
		}
		return nil, fmt.Errorf("invalid input format for EchoTool: expected a map with a string field 'input', but got %s", string(inputBytes))
	}
	return "Echo: " + inputText, nil
}

// Ensure EchoTool implements the Tool interface.
var _ tools.Tool = (*EchoTool)(nil)
