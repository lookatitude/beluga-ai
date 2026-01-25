# Voice Provider Integration Guide

> **Learn how to integrate custom voice providers (STT, TTS, S2S) with Beluga AI's voice agent system.**

## Introduction

Voice agents in Beluga AI are powered by three types of providers that work together to enable natural speech interactions:

- **STT (Speech-to-Text)**: Converts audio input to text
- **TTS (Text-to-Speech)**: Converts text responses to audio
- **S2S (Speech-to-Speech)**: End-to-end speech conversations without intermediate text

Each provider type follows the same extensibility patterns as the rest of Beluga AI - implement the interface, register with the global registry, and your provider is ready to use.

In this guide, you'll learn:

- How voice providers fit into the voice agent architecture
- How to implement custom STT, TTS, and S2S providers
- How to add OTEL instrumentation for audio latency tracking
- How to integrate with the session management system
- How to handle streaming audio properly

## Prerequisites

| Requirement | Why You Need It |
|-------------|-----------------|
| **Go 1.24+** | Required for Beluga AI framework |
| **Understanding of audio formats** | PCM, sample rates, channels |
| **Provider API credentials** | For the voice service you're integrating |

## Voice Agent Architecture

Before diving into provider implementation, let's understand how voice providers fit into the bigger picture:

```
┌─────────────────────────────────────────────────────────────────┐
│                        Voice Session                             │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐         │
│  │    VAD      │    │    Turn     │    │   Noise     │         │
│  │  Detection  │───▶│  Detection  │───▶│ Cancellation│         │
│  └─────────────┘    └─────────────┘    └─────────────┘         │
│         │                                     │                  │
│         ▼                                     ▼                  │
│  ┌─────────────────────────────────────────────────────┐        │
│  │              Option A: STT + Agent + TTS            │        │
│  │  ┌───────┐    ┌───────────┐    ┌───────┐           │        │
│  │  │  STT  │───▶│   Agent   │───▶│  TTS  │           │        │
│  │  └───────┘    └───────────┘    └───────┘           │        │
│  └─────────────────────────────────────────────────────┘        │
│                          OR                                      │
│  ┌─────────────────────────────────────────────────────┐        │
│  │              Option B: S2S (End-to-End)             │        │
│  │  ┌───────────────────────────────────────────┐     │        │
│  │  │         Speech-to-Speech Provider          │     │        │
│  │  │    (includes reasoning/conversation)       │     │        │
│  │  └───────────────────────────────────────────┘     │        │
│  └─────────────────────────────────────────────────────┘        │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

## STT Provider Integration

Speech-to-Text providers convert audio input to text transcriptions.

### STT Configuration

Before implementing your STT provider, let's understand the available configuration options:

```go
// STT Configuration with all available options
type STTConfig struct {
    // Provider specifies which STT service to use
    Provider string `mapstructure:"provider" validate:"required"`
    
    // APIKey for authentication
    APIKey string `mapstructure:"api_key" validate:"required"`
    
    // Model specifies the transcription model (e.g., "whisper-1", "latest_short")
    Model string `mapstructure:"model" default:"default"`
    
    // Language hint for transcription (e.g., "en-US", "es-ES")
    Language string `mapstructure:"language" default:"en-US"`
    
    // SampleRate of input audio in Hz (8000, 16000, 24000, 48000)
    SampleRate int `mapstructure:"sample_rate" default:"16000" validate:"oneof=8000 16000 24000 48000"`
    
    // Channels: 1 for mono (recommended), 2 for stereo
    Channels int `mapstructure:"channels" default:"1" validate:"oneof=1 2"`
    
    // EnablePunctuation adds punctuation to transcriptions
    EnablePunctuation bool `mapstructure:"enable_punctuation" default:"true"`
    
    // EnableWordTimestamps includes word-level timing
    EnableWordTimestamps bool `mapstructure:"enable_word_timestamps" default:"false"`
    
    // MaxAlternatives specifies how many transcription alternatives to return
    MaxAlternatives int `mapstructure:"max_alternatives" default:"1" validate:"min=1,max=10"`
    
    // Timeout for individual transcription requests
    Timeout time.Duration `mapstructure:"timeout" default:"30s"`
    
    // StreamingChunkDuration for real-time streaming (in milliseconds)
    StreamingChunkDuration int `mapstructure:"streaming_chunk_duration" default:"100"`
    
    // Profanity filter
    FilterProfanity bool `mapstructure:"filter_profanity" default:"false"`
    
    // Custom vocabulary for domain-specific terms
    CustomVocabulary []string `mapstructure:"custom_vocabulary"`
}
```

### Creating STT Providers with Functional Options

Use functional options to configure your STT provider at runtime:

```go
// Option functions for STT configuration
type STTOption func(*STTConfig)

// WithSTTModel sets the transcription model
func WithSTTModel(model string) STTOption {
    return func(c *STTConfig) {
        c.Model = model
    }
}

// WithSTTLanguage sets the language hint
func WithSTTLanguage(language string) STTOption {
    return func(c *STTConfig) {
        c.Language = language
    }
}

// WithSTTSampleRate sets the input audio sample rate
func WithSTTSampleRate(rate int) STTOption {
    return func(c *STTConfig) {
        c.SampleRate = rate
    }
}

// WithSTTPunctuation enables/disables punctuation
func WithSTTPunctuation(enabled bool) STTOption {
    return func(c *STTConfig) {
        c.EnablePunctuation = enabled
    }
}

// WithSTTWordTimestamps enables word-level timing
func WithSTTWordTimestamps(enabled bool) STTOption {
    return func(c *STTConfig) {
        c.EnableWordTimestamps = enabled
    }
}

// WithSTTTimeout sets the request timeout
func WithSTTTimeout(timeout time.Duration) STTOption {
    return func(c *STTConfig) {
        c.Timeout = timeout
    }
}

// WithSTTCustomVocabulary adds domain-specific terms
func WithSTTCustomVocabulary(terms []string) STTOption {
    return func(c *STTConfig) {
        c.CustomVocabulary = append(c.CustomVocabulary, terms...)
    }
}

// Example usage
provider, err := stt.NewProvider(ctx, "deepgram",
    stt.WithSTTModel("nova-2"),
    stt.WithSTTLanguage("en-US"),
    stt.WithSTTSampleRate(16000),
    stt.WithSTTPunctuation(true),
    stt.WithSTTWordTimestamps(true),
    stt.WithSTTCustomVocabulary([]string{"Beluga", "OTEL", "microservice"}),
)
```

### The STT Interface

```go
type STTProvider interface {
    // Transcribe converts audio to text
    Transcribe(ctx context.Context, audio []byte) (string, error)
    
    // StartStreaming begins a streaming transcription session
    StartStreaming(ctx context.Context) (StreamingSTTSession, error)
    
    // GetName returns the provider identifier
    GetName() string
    
    // GetSupportedFormats returns supported audio formats
    GetSupportedFormats() []AudioFormat
}

