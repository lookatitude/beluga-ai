// Package declarative provides a declarative agent definition system.
// Agents can be defined in JSON and parsed into AgentSpec values. A
// DefaultBuilder then copies the specification fields (provider, model,
// tool names) into an AgentBuild. Registry lookups for concrete LLM
// providers and tools are the responsibility of the caller.
package declarative
