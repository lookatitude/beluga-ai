# Dual Registry (ChatModel and LLM)

**Two registries:** ChatModel: `Register`/`GetProvider`/`ListProviders`/`IsRegistered`. LLM: `RegisterLLM`/`GetLLM`/`ListLLMs`/`IsLLMRegistered`. Separate because of different factory signatures and different consumers.

**Factory types:** `func(*Config) (iface.ChatModel, error)` and `func(*Config) (iface.LLM, error)`.

**Create:** `NewProvider(ctx, name, config, opts)` returns `iface.ChatModel`. Uses `GetRegistry().GetProvider(name, config)`. When `config.Provider` is empty, set `config.Provider = name`. On unknown name, use `ErrCodeUnsupportedProvider` and wrap a descriptive error.

- `ListProviders` and `IsRegistered` for ChatModel; `ListLLMs` and `IsLLMRegistered` for LLM. Same pattern as other provider packages.
- Validate config before calling the factory; on validation failure use `ErrCodeInvalidConfig`.