type StreamingSTTSession interface {
    // SendAudio sends audio data for transcription
    SendAudio(ctx context.Context, audio []byte) error
    
    // ReceiveTranscript returns a channel of transcription results
    ReceiveTranscript() <-chan TranscriptResult
    
    // Close ends the streaming session
    Close() error
}

type TranscriptResult struct {
    Text    string
    IsFinal bool
    Error   error
}
```

### Implementing a Custom STT Provider

#### Step 1: Create the Provider Structure

```go
// pkg/voice/stt/providers/myservice/provider.go
package myservice

import (
    "context"
    "sync"
    "time"

    "github.com/lookatitude/beluga-ai/pkg/voice/stt"
    "github.com/lookatitude/beluga-ai/pkg/voice/stt/iface"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/trace"
)

const ProviderName = "myservice"

type MyServiceSTT struct {
    config     *stt.Config
    client     *myapi.Client
    metrics    *stt.Metrics
    mu         sync.RWMutex
}

func NewMyServiceSTT(ctx context.Context, config stt.Config) (*MyServiceSTT, error) {
    // Validate configuration
    if config.APIKey == "" {
        return nil, stt.NewSTTError("NewMyServiceSTT", stt.ErrCodeInvalidConfig,
            errors.New("API key is required"))
    }

    client := myapi.NewClient(config.APIKey)
    
    return &MyServiceSTT{
        config:  &config,
        client:  client,
        metrics: stt.GetGlobalMetrics(),
    }, nil
}
```

#### Step 2: Implement the Transcribe Method

```go
func (s *MyServiceSTT) Transcribe(ctx context.Context, audio []byte) (string, error) {
    // Start OTEL tracing
    tracer := otel.Tracer("myservice-stt")
    ctx, span := tracer.Start(ctx, "myservice.Transcribe",
        trace.WithAttributes(
            attribute.String("provider", ProviderName),
            attribute.Int("audio_size", len(audio)),
            attribute.Int("sample_rate", s.config.SampleRate),
        ),
    )
    defer span.End()

    start := time.Now()

    // Record active request
    s.metrics.IncrementActiveRequests(ctx, ProviderName)
    defer s.metrics.DecrementActiveRequests(ctx, ProviderName)

    // Call your API
    response, err := s.client.Transcribe(ctx, audio, myapi.TranscribeOptions{
        SampleRate: s.config.SampleRate,
        Language:   s.config.Language,
        Model:      s.config.Model,
    })
    
    if err != nil {
        duration := time.Since(start)
        s.metrics.RecordError(ctx, ProviderName, s.config.Model, getErrorCode(err), duration)
        span.RecordError(err)
        return "", s.handleError("Transcribe", err)
    }

    duration := time.Since(start)
    s.metrics.RecordTranscription(ctx, ProviderName, s.config.Model, duration, len(response.Text))

    span.SetAttributes(
        attribute.String("transcript_length", fmt.Sprintf("%d", len(response.Text))),
        attribute.Float64("latency_ms", float64(duration.Milliseconds())),
    )

    return response.Text, nil
}
```

#### Step 3: Implement Streaming Support

```go
func (s *MyServiceSTT) StartStreaming(ctx context.Context) (iface.StreamingSTTSession, error) {
    tracer := otel.Tracer("myservice-stt")
    ctx, span := tracer.Start(ctx, "myservice.StartStreaming")
    
    stream, err := s.client.CreateStream(ctx, myapi.StreamOptions{
        SampleRate: s.config.SampleRate,
        Language:   s.config.Language,
    })
    if err != nil {
        span.RecordError(err)
        span.End()
        return nil, s.handleError("StartStreaming", err)
    }

    session := &myStreamingSession{
        stream:     stream,
        config:     s.config,
        metrics:    s.metrics,
        span:       span,
        resultChan: make(chan iface.TranscriptResult, 100),
    }

    // Start receiving transcripts in background
    go session.receiveLoop(ctx)

    return session, nil
}

type myStreamingSession struct {
    stream     *myapi.Stream
    config     *stt.Config
    metrics    *stt.Metrics
    span       trace.Span
    resultChan chan iface.TranscriptResult
    closed     bool
    mu         sync.Mutex
}

func (s *myStreamingSession) SendAudio(ctx context.Context, audio []byte) error {
    s.mu.Lock()
    defer s.mu.Unlock()
    
    if s.closed {
        return stt.NewSTTError("SendAudio", stt.ErrCodeStreamClosed,
            errors.New("stream is closed"))
    }

    return s.stream.SendAudio(ctx, audio)
}

func (s *myStreamingSession) ReceiveTranscript() <-chan iface.TranscriptResult {
    return s.resultChan
}

func (s *myStreamingSession) receiveLoop(ctx context.Context) {
    defer close(s.resultChan)
    defer s.span.End()

    for {
        select {
        case <-ctx.Done():
            return
        default:
            result, err := s.stream.Receive()
            if err != nil {
                if err != io.EOF {
                    s.resultChan <- iface.TranscriptResult{Error: err}
                }
                return
            }

            s.resultChan <- iface.TranscriptResult{
                Text:    result.Text,
                IsFinal: result.IsFinal,
            }
        }
    }
}

func (s *myStreamingSession) Close() error {
    s.mu.Lock()
    defer s.mu.Unlock()
    
    s.closed = true
    return s.stream.Close()
}
```

#### Step 4: Register the Provider

```go
// pkg/voice/stt/providers/myservice/init.go
package myservice

import (
    "context"
    "github.com/lookatitude/beluga-ai/pkg/voice/stt"
    "github.com/lookatitude/beluga-ai/pkg/voice/stt/iface"
)

