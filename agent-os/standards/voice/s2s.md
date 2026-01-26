# S2S (Speech-to-Speech)

**Interfaces:** In `pkg/voice/s2s/iface/` (package-local, not `voice/iface`). `S2SProvider`: `Process(ctx, input *internal.AudioInput, context *internal.ConversationContext, opts ...internal.STSOption) (*internal.AudioOutput, error)`, `Name() string`. Optional `StreamingS2SProvider` embeds `S2SProvider` and adds `StartStreaming(ctx, context, opts) (StreamingSession, error)`. `StreamingSession`: `SendAudio`, `ReceiveAudio` → `<-chan AudioOutputChunk`, `Close`. `AudioOutputChunk`: `Error`, `Audio`, `Timestamp`, `IsFinal`.

**Layout:** Same as STT/TTS/VAD: `s2s.go` (NewProvider, InitMetrics, GetMetrics), `config.go`, `errors.go`, `metrics.go`, `registry.go`, `iface/`, `providers/<name>/`. `internal/` for `AudioInput`, `AudioOutput`, `ConversationContext`, `STSOption`.

**NewProvider:** `NewProvider(ctx, providerName, config, opts) (iface.S2SProvider, error)`. Apply opts, DefaultConfig if nil, Validate, override `config.Provider` from `providerName`, `GetRegistry().GetProvider`; `ErrCodeInvalidConfig` on validation failure; `ErrCodeUnsupportedProvider` when provider not registered. Registry: GetRegistry, Register, GetProvider, ListProviders, IsRegistered.

**InitMetrics / GetMetrics:** Optional; same shape as other voice subpackages.

**Config:** Provider (required, oneof with concrete names or allow dynamic), APIKey (required_unless=Provider mock), plus LatencyTarget, Timeout, SampleRate, Channels, FallbackProviders, etc. Avoid hardcoding `oneof` to a fixed provider list if the registry is extensible; validate provider existence via `IsRegistered` when possible.
