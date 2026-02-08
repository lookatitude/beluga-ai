package schema

// ToolCall represents a request from the model to invoke a tool.
// The Arguments field contains a JSON-encoded string of the tool's input parameters.
type ToolCall struct {
	// ID is the unique identifier for this tool call, used to correlate with ToolResult.
	ID string
	// Name is the name of the tool to invoke.
	Name string
	// Arguments is the JSON-encoded input arguments for the tool.
	Arguments string
}

// ToolResult represents the output from executing a tool call.
type ToolResult struct {
	// CallID is the ID of the ToolCall this result corresponds to.
	CallID string
	// Content holds the result content parts returned by the tool.
	Content []ContentPart
	// IsError indicates whether the tool execution resulted in an error.
	IsError bool
}

// ToolDefinition describes a tool's interface for model consumption.
// It provides the tool's name, description, and input schema so the model
// can determine when and how to call it.
type ToolDefinition struct {
	// Name is the unique name identifying this tool.
	Name string
	// Description explains what the tool does, helping the model decide when to use it.
	Description string
	// InputSchema is a JSON Schema object describing the tool's expected input parameters.
	InputSchema map[string]any
}