func init() {
    stt.GetRegistry().Register(ProviderName,
        func(ctx context.Context, config *stt.Config) (iface.STTProvider, error) {
            return NewMyServiceSTT(ctx, *config)
        },
    )
}
```

## TTS Provider Integration

Text-to-Speech providers convert text to audio output.

### TTS Configuration

Here's the complete configuration for TTS providers:

```go
// TTS Configuration with all available options
type TTSConfig struct {
    // Provider specifies which TTS service to use
    Provider string `mapstructure:"provider" validate:"required"`
    
    // APIKey for authentication
    APIKey string `mapstructure:"api_key" validate:"required"`
    
    // Voice ID or name (e.g., "alloy", "echo", "shimmer")
    Voice string `mapstructure:"voice" default:"alloy"`
    
    // Model specifies the TTS model (e.g., "tts-1", "tts-1-hd")
    Model string `mapstructure:"model" default:"default"`
    
    // Speed of speech (0.25 to 4.0, where 1.0 is normal)
    Speed float64 `mapstructure:"speed" default:"1.0" validate:"min=0.25,max=4.0"`
    
    // Pitch adjustment (-20.0 to 20.0 semitones)
    Pitch float64 `mapstructure:"pitch" default:"0.0" validate:"min=-20.0,max=20.0"`
    
    // SampleRate of output audio (typically 22050 or 24000)
    SampleRate int `mapstructure:"sample_rate" default:"24000"`
    
    // OutputFormat: "mp3", "opus", "aac", "flac", "wav", "pcm"
    OutputFormat string `mapstructure:"output_format" default:"mp3" validate:"oneof=mp3 opus aac flac wav pcm"`
    
    // Timeout for generation requests
    Timeout time.Duration `mapstructure:"timeout" default:"60s"`
    
    // EnableSSML allows SSML markup for advanced control
    EnableSSML bool `mapstructure:"enable_ssml" default:"false"`
    
    // Stability controls voice consistency (0.0 to 1.0)
    Stability float64 `mapstructure:"stability" default:"0.5"`
    
    // SimilarityBoost enhances voice similarity (0.0 to 1.0)
    SimilarityBoost float64 `mapstructure:"similarity_boost" default:"0.75"`
    
    // StreamingBufferSize for real-time audio streaming
    StreamingBufferSize int `mapstructure:"streaming_buffer_size" default:"4096"`
}
```

### Creating TTS Providers with Functional Options

```go
// Option functions for TTS configuration
type TTSOption func(*TTSConfig)

// WithTTSVoice sets the voice to use
func WithTTSVoice(voice string) TTSOption {
    return func(c *TTSConfig) {
        c.Voice = voice
    }
}

// WithTTSModel sets the TTS model
func WithTTSModel(model string) TTSOption {
    return func(c *TTSConfig) {
        c.Model = model
    }
}

// WithTTSSpeed sets the speech speed
func WithTTSSpeed(speed float64) TTSOption {
    return func(c *TTSConfig) {
        c.Speed = speed
    }
}

// WithTTSPitch sets the pitch adjustment
func WithTTSPitch(pitch float64) TTSOption {
    return func(c *TTSConfig) {
        c.Pitch = pitch
    }
}

// WithTTSOutputFormat sets the audio output format
func WithTTSOutputFormat(format string) TTSOption {
    return func(c *TTSConfig) {
        c.OutputFormat = format
    }
}

// WithTTSSampleRate sets the output sample rate
func WithTTSSampleRate(rate int) TTSOption {
    return func(c *TTSConfig) {
        c.SampleRate = rate
    }
}

// WithTTSSSML enables SSML support
func WithTTSSSML(enabled bool) TTSOption {
    return func(c *TTSConfig) {
        c.EnableSSML = enabled
    }
}

// WithTTSStability sets voice stability (for neural voices)
func WithTTSStability(stability float64) TTSOption {
    return func(c *TTSConfig) {
        c.Stability = stability
    }
}

// Example usage
provider, err := tts.NewProvider(ctx, "elevenlabs",
    tts.WithTTSVoice("rachel"),
    tts.WithTTSModel("eleven_multilingual_v2"),
    tts.WithTTSSpeed(1.1),
    tts.WithTTSOutputFormat("pcm"),
    tts.WithTTSSampleRate(24000),
    tts.WithTTSStability(0.7),
)
```

### The TTS Interface

```go
type TTSProvider interface {
    // GenerateSpeech converts text to audio
    GenerateSpeech(ctx context.Context, text string) ([]byte, error)
    
    // StreamGenerate returns a reader for streaming audio output
    StreamGenerate(ctx context.Context, text string) (io.Reader, error)
    
    // GetName returns the provider identifier
    GetName() string
    
    // GetVoices returns available voices
    GetVoices() []Voice
}

type Voice struct {
    ID          string
    Name        string
    Language    string
    Gender      string
    Description string
}
```

### Implementing a Custom TTS Provider

```go
package myservice

import (
    "context"
    "io"
    "time"

    "github.com/lookatitude/beluga-ai/pkg/voice/tts"
    "github.com/lookatitude/beluga-ai/pkg/voice/tts/iface"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
)

const ProviderName = "myservice"

type MyServiceTTS struct {
    config  *tts.Config
    client  *myapi.Client
    metrics *tts.Metrics
}

func NewMyServiceTTS(ctx context.Context, config tts.Config) (*MyServiceTTS, error) {
    if config.APIKey == "" {
        return nil, tts.NewTTSError("NewMyServiceTTS", tts.ErrCodeInvalidConfig,
            errors.New("API key is required"))
    }

    return &MyServiceTTS{
        config:  &config,
        client:  myapi.NewClient(config.APIKey),
        metrics: tts.GetGlobalMetrics(),
    }, nil
}

func (t *MyServiceTTS) GenerateSpeech(ctx context.Context, text string) ([]byte, error) {
    tracer := otel.Tracer("myservice-tts")
    ctx, span := tracer.Start(ctx, "myservice.GenerateSpeech",
        trace.WithAttributes(
            attribute.String("provider", ProviderName),
            attribute.Int("text_length", len(text)),
            attribute.String("voice", t.config.Voice),
        ),
    )
    defer span.End()

    start := time.Now()

    // Record active request
    t.metrics.IncrementActiveRequests(ctx, ProviderName)
    defer t.metrics.DecrementActiveRequests(ctx, ProviderName)

    // Generate speech
    audio, err := t.client.Synthesize(ctx, myapi.SynthesizeRequest{
        Text:       text,
        Voice:      t.config.Voice,
        Speed:      t.config.Speed,
        SampleRate: t.config.SampleRate,
    })
    
    if err != nil {
        duration := time.Since(start)
        t.metrics.RecordError(ctx, ProviderName, t.config.Voice, getErrorCode(err), duration)
        span.RecordError(err)
        return nil, t.handleError("GenerateSpeech", err)
    }

    duration := time.Since(start)
    t.metrics.RecordGeneration(ctx, ProviderName, t.config.Voice, duration, len(audio))

    span.SetAttributes(
        attribute.Int("audio_size", len(audio)),
        attribute.Float64("latency_ms", float64(duration.Milliseconds())),
    )

    return audio, nil
}

