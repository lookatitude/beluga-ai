package schema

// Contract defines the semantic interface between agents. It specifies what
// an agent expects as input and what it promises to produce as output using
// JSON Schema-compatible definitions.
//
// Contracts live in the schema package as pure data types with zero
// dependencies. Validation and compatibility logic lives in agent/contract/.
type Contract struct {
	// Name is the unique identifier for this contract.
	Name string `json:"name"`

	// Description is a human-readable explanation of the contract's purpose.
	Description string `json:"description,omitempty"`

	// InputSchema is a JSON Schema object describing what the agent expects.
	// A nil schema is treated as a wildcard (any input accepted).
	InputSchema map[string]any `json:"input_schema,omitempty"`

	// OutputSchema is a JSON Schema object describing what the agent produces.
	// A nil schema is treated as a wildcard (any output possible).
	OutputSchema map[string]any `json:"output_schema,omitempty"`

	// Strict indicates whether additional properties should be rejected.
	// When true, both input and output are validated with additionalProperties=false.
	Strict bool `json:"strict,omitempty"`

	// Version is a semver-compatible version string for the contract.
	Version string `json:"version,omitempty"`
}
