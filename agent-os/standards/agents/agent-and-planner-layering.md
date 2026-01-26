# Agent and Planner Layering

**Planner:** `Plan(ctx, intermediateSteps, inputs) (AgentAction, AgentFinish, error)`. Core decision-making only.

**Agent:** Embeds `Planner` and adds: `InputVariables`, `OutputVariables`, `GetTools`, `GetConfig`, `GetLLM`, `GetMetrics`. Planning plus tools and config.

**CompositeAgent:** Embeds `core.Runnable`, `Agent`, `LifecycleManager`, `HealthChecker`, `EventEmitter`. Full production shape: run, health, events, lifecycle.

- Split `Planner` from `Agent` for reuse and testing (e.g. mock Planner, swap strategies).
- Factories (`NewBaseAgent`, `NewReActAgent`) and registry `Create` return `CompositeAgent`.
- `AgentAction`: `Tool`, `ToolInput`, `Log`. `AgentFinish`: `ReturnValues`, `Log`. `IntermediateStep`: `Action`, `Observation`.