func (t *MyServiceTTS) StreamGenerate(ctx context.Context, text string) (io.Reader, error) {
    tracer := otel.Tracer("myservice-tts")
    ctx, span := tracer.Start(ctx, "myservice.StreamGenerate")

    stream, err := t.client.StreamSynthesize(ctx, myapi.SynthesizeRequest{
        Text:       text,
        Voice:      t.config.Voice,
        Speed:      t.config.Speed,
        SampleRate: t.config.SampleRate,
    })
    if err != nil {
        span.RecordError(err)
        span.End()
        return nil, t.handleError("StreamGenerate", err)
    }

    return &streamingReader{
        stream: stream,
        span:   span,
    }, nil
}

func (t *MyServiceTTS) GetName() string {
    return ProviderName
}

func (t *MyServiceTTS) GetVoices() []iface.Voice {
    // Return available voices from your service
    return []iface.Voice{
        {ID: "voice-1", Name: "Alice", Language: "en-US", Gender: "female"},
        {ID: "voice-2", Name: "Bob", Language: "en-US", Gender: "male"},
    }
}
```

## S2S Provider Integration

Speech-to-Speech providers handle end-to-end voice conversations without intermediate text processing. These are particularly useful for low-latency conversational AI applications.

### S2S Configuration

```go
// S2S Configuration with all available options
type S2SConfig struct {
    // Provider specifies which S2S service to use
    Provider string `mapstructure:"provider" validate:"required"`
    
    // APIKey for authentication
    APIKey string `mapstructure:"api_key" validate:"required"`
    
    // Model specifies the S2S model
    Model string `mapstructure:"model" default:"default"`
    
    // Voice ID for the response voice
    Voice string `mapstructure:"voice" default:"alloy"`
    
    // SystemPrompt sets the assistant's personality/instructions
    SystemPrompt string `mapstructure:"system_prompt"`
    
    // InputSampleRate for incoming audio (Hz)
    InputSampleRate int `mapstructure:"input_sample_rate" default:"24000"`
    
    // OutputSampleRate for outgoing audio (Hz)
    OutputSampleRate int `mapstructure:"output_sample_rate" default:"24000"`
    
    // InputFormat: "pcm16", "g711_ulaw", "g711_alaw"
    InputFormat string `mapstructure:"input_format" default:"pcm16"`
    
    // OutputFormat: "pcm16", "g711_ulaw", "g711_alaw"
    OutputFormat string `mapstructure:"output_format" default:"pcm16"`
    
    // Modalities: "text", "audio", or "text+audio"
    Modalities []string `mapstructure:"modalities" default:"[\"text\", \"audio\"]"`
    
    // TurnDetection configuration
    TurnDetection TurnDetectionConfig `mapstructure:"turn_detection"`
    
    // Temperature for response generation (0.0 to 2.0)
    Temperature float64 `mapstructure:"temperature" default:"0.8"`
    
    // MaxResponseTokens limits response length
    MaxResponseTokens int `mapstructure:"max_response_tokens" default:"4096"`
    
    // Timeout for the session
    Timeout time.Duration `mapstructure:"timeout" default:"300s"`
    
    // EnableFunctionCalling allows the S2S model to call tools
    EnableFunctionCalling bool `mapstructure:"enable_function_calling" default:"false"`
    
    // Tools available for the S2S model
    Tools []Tool `mapstructure:"tools"`
}

type TurnDetectionConfig struct {
    // Type: "server_vad" or "none"
    Type string `mapstructure:"type" default:"server_vad"`
    
    // Threshold for speech detection (0.0 to 1.0)
    Threshold float64 `mapstructure:"threshold" default:"0.5"`
    
    // PrefixPaddingMs: audio to keep before speech starts
    PrefixPaddingMs int `mapstructure:"prefix_padding_ms" default:"300"`
    
    // SilenceDurationMs: silence needed to end a turn
    SilenceDurationMs int `mapstructure:"silence_duration_ms" default:"500"`
    
    // CreateResponse: automatically generate response when turn ends
    CreateResponse bool `mapstructure:"create_response" default:"true"`
}
```

### Creating S2S Providers with Functional Options

```go
// Option functions for S2S configuration
type S2SOption func(*S2SConfig)

// WithS2SModel sets the S2S model
func WithS2SModel(model string) S2SOption {
    return func(c *S2SConfig) {
        c.Model = model
    }
}

// WithS2SVoice sets the response voice
func WithS2SVoice(voice string) S2SOption {
    return func(c *S2SConfig) {
        c.Voice = voice
    }
}

// WithS2SSystemPrompt sets the system instructions
func WithS2SSystemPrompt(prompt string) S2SOption {
    return func(c *S2SConfig) {
        c.SystemPrompt = prompt
    }
}

// WithS2STurnDetection configures VAD-based turn detection
func WithS2STurnDetection(threshold float64, silenceMs int) S2SOption {
    return func(c *S2SConfig) {
        c.TurnDetection.Type = "server_vad"
        c.TurnDetection.Threshold = threshold
        c.TurnDetection.SilenceDurationMs = silenceMs
    }
}

// WithS2STemperature sets response randomness
func WithS2STemperature(temp float64) S2SOption {
    return func(c *S2SConfig) {
        c.Temperature = temp
    }
}

// WithS2SModalities sets which modalities to use
func WithS2SModalities(modalities ...string) S2SOption {
    return func(c *S2SConfig) {
        c.Modalities = modalities
    }
}

// WithS2STools enables function calling with tools
func WithS2STools(tools ...Tool) S2SOption {
    return func(c *S2SConfig) {
        c.EnableFunctionCalling = true
        c.Tools = tools
    }
}

// Example usage
provider, err := s2s.NewProvider(ctx, "openai-realtime",
    s2s.WithS2SModel("gpt-4o-realtime-preview"),
    s2s.WithS2SVoice("alloy"),
    s2s.WithS2SSystemPrompt("You are a helpful customer service agent."),
    s2s.WithS2STurnDetection(0.5, 500),
    s2s.WithS2STemperature(0.7),
    s2s.WithS2SModalities("text", "audio"),
    s2s.WithS2STools(weatherTool, searchTool),
)
```

### The S2S Interface

```go
type S2SProvider interface {
    // ProcessConversation handles a complete speech-to-speech turn
    ProcessConversation(ctx context.Context, audio []byte) (*AudioOutput, error)
    
    // GetName returns the provider identifier
    GetName() string
    
    // SupportsStreaming indicates if streaming is available
    SupportsStreaming() bool
}

