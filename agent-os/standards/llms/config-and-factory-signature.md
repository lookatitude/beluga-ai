# Config and Factory Signature

**Config:** `Provider`, `ModelName`, `APIKey`, `Temperature`, `MaxTokens`, `TopP`, `Timeout`, `StopSequences`, `ProviderSpecific`, etc. Use `ConfigOption` and `Validate` before create.

**Factory:** `func(*Config) (iface.ChatModel, error)` or `func(*Config) (iface.LLM, error)`. No `ctx` in the factory; use `*Config` only. Keep context in the create path (e.g. `NewProvider(ctx, ...)`) and make it optional where the factory does not need it, to ease implementation.

**Provider in config:** When `config.Provider` is empty, set `config.Provider = name` before calling the factory. When `config.Provider` is already set, do not override; `name` is for registry lookup only.

- Validate config before calling the factory; on failure use `ErrCodeInvalidConfig`.
- `NewProvider(ctx, name, config, opts)`: apply opts, then validate, then `GetProvider`/`GetLLM`.
