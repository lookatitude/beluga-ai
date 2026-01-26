# Provider Interface

**Load(configStruct any) error** — Populate the full root config. Use for loading the main `Config` (or equivalent) in one go.

**UnmarshalKey(key string, rawVal any) error** — Decode one key or sub-struct into `rawVal`. Use when you need a single section (e.g. from a composite where only one provider has that key).

**Primitives:** `GetString`, `GetInt`, `GetBool`, `GetFloat64`, `GetStringMapString`, `IsSet` — every provider implements these.

**Domain getters:** Part of the contract. Each provider implements: `GetLLMProviderConfig`, `GetLLMProvidersConfig`, `GetEmbeddingProvidersConfig`, `GetVectorStoresConfig`, `GetAgentConfig`, `GetAgentsConfig`, `GetToolConfig`, `GetToolsConfig` (or the current set used by the app).

**Validate() error** and **SetDefaults() error** — validate and apply defaults after load.