type StreamingS2SProvider interface {
    S2SProvider
    
    // StartStreaming begins a bidirectional streaming session
    StartStreaming(ctx context.Context, convCtx *ConversationContext) (S2SStreamingSession, error)
}

type S2SStreamingSession interface {
    // SendAudio sends audio input
    SendAudio(ctx context.Context, audio []byte) error
    
    // ReceiveAudio returns a channel of audio output chunks
    ReceiveAudio() <-chan AudioOutputChunk
    
    // Close ends the session
    Close() error
}

type AudioOutput struct {
    Data      []byte
    Format    AudioFormat
    Timestamp time.Time
    Latency   time.Duration
}

type AudioOutputChunk struct {
    Audio   []byte
    IsFinal bool
    Error   error
}
```

### Implementing a Custom S2S Provider

```go
package myservice

import (
    "context"
    "time"

    "github.com/lookatitude/beluga-ai/pkg/voice/s2s"
    "github.com/lookatitude/beluga-ai/pkg/voice/s2s/iface"
    "github.com/lookatitude/beluga-ai/pkg/voice/s2s/internal"
    "go.opentelemetry.io/otel"
)

const ProviderName = "myservice-s2s"

type MyServiceS2S struct {
    config  *s2s.Config
    client  *myapi.Client
    metrics *s2s.Metrics
}

func NewMyServiceS2S(ctx context.Context, config s2s.Config) (*MyServiceS2S, error) {
    return &MyServiceS2S{
        config:  &config,
        client:  myapi.NewClient(config.APIKey),
        metrics: s2s.GetGlobalMetrics(),
    }, nil
}

func (s *MyServiceS2S) ProcessConversation(ctx context.Context, audio []byte) (*iface.AudioOutput, error) {
    tracer := otel.Tracer("myservice-s2s")
    ctx, span := tracer.Start(ctx, "myservice.ProcessConversation")
    defer span.End()

    start := time.Now()

    // Process through your S2S API
    response, err := s.client.Converse(ctx, audio)
    if err != nil {
        span.RecordError(err)
        return nil, err
    }

    latency := time.Since(start)

    return &iface.AudioOutput{
        Data:      response.Audio,
        Format:    iface.AudioFormat{SampleRate: 24000, Channels: 1},
        Timestamp: time.Now(),
        Latency:   latency,
    }, nil
}

func (s *MyServiceS2S) StartStreaming(ctx context.Context, convCtx *internal.ConversationContext) (iface.S2SStreamingSession, error) {
    stream, err := s.client.CreateConversationStream(ctx, convCtx.ConversationID)
    if err != nil {
        return nil, err
    }

    return &myS2SSession{
        stream:  stream,
        config:  s.config,
        audioCh: make(chan iface.AudioOutputChunk, 100),
    }, nil
}

func (s *MyServiceS2S) GetName() string {
    return ProviderName
}

func (s *MyServiceS2S) SupportsStreaming() bool {
    return true
}
```

## Session Integration

Voice providers integrate with the session management system to enable stateful, long-running voice conversations. The session manager handles connection lifecycle, audio routing, turn coordination, and error recovery.

### Session Configuration

```go
// SessionConfig contains all session-related settings
type SessionConfig struct {
    // SessionID unique identifier (auto-generated if not provided)
    SessionID string `mapstructure:"session_id"`
    
    // Mode: "stt_tts" for pipeline or "s2s" for end-to-end
    Mode string `mapstructure:"mode" default:"stt_tts" validate:"oneof=stt_tts s2s"`
    
    // Transport: "websocket" or "webrtc"
    Transport string `mapstructure:"transport" default:"websocket" validate:"oneof=websocket webrtc"`
    
    // MaxDuration limits session length (0 = unlimited)
    MaxDuration time.Duration `mapstructure:"max_duration" default:"30m"`
    
    // IdleTimeout ends session after inactivity
    IdleTimeout time.Duration `mapstructure:"idle_timeout" default:"5m"`
    
    // EnableHistory maintains conversation history
    EnableHistory bool `mapstructure:"enable_history" default:"true"`
    
    // MaxHistoryTurns limits history size
    MaxHistoryTurns int `mapstructure:"max_history_turns" default:"50"`
    
    // ReconnectAttempts before giving up
    ReconnectAttempts int `mapstructure:"reconnect_attempts" default:"3"`
    
    // ReconnectDelay between attempts
    ReconnectDelay time.Duration `mapstructure:"reconnect_delay" default:"1s"`
    
    // VAD configuration for pipeline mode
    VAD VADConfig `mapstructure:"vad"`
    
    // Metrics configuration
    EnableMetrics bool `mapstructure:"enable_metrics" default:"true"`
}

type VADConfig struct {
    // Enabled turns on voice activity detection
    Enabled bool `mapstructure:"enabled" default:"true"`
    
    // Threshold for speech detection (0.0 to 1.0)
    Threshold float64 `mapstructure:"threshold" default:"0.5"`
    
    // MinSpeechDuration to trigger recognition
    MinSpeechDuration time.Duration `mapstructure:"min_speech_duration" default:"100ms"`
    
    // MaxSpeechDuration before forcing end
    MaxSpeechDuration time.Duration `mapstructure:"max_speech_duration" default:"30s"`
    
    // SilenceTimeout to end speech
    SilenceTimeout time.Duration `mapstructure:"silence_timeout" default:"500ms"`
}
```

### Creating Voice Sessions

```go
import (
    "context"
    
    "github.com/lookatitude/beluga-ai/pkg/voice/session"
    "github.com/lookatitude/beluga-ai/pkg/voice/stt"
    "github.com/lookatitude/beluga-ai/pkg/voice/tts"
    "github.com/lookatitude/beluga-ai/pkg/voice/s2s"
)

