package tools

import (
	"context"
	"encoding/json"
	"fmt"
)

// Tool is the interface that tools must implement.
// Tools are functions that agents can call to interact with the world.
type Tool interface {
	// GetName returns the name of the tool.
	GetName() string
	// GetDescription returns a description of the tool.
	// This is used by the agent to decide when to use the tool.
	GetDescription() string
	// Execute runs the tool with the given input.
	// The input is a map of argument names to their values.
	Execute(ctx context.Context, input map[string]interface{}) (string, error)
	// GetInputSchema returns a JSON schema describing the expected input for the tool as a map.
	GetInputSchema() map[string]interface{}
	// GetInputSchemaString returns a JSON string representation of the input schema.
	GetInputSchemaString() (string, error)
}

// ToolAgentAction represents an action an agent should take with a tool.
// It includes the tool to use and the input to provide to the tool.
type ToolAgentAction struct {
	ToolName  string                 `json:"tool_name"`
	ToolInput map[string]interface{} `json:"tool_input"`
	Log       string                 `json:"log"` // Additional log or thought process for this action
}

// BaseTool provides a basic implementation of the Tool interface.
// It can be embedded in specific tool implementations.
type BaseTool struct {
	Name        string
	Description string
	InputSchema map[string]interface{} // Store schema as a parsed map
}

// NewBaseTool creates a new BaseTool.
func NewBaseTool(name, description string, inputSchema map[string]interface{}) *BaseTool {
	return &BaseTool{
		Name:        name,
		Description: description,
		InputSchema: inputSchema,
	}
}

// GetName returns the name of the tool.
func (bt *BaseTool) GetName() string {
	return bt.Name
}

// GetDescription returns the description of the tool.
func (bt *BaseTool) GetDescription() string {
	return bt.Description
}

// GetInputSchema returns the input schema for the tool as a map.
func (bt *BaseTool) GetInputSchema() map[string]interface{} {
	return bt.InputSchema
}

// GetInputSchemaString returns a JSON string representation of the input schema.
func (bt *BaseTool) GetInputSchemaString() (string, error) {
	if bt.InputSchema == nil {
		return "{}", nil // Represent nil schema as empty JSON object string
	}
	schemaBytes, err := json.Marshal(bt.InputSchema)
	if err != nil {
		return "", fmt.Errorf("failed to marshal input schema to string: %w", err)
	}
	return string(schemaBytes), nil
}

// Execute is a placeholder and should be overridden by specific tool implementations.
func (bt *BaseTool) Execute(ctx context.Context, input map[string]interface{}) (string, error) {
	return "", fmt.Errorf("BaseTool Execute method must be overridden by specific tool implementations")
}

