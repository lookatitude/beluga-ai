# Registry and AgentCreatorFunc

**AgentCreatorFunc:** `func(ctx, agentType, name string, llm any, tools []iface.Tool, config Config) (iface.CompositeAgent, error)`. When the registry uses one signature for multiple agent types, `llm` is `any`; the creator must type-assert to the interface that agent type needs (`llmsiface.LLM` or `llmsiface.ChatModel`) and return `NewAgentErrorWithMessage(..., ErrCodeInitialization, ...)` on mismatch.

**Registry:** `GetRegistry()`, `Register(agentType, creator)`, `Create(...)`, `ListAgentTypes()`, `IsRegistered(agentType)`. On unknown `agentType`, `Create` returns `NewAgentErrorWithMessage(..., ErrCodeInitialization, "agent type 'x' not registered...", err)`.

- Use `ErrCodeInitialization` for: unknown agent type, wrong LLM type (e.g. `ChatModel` for Base or `LLM` for ReAct). Wrap a descriptive error.
- `init()` registers built-in types (e.g. `"base"`, `"react"`). `AgentTypeBase`, `AgentTypeReAct` as constants.
- `IsRegistered(agentType)` required for discovery and validation.