func main() {
    ctx := context.Background()

    // Option 1: Use STT + Agent + TTS pipeline
    // This gives you more control and works with any LLM
    sttProvider, err := stt.NewProvider(ctx, "deepgram",
        stt.WithSTTModel("nova-2"),
        stt.WithSTTLanguage("en-US"),
    )
    if err != nil {
        log.Fatalf("Failed to create STT provider: %v", err)
    }
    
    ttsProvider, err := tts.NewProvider(ctx, "elevenlabs",
        tts.WithTTSVoice("rachel"),
        tts.WithTTSModel("eleven_multilingual_v2"),
    )
    if err != nil {
        log.Fatalf("Failed to create TTS provider: %v", err)
    }

    voiceSession, err := session.NewVoiceSession(ctx,
        session.WithSTTProvider(sttProvider),
        session.WithTTSProvider(ttsProvider),
        session.WithAgentInstance(agent),
        session.WithVAD(0.5, 500*time.Millisecond),
        session.WithMaxDuration(30*time.Minute),
        session.WithIdleTimeout(5*time.Minute),
        session.WithHistory(50),
    )
    if err != nil {
        log.Fatalf("Failed to create session: %v", err)
    }

    // Option 2: Use S2S (end-to-end)
    // Lower latency but requires S2S-capable model
    s2sProvider, err := s2s.NewProvider(ctx, "openai-realtime",
        s2s.WithS2SModel("gpt-4o-realtime-preview"),
        s2s.WithS2SVoice("alloy"),
        s2s.WithS2SSystemPrompt("You are a helpful assistant."),
        s2s.WithS2STurnDetection(0.5, 500),
    )
    if err != nil {
        log.Fatalf("Failed to create S2S provider: %v", err)
    }

    voiceSession, err := session.NewVoiceSession(ctx,
        session.WithS2SProvider(s2sProvider),
        session.WithMaxDuration(30*time.Minute),
        session.WithReconnect(3, 1*time.Second),
    )
    if err != nil {
        log.Fatalf("Failed to create S2S session: %v", err)
    }

    // Start the session and handle events
    if err := voiceSession.Start(ctx); err != nil {
        log.Fatalf("Failed to start session: %v", err)
    }
    defer voiceSession.Close()
}
```

### Session Lifecycle and Events

```go
// Session events for handling state changes
type SessionEvent struct {
    Type      SessionEventType
    Timestamp time.Time
    Data      interface{}
    Error     error
}

type SessionEventType string

const (
    EventSessionStarted    SessionEventType = "session_started"
    EventSessionEnded      SessionEventType = "session_ended"
    EventSpeechStarted     SessionEventType = "speech_started"
    EventSpeechEnded       SessionEventType = "speech_ended"
    EventTranscription     SessionEventType = "transcription"
    EventResponseStarted   SessionEventType = "response_started"
    EventResponseAudio     SessionEventType = "response_audio"
    EventResponseEnded     SessionEventType = "response_ended"
    EventFunctionCall      SessionEventType = "function_call"
    EventError             SessionEventType = "error"
    EventReconnecting      SessionEventType = "reconnecting"
    EventReconnected       SessionEventType = "reconnected"
)

// Subscribe to session events
func handleSessionEvents(session *session.VoiceSession) {
    events := session.Events()
    
    for event := range events {
        switch event.Type {
        case EventSessionStarted:
            log.Printf("Session started: %s", session.ID())
            
        case EventTranscription:
            transcript := event.Data.(*TranscriptionEvent)
            log.Printf("User said: %s (final: %v)", transcript.Text, transcript.IsFinal)
            
        case EventResponseAudio:
            audio := event.Data.(*AudioChunk)
            // Play audio to user
            playAudio(audio.Data)
            
        case EventFunctionCall:
            call := event.Data.(*FunctionCallEvent)
            log.Printf("Function call: %s(%v)", call.Name, call.Arguments)
            // Execute and send result back
            result := executeFunction(call)
            session.SendFunctionResult(call.ID, result)
            
        case EventError:
            log.Printf("Session error: %v", event.Error)
            
        case EventReconnecting:
            log.Printf("Reconnecting (attempt %d)", event.Data.(int))
            
        case EventSessionEnded:
            log.Printf("Session ended")
            return
        }
    }
}
```

### Handling Disconnections and Reconnection

```go
// Configure session with reconnection handling
voiceSession, err := session.NewVoiceSession(ctx,
    session.WithS2SProvider(s2sProvider),
    session.WithReconnect(3, 1*time.Second),
    session.WithOnReconnect(func(attempt int) {
        log.Printf("Reconnecting (attempt %d/3)...", attempt)
    }),
    session.WithOnReconnected(func() {
        log.Printf("Reconnected successfully!")
    }),
    session.WithOnDisconnect(func(err error) {
        log.Printf("Disconnected: %v", err)
    }),
)

// Manual reconnection if needed
if err := voiceSession.Reconnect(ctx); err != nil {
    log.Printf("Manual reconnection failed: %v", err)
}
```

### Session State Management

```go
// Get session state
state := voiceSession.State()

switch state {
case session.StateConnecting:
    log.Println("Connecting to voice service...")
case session.StateConnected:
    log.Println("Connected and ready")
case session.StateSpeaking:
    log.Println("User is speaking")
case session.StateProcessing:
    log.Println("Processing user input")
case session.StateResponding:
    log.Println("AI is responding")
case session.StateIdle:
    log.Println("Waiting for input")
case session.StateReconnecting:
    log.Println("Attempting to reconnect")
case session.StateDisconnected:
    log.Println("Disconnected")
case session.StateError:
    log.Println("Session in error state")
}

// Get session metrics
metrics := voiceSession.GetMetrics()
log.Printf("Session stats: %d turns, avg latency: %v", 
    metrics.TotalTurns, 
    metrics.AverageLatency,
)
```

## Audio Format Handling

Voice providers must handle audio format conversion properly:

```go
type AudioFormat struct {
    SampleRate int    // 8000, 16000, 24000, 48000
    Channels   int    // 1 (mono) or 2 (stereo)
    BitDepth   int    // 16 or 32
    Encoding   string // "PCM", "MP3", "OGG", "WAV"
}

// Convert between formats
func ConvertAudioFormat(audio []byte, from, to AudioFormat) ([]byte, error) {
    // Use internal audio utilities
    return internal.ConvertAudio(audio, from, to)
}
```

## OTEL Metrics for Audio

Voice providers should track audio-specific metrics to understand latency, throughput, and quality. Here's a comprehensive approach to voice telemetry.

### Standard Voice Metrics

```go
// Standard voice metrics with proper naming conventions
const (
    // STT Metrics
    MetricSTTTranscriptionsTotal    = "beluga.stt.transcriptions.total"
    MetricSTTTranscriptionLatency   = "beluga.stt.transcription_latency_seconds"
    MetricSTTActiveStreams          = "beluga.stt.active_streams"
    MetricSTTAudioBytesProcessed    = "beluga.stt.audio_bytes_processed.total"
    MetricSTTTranscriptLength       = "beluga.stt.transcript_length_chars"
    MetricSTTWordConfidence         = "beluga.stt.word_confidence"
    MetricSTTErrorsTotal            = "beluga.stt.errors.total"
    
    // TTS Metrics
    MetricTTSGenerationsTotal       = "beluga.tts.generations.total"
    MetricTTSGenerationLatency      = "beluga.tts.generation_latency_seconds"
    MetricTTSAudioBytesGenerated    = "beluga.tts.audio_bytes_generated.total"
    MetricTTSTextLength             = "beluga.tts.text_length_chars"
    MetricTTSFirstByteLatency       = "beluga.tts.first_byte_latency_seconds"
    MetricTTSErrorsTotal            = "beluga.tts.errors.total"
    
    // S2S Metrics
    MetricS2SConversationsTotal     = "beluga.s2s.conversations.total"
    MetricS2STurnLatency            = "beluga.s2s.turn_latency_seconds"
    MetricS2SActiveSessions         = "beluga.s2s.active_sessions"
    MetricS2SFunctionCallsTotal     = "beluga.s2s.function_calls.total"
    MetricS2SInterruptionsTotal     = "beluga.s2s.interruptions.total"
    MetricS2SErrorsTotal            = "beluga.s2s.errors.total"
    
    // Session Metrics
    MetricSessionDuration           = "beluga.voice.session_duration_seconds"
    MetricSessionTurns              = "beluga.voice.session_turns.total"
)
```

### Implementing Comprehensive OTEL Instrumentation

```go
package voice

