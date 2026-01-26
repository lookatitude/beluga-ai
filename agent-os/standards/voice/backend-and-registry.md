# VoiceBackend and Registry

**VoiceBackend:** `Start`, `Stop`, `CreateSession`, `GetSession`, `ListSessions`, `CloseSession`, `HealthCheck`, `GetConnectionState`, `GetActiveSessionCount`, `GetConfig`, `UpdateConfig`.

**HealthCheck** — Check dependencies and return a structured `HealthStatus`. **GetConnectionState** — Return only the local connection state (e.g. disconnected, connecting, connected, reconnecting, error).

**Registry:** `GetRegistry`, `Register`, `Create`, `ListProviders`, `IsRegistered`. Validate config before `Create`. On unknown name, return `ErrCodeProviderNotFound` and wrap a descriptive error. Set `config.Provider` when it is empty.
