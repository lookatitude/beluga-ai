package providers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/lookatitude/beluga-ai/pkg/agents/tools"
	"github.com/lookatitude/beluga-ai/pkg/config" // Added for NewEchoTool signature
)

// EchoTool is a simple tool that echoes back its input.
type EchoTool struct {
	tools.BaseTool
}

// NewEchoTool creates a new EchoTool.
// The config parameter is added to match the expected signature for tool provider constructors,
// though EchoTool itself might not use it directly.
func NewEchoTool(cfg config.ToolConfig) (*EchoTool, error) {
	// Define the schema as a map
	inputSchema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"input": map[string]interface{}{
				"type":        "string",
				"description": "The text to echo back.",
			},
		},
		"required": []string{"input"},
	}

	return &EchoTool{
		BaseTool: tools.BaseTool{
			Name:        cfg.Name, // Use name from config
			Description: cfg.Description, // Use description from config
			InputSchema: inputSchema,
		},
	}, nil
}

// Execute echoes back the input string, which is expected to be in the input map under the key "input".
func (et *EchoTool) Execute(ctx context.Context, input map[string]interface{}) (string, error) {
	inputText, ok := input["input"].(string)
	if !ok {
		// Attempt to marshal the input to string if it's not directly a string, or provide a more informative error.
		inputBytes, err := json.Marshal(input)
		if err != nil {
			return "", fmt.Errorf("invalid input format for EchoTool: expected a map with a string field 'input', but got something unmarshalable: %v", input)
		}
		return "", fmt.Errorf("invalid input format for EchoTool: expected a map with a string field 'input', but got %s", string(inputBytes))
	}
	return fmt.Sprintf("Echo: %s", inputText), nil
}

// Ensure EchoTool implements the Tool interface.
var _ tools.Tool = (*EchoTool)(nil)

