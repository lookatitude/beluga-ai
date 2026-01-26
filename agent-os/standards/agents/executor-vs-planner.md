# Executor vs Planner

**Planner:** `Plan(ctx, intermediateSteps, inputs) (AgentAction, AgentFinish, error)` — returns the **next** action or finish. Used in the agent's own loop (e.g. ReAct).

**Executor:** `ExecutePlan(ctx, agent, plan []schema.Step) (schema.FinalAnswer, error)` — runs a **pre-built** plan. Use for reuse of stored/recorded plans and to separate planning from execution.

- **Step handling:** `Action.Tool` non-empty → run via `agent.GetTools()` and `tool.Execute`. `Observation.Output` non-empty → use as observation. `Action.Log` non-empty (and no tool) → `llm.Invoke(ctx, Action.Log)` as a direct LLM step. If none apply, return a generic "no action or observation" message.
- **Options:** `WithMaxIterations`, `WithReturnIntermediateSteps`, `WithHandleParsingErrors`. Enforce `maxIterations` in the step loop; on exceed, return an error.
- `schema.Step`: `Action` (Tool, ToolInput, Log), `Observation` (Output). `FinalAnswer`: `Output`, `IntermediateSteps` (when `returnIntermediateSteps`).
