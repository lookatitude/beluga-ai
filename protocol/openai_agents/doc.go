// Package openai_agents provides a compatibility layer between Beluga AI agents
// and the OpenAI Agents SDK format. It converts Beluga agents, tools, and
// handoffs into the OpenAI Agents SDK wire format for interoperability.
//
// This allows Beluga agents to be exposed via an API compatible with the OpenAI
// Agents SDK, enabling clients built for that SDK to interact with Beluga agents.
//
// # Agent Conversion
//
// FromAgent converts a Beluga Agent into an OpenAI Agents SDK AgentDef,
// preserving the agent's identity, tools, and handoff relationships:
//
//	agentDef := openai_agents.FromAgent(belugaAgent)
//	jsonBytes, err := json.Marshal(agentDef)
//
// # Tool Conversion
//
// FromTools converts Beluga tools into OpenAI Agents SDK ToolDefs:
//
//	toolDefs := openai_agents.FromTools(myTools)
//
// # Runner
//
// Runner executes agents using the OpenAI Agents SDK request/response format.
// It maintains a registry of agents and dispatches requests by agent name.
//
//	runner := openai_agents.NewRunner(agentA, agentB)
//
//	resp, err := runner.Run(ctx, openai_agents.RunRequest{
//	    AgentName: "agentA",
//	    Input:     "Hello",
//	})
//
//	agents := runner.ListAgents() // returns AgentDefs for all registered agents
//
// # Key Types
//
//   - AgentDef — agent definition in the OpenAI Agents SDK format
//   - ToolDef / FunctionDef — tool definitions
//   - Handoff — agent handoff descriptor
//   - RunRequest / RunResponse — execution request and response types
//   - Runner — dispatches agent execution requests
//   - ToolCallResult — tool call result in the response
package openai_agents
