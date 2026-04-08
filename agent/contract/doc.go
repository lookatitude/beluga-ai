// Package contract provides semantic contract validation and compatibility
// checking for agents. Contracts define what an agent expects as input and
// what it promises to produce as output using JSON Schema-compatible definitions.
//
// The contract type itself lives in schema/ as pure data. This package provides:
//   - Constructors and functional options for building contracts
//   - JSON Schema validation of inputs and outputs
//   - Compatibility checking between agents in a pipeline
//   - Validation middleware for automatic contract enforcement
//   - A registry of reusable contract templates
//
// Usage:
//
//	c := contract.New("summarizer",
//	    contract.WithDescription("Summarizes text input"),
//	    contract.WithInputSchema(map[string]any{"type": "string"}),
//	    contract.WithOutputSchema(map[string]any{"type": "string"}),
//	)
//
//	a := agent.New("summarizer-agent", agent.WithContract(c))
package contract
