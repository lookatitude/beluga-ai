# Session and Config

**VoiceSession:** `Start`, `Stop`, `ProcessAudio`, `SendAudio`, `ReceiveAudio`, `SetAgentCallback`, `SetAgentInstance`, `GetState`, `GetPersistenceStatus`, `UpdateMetadata`, `GetID`. Defined in `backend/iface/session.go`. Sessions are created by `VoiceBackend.CreateSession(ctx, *SessionConfig)`.

**Config (backend):** In `backend/iface/config.go`. `Provider` (required) — backend name for registry lookup. `PipelineType` (`stt_tts` | `s2s`) with `validate:"required,oneof=stt_tts s2s"`. Mode-specific: `STTProvider`/`TTSProvider` with `required_if=PipelineType stt_tts`; `S2SProvider` with `required_if=PipelineType s2s`. Optional: `VADProvider`, `TurnDetectionProvider`, `NoiseCancellationProvider`. Use `mapstructure:"-"` for runtime injects (ChatModel, Memory, Orchestrator, etc.). `ValidateConfig` must enforce pipeline-type rules, `LatencyTarget` (e.g. 100ms–5s), `Timeout` (e.g. 1s–5m).

**SessionConfig:** In `backend/iface/session_config.go`. Per-session: `UserID`, `Transport` (`webrtc`|`websocket`), `ConnectionURL`, `PipelineType`. Exactly one of `AgentInstance` or `AgentCallback` — `ValidateSessionConfig` returns an error if both are nil. Use `mapstructure:"-"` for AgentInstance/AgentCallback. Optional: `Metadata`, `MemoryConfig`, `OrchestrationConfig`, `RAGConfig`.

**Backend resolution:** `NewBackend(ctx, providerName, config)` → `GetRegistry().Create(ctx, name, config)`. Registry: validate config, set `config.Provider = name` when empty, then creator. On unknown `name`, return `ErrCodeProviderNotFound` and wrap a descriptive error (e.g. `"voice backend provider 'x' not found"`). `ErrCodeProviderNotFound` in `backend/errors.go`.
