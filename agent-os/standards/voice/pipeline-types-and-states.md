# Pipeline Types and States

**PipelineType** (`backend/iface/pipeline.go`): `stt_tts` (STT→Agent→TTS) or `s2s` (speech-to-speech). Use `PipelineTypeSTTTTS`, `PipelineTypeS2S`. Required in Config and SessionConfig; `validate:"required,oneof=stt_tts s2s"`.

**ConnectionState:** Backend-level; from `VoiceBackend.GetConnectionState()`. Values: `disconnected`, `connecting`, `connected`, `reconnecting`, `error`. Reflects transport/connection to the voice service, not per-session pipeline.

**PipelineState:** Session-level; from `VoiceSession.GetState()`. Values: `idle`, `listening`, `processing`, `speaking`, `error`. `idle` — not processing; `listening` — capturing user speech; `processing` — agent/LLM or S2S; `speaking` — playing agent audio. Implementations transition on VAD, turn detection, and pipeline events.

**PersistenceStatus:** From `VoiceSession.GetPersistenceStatus()`. `active` — session should persist; `completed` — ephemeral, can be discarded. Used by backends for retention and cleanup.

**PipelineConfiguration:** Optional struct describing pipeline setup: `Type`, provider names (STT, TTS, S2S, VAD, turn detection, noise), `ProcessingOrder`, `CustomProcessors`, `LatencyTarget`. Use for introspection or provider capabilities, not as the primary config.

**CustomProcessor:** Optional extensibility interface: `Process(ctx, audio, metadata) ([]byte, error)`, `GetName()`, `GetOrder()`. Lower `GetOrder()` runs first. Pass via `Config.CustomProcessors` (`mapstructure:"-"`).
