# Session Mode: STT+TTS vs S2S

**Mutual exclusion:** A session runs in exactly one mode: **STT+TTS** (`stt_tts`) or **S2S** (`s2s`). Selected by `PipelineType` in Config and SessionConfig. Do not allow both modes active at once; there is no "neither" for PipelineType (it is required and oneof).

**STT+TTS mode:** For `PipelineTypeSTTTTS` require `STTProvider` and `TTSProvider` to be non-empty. Optionally use `VADProvider`, `TurnDetectionProvider` (or `turn_detection_provider`), `NoiseCancellationProvider`. `ValidateConfig` must return an error (e.g. `"stt_provider is required for STT_TTS pipeline"`) when either is missing; the caller wraps with `ErrCodeInvalidConfig`.

**S2S mode:** For `PipelineTypeS2S` require `S2SProvider` to be non-empty. STT, TTS, VAD, turn detection, noise are not used for the S2S path. `ValidateConfig` must return an error (e.g. `"s2s_provider is required for S2S pipeline"`) when missing; wrap with `ErrCodeInvalidConfig`.

**Validation layer:** Implement in `ValidateConfig` (backend). Use a `switch config.PipelineType` and check the corresponding provider fields. Use `validate:"required_if=PipelineType stt_tts"` and `required_if=PipelineType s2s` on the provider fields where the validator supports it; otherwise enforce in `ValidateConfig`. On invalid combinations return a clear, descriptive error; the Create/UpdateConfig call path wraps with `NewBackendError(Op, ErrCodeInvalidConfig, err)`.
