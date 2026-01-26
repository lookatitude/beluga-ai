# Tool and ToolRegistry

**Tool:** `core.Tool` — `Name()`, `Description()`, `Definition()`, `Execute(ctx, input)`, `Batch(ctx, inputs)`. Defined in `core` for reuse (LLMs, RAG, etc.). `agents/iface` and `agents/tools` re-export `Tool` and `ToolDefinition`.

**BaseTool:** In `agents/tools`. Embed and override `Execute`; `Batch` has a default concurrent implementation. Use for concrete tools.

**ToolRegistry:** Interface in `agents/iface`: `RegisterTool`, `GetTool`, `ListTools`, `GetToolDescriptions`. `InMemoryToolRegistry` in iface; `agents/tools` re-exports `Registry`, `InMemoryToolRegistry`, `NewInMemoryToolRegistry` so the tools subpackage also exposes the registry.

- When building an agent's tool list, reject duplicate tool names (e.g. in ReAct/PlanExecute: `fmt.Errorf("duplicate tool name: %s", toolName)`).
- `agents.NewToolRegistry()` returns `iface.NewInMemoryToolRegistry()`. Prefer the tools subpackage when wiring tool implementations.
