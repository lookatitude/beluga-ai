# LLM vs ChatModel

**LLM** (`llms/iface`): `Invoke(ctx, input any, opts ...core.Option) (any, error)`, `GetModelName()`, `GetProviderName()`. Use for simple prompt→response.

**ChatModel** (`llms/iface`): Embeds `core.Runnable` and `LLM`. Adds: `Generate(ctx, messages, opts) (schema.Message, error)`, `StreamChat(ctx, messages, opts) (<-chan AIMessageChunk, error)`, `BindTools([]core.Tool) ChatModel`, `GetModelName()`, `CheckHealth()`. Use when you need messages, streaming, or tools.

- **Implement only LLM** when: legacy integration, simple models, or providers that don't support streaming/tools.
- **ChatModel always embeds** `core.Runnable`; do not define a ChatModel without it.
- `AIMessageChunk`: `Err`, `AdditionalArgs`, `Content`, `ToolCallChunks`. Stream ends when channel closes or `Err` set.
