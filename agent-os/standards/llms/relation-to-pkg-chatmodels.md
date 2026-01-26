# Relation to pkg/chatmodels

**llms/iface.ChatModel** is the canonical interface for agents and LLM providers: `Generate`, `StreamChat`, `BindTools`, `GetModelName`, `CheckHealth`; embeds `Runnable` and `LLM`.

**pkg/chatmodels** defines a different `ChatModel`: `GenerateMessages` (returns `[]schema.Message`), `StreamMessages` (returns `<-chan schema.Message`), `GetModelInfo`, `HealthChecker`, `Runnable`. Exists for history and different method shapes.

- **Prefer `llms/iface`** as the default for agents, RAG, and new code. Use `pkg/chatmodels` only when that interface is required or when maintaining chatmodels-based code.
- **Bridging:** Use an adapter (e.g. `ChatModelAdapter` in `llms`) that wraps `iface.LLM` and implements the chatmodels-style `ChatModel` when a consumer expects `pkg/chatmodels.ChatModel`.
