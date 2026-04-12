// Package codeact implements the Code-as-Action (CodeAct) agent pattern.
//
// In the CodeAct pattern, the LLM generates executable code blocks instead of
// tool calls. The agent parses code from the LLM response, executes it via a
// pluggable [CodeExecutor], and feeds the result back as an observation for the
// next reasoning step.
//
// # Architecture
//
// The package provides three main components:
//
//   - [CodeExecutor] interface for executing code (with built-in Noop and Process
//     implementations).
//   - [CodeActPlanner] implementing agent.Planner, registered as "codeact" in the
//     planner registry. Instructs the LLM to emit fenced code blocks and parses
//     them into ActionCode actions.
//   - [CodeActAgent] wrapping agent.BaseAgent with a CodeExecutor, intercepting
//     ActionCode actions and routing them to the executor.
//
// # Usage
//
//	import "github.com/lookatitude/beluga-ai/agent/codeact"
//
//	a := codeact.NewCodeActAgent("solver",
//	    codeact.WithAgentLLM(model),
//	    codeact.WithLanguage("python"),
//	    codeact.WithExecutor(codeact.NewNoopExecutor()),
//	    codeact.WithExecTimeout(30 * time.Second),
//	)
//
//	result, err := a.Invoke(ctx, "Calculate the fibonacci sequence up to 100")
//
// # Execution State
//
// [ExecutionState] tracks variables and outputs across code steps within a
// session, stored in PlannerState.Metadata under the key "codeact_state".
package codeact
