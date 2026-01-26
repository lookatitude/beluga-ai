# STT, TTS, VAD, Transport, TurnDetector, Noise

**Interfaces:** Canonical definitions in `pkg/voice/iface/`: `stt.go` (STTProvider, StreamingSession, TranscriptResult), `tts.go` (TTSProvider), `vad.go` (VADProvider, VADResult), `transport.go` (Transport), `turndetection.go` (TurnDetector), `noise.go` (NoiseCancellation). Subpackages `stt`, `tts`, `vad`, `transport`, `turndetection`, `noise` each have `iface/` that **alias** these (e.g. `type STTProvider = voiceiface.STTProvider`) to allow package-specific extensions.

**Subpackage layout:** Each has `{pkg}.go` (NewProvider, InitMetrics, GetMetrics), `config.go`, `errors.go`, `metrics.go`, `registry.go`, `iface/`, `providers/<name>/` (init.go, provider.go, config). Follow backend `registry-shape` and `provider-subpackage-layout` patterns.

**NewProvider:** `NewProvider(ctx, providerName string, config *Config, opts ...ConfigOption) (iface.X, error)`. Apply opts to config; if config is nil use DefaultConfig. Validate (config.Validate); on failure return `NewXxxError("NewProvider", ErrCodeInvalidConfig, err)`. If `providerName != ""` set `config.Provider = providerName`. `GetRegistry().GetProvider(config.Provider, config)`; on unknown provider return `ErrCodeUnsupportedProvider` (or package equivalent) and wrap a descriptive error. Registry: `GetRegistry`, `Register`, `GetProvider` (or `Create`), `ListProviders`, `IsRegistered`.

**InitMetrics / GetMetrics:** Optional. `InitMetrics(meter, tracer)`, `GetMetrics() *Metrics`. Same shape as `backend/metrics-go-shape`.

**Interface summary:** STT: `Transcribe`, `StartStreaming` → StreamingSession. TTS: `GenerateSpeech`, `StreamGenerate` → io.Reader. VAD: `Process`, `ProcessStream` → VADResult. Transport: `SendAudio`, `ReceiveAudio`, `OnAudioReceived`, `Close`. TurnDetector: `DetectTurn`, `DetectTurnWithSilence`. NoiseCancellation: `Process`, `ProcessStream`.
