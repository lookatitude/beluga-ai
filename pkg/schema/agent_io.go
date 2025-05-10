package schema

// AgentAction represents an action taken by an agent, typically involving a tool.
type AgentAction struct {
	// Tool is the name of the tool to be used for this action.
	Tool string `json:"tool" yaml:"tool"`
	// ToolInput is the input to the tool. It can be a string or a structured map.
	ToolInput interface{} `json:"tool_input" yaml:"tool_input"`
	// Log is a textual representation of the agent's thought process for taking this action.
	Log string `json:"log" yaml:"log"`
}

// AgentObservation represents the output or result obtained from executing an AgentAction.
type AgentObservation struct {
	// ActionLog is the log from the AgentAction that led to this observation.
	// This helps in tracing the sequence of thought, action, and observation.
	ActionLog string `json:"action_log" yaml:"action_log"`
	// Output is the result from the tool execution or LLM response.
	Output string `json:"output" yaml:"output"`
	// ParsedOutput can be a structured representation of the output, if applicable.
	ParsedOutput interface{} `json:"parsed_output,omitempty" yaml:"parsed_output,omitempty"`
}

// Step represents a single step in an agent's execution trace.
// It pairs an action with its corresponding observation.
// This is useful for maintaining the history of agent interactions and for ReAct-style prompting.	ype Step struct {
	Action     AgentAction      `json:"action" yaml:"action"`
	Observation AgentObservation `json:"observation" yaml:"observation"`
}

// FinalAnswer represents the final output of an agent after it has completed its task.
// It might be a direct answer, a summary, or the result of its last action if that concludes its goal.	ype FinalAnswer struct {
	// Output is the final response from the agent.
	Output string `json:"output" yaml:"output"`
	// SourceDocuments can be a list of documents that contributed to the final answer, especially in RAG contexts.
	SourceDocuments []Document `json:"source_documents,omitempty" yaml:"source_documents,omitempty"`
	// IntermediateSteps can include the sequence of actions and observations that led to the final answer,
	// useful for transparency and debugging.
	IntermediateSteps []Step `json:"intermediate_steps,omitempty" yaml:"intermediate_steps,omitempty"`
}