import (
    "context"
    "time"
    
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/metric"
    "go.opentelemetry.io/otel/trace"
)

type VoiceMetrics struct {
    tracer          trace.Tracer
    meter           metric.Meter
    
    // STT metrics
    sttLatency      metric.Float64Histogram
    sttTranscripts  metric.Int64Counter
    sttActiveStreams metric.Int64UpDownCounter
    sttErrors       metric.Int64Counter
    
    // TTS metrics
    ttsLatency          metric.Float64Histogram
    ttsFirstByteLatency metric.Float64Histogram
    ttsGenerations      metric.Int64Counter
    ttsErrors           metric.Int64Counter
    
    // S2S metrics
    s2sTurnLatency      metric.Float64Histogram
    s2sActiveSessions   metric.Int64UpDownCounter
    s2sFunctionCalls    metric.Int64Counter
    s2sInterruptions    metric.Int64Counter
}

func NewVoiceMetrics(meterProvider metric.MeterProvider) (*VoiceMetrics, error) {
    meter := meterProvider.Meter("beluga.voice")
    tracer := otel.Tracer("beluga.voice")
    
    m := &VoiceMetrics{
        tracer: tracer,
        meter:  meter,
    }
    
    var err error
    
    // STT latency histogram with audio-appropriate buckets
    m.sttLatency, err = meter.Float64Histogram(
        MetricSTTTranscriptionLatency,
        metric.WithDescription("Latency of speech-to-text transcription"),
        metric.WithUnit("s"),
        metric.WithExplicitBucketBoundaries(0.1, 0.25, 0.5, 0.75, 1.0, 2.0, 5.0, 10.0),
    )
    if err != nil {
        return nil, err
    }
    
    // TTS first-byte latency (important for perceived responsiveness)
    m.ttsFirstByteLatency, err = meter.Float64Histogram(
        MetricTTSFirstByteLatency,
        metric.WithDescription("Time to first audio byte in TTS streaming"),
        metric.WithUnit("s"),
        metric.WithExplicitBucketBoundaries(0.05, 0.1, 0.2, 0.3, 0.5, 1.0),
    )
    if err != nil {
        return nil, err
    }
    
    // S2S turn latency (critical for conversational flow)
    m.s2sTurnLatency, err = meter.Float64Histogram(
        MetricS2STurnLatency,
        metric.WithDescription("End-to-end latency for S2S conversation turn"),
        metric.WithUnit("s"),
        metric.WithExplicitBucketBoundaries(0.2, 0.4, 0.6, 0.8, 1.0, 1.5, 2.0, 3.0),
    )
    if err != nil {
        return nil, err
    }
    
    // Active streams counter
    m.sttActiveStreams, err = meter.Int64UpDownCounter(
        MetricSTTActiveStreams,
        metric.WithDescription("Number of active STT streaming sessions"),
    )
    if err != nil {
        return nil, err
    }
    
    // Active S2S sessions
    m.s2sActiveSessions, err = meter.Int64UpDownCounter(
        MetricS2SActiveSessions,
        metric.WithDescription("Number of active S2S sessions"),
    )
    if err != nil {
        return nil, err
    }
    
    return m, nil
}

// RecordSTTLatency records STT transcription latency
func (m *VoiceMetrics) RecordSTTLatency(ctx context.Context, provider, model string, duration time.Duration) {
    m.sttLatency.Record(ctx, duration.Seconds(),
        metric.WithAttributes(
            attribute.String("provider", provider),
            attribute.String("model", model),
        ),
    )
}

// RecordTTSFirstByte records time to first audio byte
func (m *VoiceMetrics) RecordTTSFirstByte(ctx context.Context, provider, voice string, duration time.Duration) {
    m.ttsFirstByteLatency.Record(ctx, duration.Seconds(),
        metric.WithAttributes(
            attribute.String("provider", provider),
            attribute.String("voice", voice),
        ),
    )
}

// RecordS2STurnLatency records S2S conversation turn latency
func (m *VoiceMetrics) RecordS2STurnLatency(ctx context.Context, provider, model string, duration time.Duration) {
    m.s2sTurnLatency.Record(ctx, duration.Seconds(),
        metric.WithAttributes(
            attribute.String("provider", provider),
            attribute.String("model", model),
        ),
    )
}

// StartSTTStream tracks stream lifecycle
func (m *VoiceMetrics) StartSTTStream(ctx context.Context, provider string) {
    m.sttActiveStreams.Add(ctx, 1,
        metric.WithAttributes(attribute.String("provider", provider)),
    )
}

func (m *VoiceMetrics) EndSTTStream(ctx context.Context, provider string) {
    m.sttActiveStreams.Add(ctx, -1,
        metric.WithAttributes(attribute.String("provider", provider)),
    )
}
```

### Tracing Voice Operations

```go
// StartSTTSpan creates a span for STT operations
func (m *VoiceMetrics) StartSTTSpan(ctx context.Context, operation string, audioSize int) (context.Context, trace.Span) {
    return m.tracer.Start(ctx, "stt."+operation,
        trace.WithAttributes(
            attribute.Int("audio.size_bytes", audioSize),
            attribute.String("operation", operation),
        ),
        trace.WithSpanKind(trace.SpanKindClient),
    )
}

// StartTTSSpan creates a span for TTS operations
func (m *VoiceMetrics) StartTTSSpan(ctx context.Context, operation string, textLen int) (context.Context, trace.Span) {
    return m.tracer.Start(ctx, "tts."+operation,
        trace.WithAttributes(
            attribute.Int("text.length", textLen),
            attribute.String("operation", operation),
        ),
        trace.WithSpanKind(trace.SpanKindClient),
    )
}

// StartS2SSpan creates a span for S2S operations
func (m *VoiceMetrics) StartS2SSpan(ctx context.Context, operation, sessionID string) (context.Context, trace.Span) {
    return m.tracer.Start(ctx, "s2s."+operation,
        trace.WithAttributes(
            attribute.String("session.id", sessionID),
            attribute.String("operation", operation),
        ),
        trace.WithSpanKind(trace.SpanKindClient),
    )
}

// RecordVoiceError records errors with context
func (m *VoiceMetrics) RecordVoiceError(span trace.Span, err error, errorCode string) {
    span.RecordError(err,
        trace.WithAttributes(
            attribute.String("error.code", errorCode),
        ),
    )
}
```

### Audio-Specific Latency Considerations

When measuring voice latency, consider these components:

```go
// Audio latency components
type LatencyBreakdown struct {
    // Network latency to provider
    NetworkLatency time.Duration
    
    // Time to first byte (important for perceived responsiveness)
    FirstByteLatency time.Duration
    
    // Total processing time
    ProcessingTime time.Duration
    
    // Streaming chunk interval
    ChunkInterval time.Duration
    
    // Client-side buffering
    BufferLatency time.Duration
}

// Track latency components in spans
func recordLatencyBreakdown(span trace.Span, breakdown LatencyBreakdown) {
    span.SetAttributes(
        attribute.Float64("latency.network_ms", float64(breakdown.NetworkLatency.Milliseconds())),
        attribute.Float64("latency.first_byte_ms", float64(breakdown.FirstByteLatency.Milliseconds())),
        attribute.Float64("latency.processing_ms", float64(breakdown.ProcessingTime.Milliseconds())),
        attribute.Float64("latency.buffer_ms", float64(breakdown.BufferLatency.Milliseconds())),
    )
}
```

## Error Handling

Voice providers should use consistent error handling:

```go
// Error codes for voice providers
const (
    ErrCodeInvalidConfig   = "invalid_config"
    ErrCodeNetworkError    = "network_error"
    ErrCodeTimeout         = "timeout"
    ErrCodeRateLimit       = "rate_limit"
    ErrCodeAuthentication  = "authentication_failed"
    ErrCodeStreamError     = "stream_error"
    ErrCodeStreamClosed    = "stream_closed"
    ErrCodeAudioFormat     = "invalid_audio_format"
)

func (p *Provider) handleError(operation string, err error) error {
    var code string
    
    switch {
    case strings.Contains(err.Error(), "rate limit"):
        code = ErrCodeRateLimit
    case strings.Contains(err.Error(), "unauthorized"):
        code = ErrCodeAuthentication
    case strings.Contains(err.Error(), "timeout"):
        code = ErrCodeTimeout
    default:
        code = ErrCodeNetworkError
    }
    
    return NewProviderError(operation, code, err)
}
```

## Testing Voice Providers

### Mock Providers

```go
// Use built-in mocks for testing
mockSTT := stt.NewAdvancedMockSTTProvider("test",
    stt.WithTranscriptions("Hello world", "How are you"),
    stt.WithStreamingDelay(10 * time.Millisecond),
)

mockTTS := tts.NewAdvancedMockTTSProvider("test",
    tts.WithAudioResponses([]byte{1, 2, 3, 4}),
)
```

### Integration Tests

```go
func TestMyServiceSTT(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test")
    }

    config := stt.Config{
        Provider: "myservice",
        APIKey:   os.Getenv("MYSERVICE_API_KEY"),
    }

    provider, err := NewMyServiceSTT(context.Background(), config)
    require.NoError(t, err)

    // Test transcription
    audio := loadTestAudio(t, "test.wav")
    text, err := provider.Transcribe(context.Background(), audio)
    require.NoError(t, err)
    assert.NotEmpty(t, text)
}
```

### Streaming Tests

```go
func TestStreamingTranscription(t *testing.T) {
    provider := setupTestProvider(t)
    
    session, err := provider.StartStreaming(context.Background())
    require.NoError(t, err)
    defer session.Close()

    // Send audio chunks
    for _, chunk := range audioChunks {
        err := session.SendAudio(context.Background(), chunk)
        require.NoError(t, err)
    }

    // Collect results
    var transcript string
    for result := range session.ReceiveTranscript() {
        require.NoError(t, result.Error)
        if result.IsFinal {
            transcript = result.Text
            break
        }
    }

    assert.NotEmpty(t, transcript)
}
```

## Best Practices

### 1. Handle Streaming Gracefully

```go
// Always check context cancellation in streaming loops
func (s *session) receiveLoop(ctx context.Context) {
    for {
        select {
        case <-ctx.Done():
            return
        default:
            result, err := s.stream.Receive()
            if err != nil {
                // Handle gracefully
                return
            }
            s.resultChan <- result
        }
    }
}
```

### 2. Buffer Audio Appropriately

```go
// Use appropriate buffer sizes for audio
const (
    AudioChunkSize = 4096  // bytes
    BufferDuration = 100   // milliseconds
)

// Calculate buffer size based on format
func bufferSize(sampleRate, channels, bitDepth int, durationMs int) int {
    bytesPerSample := bitDepth / 8
    samplesPerMs := sampleRate / 1000
    return samplesPerMs * channels * bytesPerSample * durationMs
}
```

### 3. Implement Health Checks

```go
func (p *Provider) CheckHealth() map[string]any {
    return map[string]any{
        "provider":     p.GetName(),
        "status":       "healthy",
        "last_latency": p.lastLatency.Load(),
        "error_rate":   p.errorRate.Load(),
    }
}
```

## Related Resources

- **[LLM Provider Integration Guide](./llm-providers.md)**: Similar patterns for LLM providers
- **[Extensibility Guide](./extensibility.md)**: General framework extension patterns
- **[Voice Backends Cookbook](../cookbook/voice-backends.md)**: Quick recipes for voice configuration
- **[Voice Sessions Use Case](../use-cases/voice-sessions.md)**: Real-world voice agent implementation
- **[STT Package Documentation](https://github.com/lookatitude/beluga-ai/blob/main/pkg/voice/stt/README.md)**: Detailed STT package docs
- **[TTS Package Documentation](https://github.com/lookatitude/beluga-ai/blob/main/pkg/voice/tts/README.md)**: Detailed TTS package docs
- **[S2S Package Documentation](https://github.com/lookatitude/beluga-ai/blob/main/pkg/voice/s2s/README.md)**: Detailed S2S package docs