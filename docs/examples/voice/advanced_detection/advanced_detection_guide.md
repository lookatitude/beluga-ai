# Advanced Voice Detection Guide

> **Learn how to implement VAD, turn detection, and noise cancellation for production-quality voice agents.**

## Introduction

Building a natural-feeling voice agent requires more than just speech-to-text and text-to-speech. You need to detect when the user is speaking, know when they've finished their turn, and handle background noise gracefully. These three components—Voice Activity Detection (VAD), turn detection, and noise cancellation—work together to create a seamless conversational experience.

In this guide, you'll learn:

- How VAD algorithms detect speech in audio streams
- How turn detection determines when it's the agent's turn to respond
- How noise cancellation improves transcription accuracy
- How to combine all three for production voice agents
- How to instrument these components with OTEL

## Prerequisites

| Requirement | Why You Need It |
|-------------|-----------------|
| **Go 1.24+** | Required for Beluga AI framework |
| **Understanding of audio fundamentals** | PCM, sample rates, frames |
| **Voice provider setup** | STT/TTS or S2S provider configured |

## Concepts

### Voice Activity Detection (VAD)

VAD determines whether audio contains speech or silence. Think of it like a smart microphone that knows when to listen:

```
Audio Stream:  [silence][noise][SPEECH][silence][SPEECH][noise][silence]
VAD Output:    [  0   ][  0  ][  1   ][   0   ][  1   ][  0  ][   0   ]
```

VAD algorithms typically analyze:
- **Energy levels**: Speech is louder than silence
- **Zero-crossing rate**: Speech has characteristic patterns
- **Spectral features**: Speech has different frequency content than noise
- **Neural models**: Modern VAD uses trained models for accuracy

### Turn Detection

Turn detection builds on VAD to determine when the user has finished speaking and expects a response. This is more nuanced than VAD alone:

```
User: "Hey, can you help me with... [pause] ...finding a restaurant?"
                                   ↑ VAD might end here, but turn isn't complete

Turn Detection considers:
- Silence duration (longer = turn complete)
- Intonation patterns (falling pitch = statement complete)
- Semantic completion (sentence structure)
- Context (follow-up question expected?)
```

### Noise Cancellation

Noise cancellation removes unwanted sounds to improve transcription accuracy:

```
Input:  [keyboard + fan + SPEECH + typing]
Output: [             SPEECH              ]

Techniques:
- Spectral subtraction: Remove known noise patterns
- Adaptive filtering: Learn and remove consistent background
- Neural denoising: ML-based noise separation
```

## Step-by-Step Tutorial

### Step 1: Set Up the Audio Pipeline

First, let's create the foundation for processing audio:

```go
package main

import (
    "context"
    "sync"
    "time"

    "github.com/lookatitude/beluga-ai/pkg/voice/vad"
    "github.com/lookatitude/beluga-ai/pkg/voice/turn"
    "github.com/lookatitude/beluga-ai/pkg/voice/denoise"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/metric"
)

// AudioPipeline processes audio through VAD, denoising, and turn detection
type AudioPipeline struct {
    vad      vad.Detector
    denoiser denoise.Denoiser
    turn     turn.Detector
    
    // Configuration
    sampleRate int
    channels   int
    frameSize  int
    
    // State
    isSpeaking   bool
    speechBuffer []byte
    mu           sync.Mutex
    
    // Metrics
    metrics *PipelineMetrics
}

type PipelineConfig struct {
    SampleRate int           `default:"16000"`
    Channels   int           `default:"1"`
    FrameSize  int           `default:"512"` // samples per frame
    
    // VAD settings
    VADThreshold float64     `default:"0.5"`
    VADModel     string      `default:"silero"` // "silero", "webrtc", "energy"
    
    // Turn detection settings
    SilenceDuration    time.Duration `default:"500ms"`
    MinSpeechDuration  time.Duration `default:"100ms"`
    MaxSpeechDuration  time.Duration `default:"30s"`
    
    // Noise cancellation
    EnableDenoise bool   `default:"true"`
    DenoiseModel  string `default:"rnnoise"` // "rnnoise", "spectral"
}

// NewAudioPipeline creates a configured audio processing pipeline
func NewAudioPipeline(ctx context.Context, config PipelineConfig) (*AudioPipeline, error) {
    // Create VAD detector
    vadDetector, err := vad.NewDetector(ctx,
        vad.WithModel(config.VADModel),
        vad.WithThreshold(config.VADThreshold),
        vad.WithSampleRate(config.SampleRate),
    )
    if err != nil {
        return nil, fmt.Errorf("failed to create VAD: %w", err)
    }
    
    // Create turn detector
    turnDetector := turn.NewDetector(
        turn.WithSilenceDuration(config.SilenceDuration),
        turn.WithMinSpeechDuration(config.MinSpeechDuration),
        turn.WithMaxSpeechDuration(config.MaxSpeechDuration),
    )
    
    // Create denoiser if enabled
    var denoiser denoise.Denoiser
    if config.EnableDenoise {
        denoiser, err = denoise.NewDenoiser(ctx,
            denoise.WithModel(config.DenoiseModel),
            denoise.WithSampleRate(config.SampleRate),
        )
        if err != nil {
            return nil, fmt.Errorf("failed to create denoiser: %w", err)
        }
    }
    
    // Initialize metrics
    metrics, err := newPipelineMetrics()
    if err != nil {
        return nil, fmt.Errorf("failed to create metrics: %w", err)
    }
    
    return &AudioPipeline{
        vad:        vadDetector,
        denoiser:   denoiser,
        turn:       turnDetector,
        sampleRate: config.SampleRate,
        channels:   config.Channels,
        frameSize:  config.FrameSize,
        metrics:    metrics,
    }, nil
}
```

### Step 2: Implement VAD Processing

Now let's implement the core VAD logic:

```go
// VAD result for a single frame
type VADResult struct {
    IsSpeech    bool
    Probability float64
    Timestamp   time.Time
}

// ProcessFrame runs VAD on a single audio frame
func (p *AudioPipeline) ProcessFrame(ctx context.Context, frame []byte) (*VADResult, error) {
    tracer := otel.Tracer("audio-pipeline")
    ctx, span := tracer.Start(ctx, "vad.ProcessFrame",
        trace.WithAttributes(
            attribute.Int("frame_size", len(frame)),
        ),
    )
    defer span.End()
    
    start := time.Now()
    
    // Denoise first if enabled
    var cleanFrame []byte
    if p.denoiser != nil {
        var err error
        cleanFrame, err = p.denoiser.Process(ctx, frame)
        if err != nil {
            span.RecordError(err)
            return nil, fmt.Errorf("denoise failed: %w", err)
        }
        p.metrics.recordDenoiseLatency(ctx, time.Since(start))
    } else {
        cleanFrame = frame
    }
    
    // Run VAD
    vadStart := time.Now()
    probability, err := p.vad.Detect(ctx, cleanFrame)
    if err != nil {
        span.RecordError(err)
        return nil, fmt.Errorf("VAD failed: %w", err)
    }
    p.metrics.recordVADLatency(ctx, time.Since(vadStart))
    
    isSpeech := probability >= p.vad.Threshold()
    
    span.SetAttributes(
        attribute.Bool("is_speech", isSpeech),
        attribute.Float64("probability", probability),
    )
    
    return &VADResult{
        IsSpeech:    isSpeech,
        Probability: probability,
        Timestamp:   time.Now(),
    }, nil
}
```

### Step 3: Implement Turn Detection

Turn detection tracks speech segments and determines when a turn is complete:

```go
// TurnEvent represents a complete user turn
type TurnEvent struct {
    Audio       []byte
    StartTime   time.Time
    EndTime     time.Time
    Duration    time.Duration
    Reason      TurnEndReason
}

type TurnEndReason string

const (
    TurnEndSilence    TurnEndReason = "silence"
    TurnEndMaxLength  TurnEndReason = "max_length"
    TurnEndInterrupt  TurnEndReason = "interrupt"
)

// ProcessAudioStream handles continuous audio input
func (p *AudioPipeline) ProcessAudioStream(
    ctx context.Context,
    audioIn <-chan []byte,
    turnsOut chan<- *TurnEvent,
) error {
    tracer := otel.Tracer("audio-pipeline")
    
    var speechStart time.Time
    silenceStart := time.Now()
    
    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
            
        case frame, ok := <-audioIn:
            if !ok {
                // Channel closed, flush any remaining speech
                if p.isSpeaking && len(p.speechBuffer) > 0 {
                    p.emitTurn(turnsOut, speechStart, TurnEndSilence)
                }
                return nil
            }
            
            // Process the frame
            result, err := p.ProcessFrame(ctx, frame)
            if err != nil {
                // Log but continue processing
                continue
            }
            
            p.mu.Lock()
            
            if result.IsSpeech {
                if !p.isSpeaking {
                    // Speech started
                    _, span := tracer.Start(ctx, "turn.SpeechStarted")
                    span.End()
                    
                    p.isSpeaking = true
                    speechStart = result.Timestamp
                    p.speechBuffer = nil
                    p.metrics.incrementSpeechSegments(ctx)
                }
                
                // Accumulate speech
                p.speechBuffer = append(p.speechBuffer, frame...)
                silenceStart = time.Now()
                
                // Check max duration
                if time.Since(speechStart) > p.turn.MaxSpeechDuration() {
                    p.emitTurn(turnsOut, speechStart, TurnEndMaxLength)
                    p.isSpeaking = false
                    p.speechBuffer = nil
                }
                
            } else {
                if p.isSpeaking {
                    // Still include some silence in buffer for natural boundaries
                    p.speechBuffer = append(p.speechBuffer, frame...)
                    
                    // Check if silence duration exceeds threshold
                    silenceDuration := time.Since(silenceStart)
                    speechDuration := time.Since(speechStart)
                    
                    if silenceDuration >= p.turn.SilenceDuration() &&
                       speechDuration >= p.turn.MinSpeechDuration() {
                        // Turn complete!
                        p.emitTurn(turnsOut, speechStart, TurnEndSilence)
                        p.isSpeaking = false
                        p.speechBuffer = nil
                    }
                }
            }
            
            p.mu.Unlock()
        }
    }
}

func (p *AudioPipeline) emitTurn(
    turnsOut chan<- *TurnEvent,
    startTime time.Time,
    reason TurnEndReason,
) {
    endTime := time.Now()
    duration := endTime.Sub(startTime)
    
    // Record turn metrics
    p.metrics.recordTurnDuration(context.Background(), duration)
    
    // Create a copy of the buffer
    audioCopy := make([]byte, len(p.speechBuffer))
    copy(audioCopy, p.speechBuffer)
    
    turnsOut <- &TurnEvent{
        Audio:     audioCopy,
        StartTime: startTime,
        EndTime:   endTime,
        Duration:  duration,
        Reason:    reason,
    }
}
```

### Step 4: Add Noise Cancellation

Integrate noise cancellation to improve transcription quality:

```go
// DenoiseConfig configures the noise cancellation
type DenoiseConfig struct {
    Model          string  // "rnnoise", "spectral", "adaptive"
    AggressiveLevel int    // 0-3, higher = more aggressive
    LearnRate      float64 // For adaptive filter
}

// RNNoiseDenoiser uses RNNoise neural model
type RNNoiseDenoiser struct {
    model   *rnnoise.Model
    state   *rnnoise.State
    metrics *DenoiseMetrics
}

func NewRNNoiseDenoiser(sampleRate int) (*RNNoiseDenoiser, error) {
    // RNNoise expects 48kHz, we'll need to resample
    model, err := rnnoise.Load()
    if err != nil {
        return nil, fmt.Errorf("failed to load RNNoise model: %w", err)
    }
    
    return &RNNoiseDenoiser{
        model: model,
        state: rnnoise.NewState(model),
    }, nil
}

func (d *RNNoiseDenoiser) Process(ctx context.Context, audio []byte) ([]byte, error) {
    tracer := otel.Tracer("denoise")
    ctx, span := tracer.Start(ctx, "rnnoise.Process")
    defer span.End()
    
    start := time.Now()
    
    // Convert to float32 for processing
    samples := bytesToFloat32(audio)
    
    // Process through RNNoise
    denoised := make([]float32, len(samples))
    vadProb := d.state.Process(samples, denoised)
    
    span.SetAttributes(
        attribute.Float64("vad_probability", float64(vadProb)),
        attribute.Int("samples_processed", len(samples)),
    )
    
    d.metrics.recordLatency(ctx, time.Since(start))
    
    // Convert back to bytes
    return float32ToBytes(denoised), nil
}

// SpectralDenoiser uses spectral subtraction
type SpectralDenoiser struct {
    noiseProfile []float64
    fftSize      int
    hopSize      int
}

func (d *SpectralDenoiser) Process(ctx context.Context, audio []byte) ([]byte, error) {
    // Perform FFT
    // Subtract noise spectrum
    // Perform inverse FFT
    // This is a simplified version - production would use overlap-add
    
    samples := bytesToFloat32(audio)
    
    // Apply spectral subtraction
    for i := range samples {
        // Simple spectral floor (production would be more sophisticated)
        if abs(samples[i]) < d.noiseFloor {
            samples[i] = 0
        }
    }
    
    return float32ToBytes(samples), nil
}
```

### Step 5: Combine All Components

Here's how to use all three components together in a voice agent:

```go
// VoiceAgent integrates audio pipeline with STT and agent
type VoiceAgent struct {
    pipeline *AudioPipeline
    stt      stt.Provider
    tts      tts.Provider
    agent    agents.Agent
    
    // Channels
    audioIn    chan []byte
    turnsOut   chan *TurnEvent
    responseCh chan *Response
}

func (v *VoiceAgent) Run(ctx context.Context) error {
    // Start audio pipeline in background
    go func() {
        if err := v.pipeline.ProcessAudioStream(ctx, v.audioIn, v.turnsOut); err != nil {
            log.Printf("Audio pipeline error: %v", err)
        }
    }()
    
    // Process completed turns
    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
            
        case turn := <-v.turnsOut:
            // Turn complete - transcribe the audio
            transcript, err := v.stt.Transcribe(ctx, turn.Audio)
            if err != nil {
                log.Printf("Transcription error: %v", err)
                continue
            }
            
            log.Printf("User: %s (duration: %v)", transcript, turn.Duration)
            
            // Get agent response
            response, err := v.agent.Run(ctx, transcript)
            if err != nil {
                log.Printf("Agent error: %v", err)
                continue
            }
            
            // Generate speech
            audio, err := v.tts.GenerateSpeech(ctx, response.Content)
            if err != nil {
                log.Printf("TTS error: %v", err)
                continue
            }
            
            // Send response
            v.responseCh <- &Response{
                Text:  response.Content,
                Audio: audio,
            }
        }
    }
}
```

### Step 6: Add OTEL Instrumentation

Complete instrumentation for the audio pipeline:

```go
type PipelineMetrics struct {
    tracer trace.Tracer
    meter  metric.Meter
    
    vadLatency       metric.Float64Histogram
    denoiseLatency   metric.Float64Histogram
    turnDuration     metric.Float64Histogram
    speechSegments   metric.Int64Counter
    vadDecisions     metric.Int64Counter
}

func newPipelineMetrics() (*PipelineMetrics, error) {
    meter := otel.Meter("beluga.voice.pipeline")
    
    vadLatency, err := meter.Float64Histogram(
        "beluga.voice.vad_latency_seconds",
        metric.WithDescription("Latency of VAD processing per frame"),
        metric.WithUnit("s"),
        metric.WithExplicitBucketBoundaries(0.001, 0.005, 0.01, 0.02, 0.05),
    )
    if err != nil {
        return nil, err
    }
    
    denoiseLatency, err := meter.Float64Histogram(
        "beluga.voice.denoise_latency_seconds",
        metric.WithDescription("Latency of noise cancellation per frame"),
        metric.WithUnit("s"),
        metric.WithExplicitBucketBoundaries(0.001, 0.005, 0.01, 0.02, 0.05),
    )
    if err != nil {
        return nil, err
    }
    
    turnDuration, err := meter.Float64Histogram(
        "beluga.voice.turn_duration_seconds",
        metric.WithDescription("Duration of user speech turns"),
        metric.WithUnit("s"),
        metric.WithExplicitBucketBoundaries(0.5, 1.0, 2.0, 5.0, 10.0, 30.0),
    )
    if err != nil {
        return nil, err
    }
    
    speechSegments, err := meter.Int64Counter(
        "beluga.voice.speech_segments_total",
        metric.WithDescription("Total number of detected speech segments"),
    )
    if err != nil {
        return nil, err
    }
    
    vadDecisions, err := meter.Int64Counter(
        "beluga.voice.vad_decisions_total",
        metric.WithDescription("Total VAD decisions"),
    )
    if err != nil {
        return nil, err
    }
    
    return &PipelineMetrics{
        tracer:         otel.Tracer("beluga.voice.pipeline"),
        meter:          meter,
        vadLatency:     vadLatency,
        denoiseLatency: denoiseLatency,
        turnDuration:   turnDuration,
        speechSegments: speechSegments,
        vadDecisions:   vadDecisions,
    }, nil
}

func (m *PipelineMetrics) recordVADLatency(ctx context.Context, d time.Duration) {
    m.vadLatency.Record(ctx, d.Seconds())
}

func (m *PipelineMetrics) recordDenoiseLatency(ctx context.Context, d time.Duration) {
    m.denoiseLatency.Record(ctx, d.Seconds())
}

func (m *PipelineMetrics) recordTurnDuration(ctx context.Context, d time.Duration) {
    m.turnDuration.Record(ctx, d.Seconds())
}

func (m *PipelineMetrics) incrementSpeechSegments(ctx context.Context) {
    m.speechSegments.Add(ctx, 1)
}
```

## Code Example

See the complete implementation:

- [advanced_detection.go](./advanced_detection.go) - Full production implementation
- [advanced_detection_test.go](./advanced_detection_test.go) - Test suite

## Testing

### Unit Testing VAD

```go
func TestVADDetection(t *testing.T) {
    tests := []struct {
        name       string
        audio      []byte
        wantSpeech bool
    }{
        {
            name:       "silence should not trigger",
            audio:      generateSilence(16000, 100*time.Millisecond),
            wantSpeech: false,
        },
        {
            name:       "speech should trigger",
            audio:      loadTestAudio(t, "speech_sample.wav"),
            wantSpeech: true,
        },
        {
            name:       "noise alone should not trigger",
            audio:      loadTestAudio(t, "background_noise.wav"),
            wantSpeech: false,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            vad := createTestVAD(t)
            result, err := vad.Detect(context.Background(), tt.audio)
            require.NoError(t, err)
            assert.Equal(t, tt.wantSpeech, result.IsSpeech)
        })
    }
}
```

### Testing Turn Detection

```go
func TestTurnDetection(t *testing.T) {
    pipeline := createTestPipeline(t)
    
    audioIn := make(chan []byte, 100)
    turnsOut := make(chan *TurnEvent, 10)
    
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    
    go pipeline.ProcessAudioStream(ctx, audioIn, turnsOut)
    
    // Simulate: speech -> silence -> turn end
    for _, frame := range generateSpeechFrames(1 * time.Second) {
        audioIn <- frame
    }
    for _, frame := range generateSilenceFrames(600 * time.Millisecond) {
        audioIn <- frame
    }
    close(audioIn)
    
    // Should receive exactly one turn
    select {
    case turn := <-turnsOut:
        assert.Equal(t, TurnEndSilence, turn.Reason)
        assert.True(t, turn.Duration >= 1*time.Second)
    case <-time.After(2 * time.Second):
        t.Fatal("Expected turn event not received")
    }
}
```

### Benchmarking

```go
func BenchmarkVADProcessing(b *testing.B) {
    vad := createTestVAD(b)
    frame := generateSpeechFrame(512) // 512 samples
    ctx := context.Background()
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, _ = vad.Detect(ctx, frame)
    }
}

func BenchmarkDenoising(b *testing.B) {
    denoiser := createTestDenoiser(b)
    frame := loadNoisyAudioFrame(b)
    ctx := context.Background()
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, _ = denoiser.Process(ctx, frame)
    }
}
```

## Best Practices

### 1. Tune VAD Threshold for Your Use Case

```go
// More sensitive for quiet environments
vad.WithThreshold(0.3)

// Less sensitive for noisy environments
vad.WithThreshold(0.7)

// Adaptive threshold based on ambient noise
vad.WithAdaptiveThreshold(true)
```

### 2. Handle Edge Cases in Turn Detection

```go
// Handle user interruptions
if currentlyResponding && turnDetected {
    // Stop current response
    cancelCurrentResponse()
    // Process new turn
    processNewTurn(turn)
}

// Handle very short utterances
if turn.Duration < 200*time.Millisecond {
    // Might be a noise burst, verify with STT
    transcript, _ := stt.Transcribe(ctx, turn.Audio)
    if len(transcript) < 2 {
        // Probably noise, ignore
        continue
    }
}
```

### 3. Choose the Right Denoising Strategy

```go
// Use RNNoise for general purpose denoising
denoise.WithModel("rnnoise")

// Use spectral subtraction for stationary noise
denoise.WithModel("spectral")

// Use adaptive filtering for changing environments
denoise.WithModel("adaptive")
denoise.WithLearnRate(0.01)
```

### 4. Optimize for Latency

```go
// Process smaller frames for lower latency
config.FrameSize = 256  // ~16ms at 16kHz

// Use efficient models
config.VADModel = "webrtc"  // Fast CPU-based VAD
config.DenoiseModel = "spectral"  // Fast spectral method

// Parallelize when possible
go denoiser.Process(ctx, frame)
go vad.Detect(ctx, cleanFrame)
```

## Troubleshooting

### Q: VAD is triggering on background noise
**A:** Increase the VAD threshold, or enable noise cancellation before VAD. Also consider using a model-based VAD (Silero) instead of energy-based.

### Q: Turn detection ends too early
**A:** Increase `SilenceDuration` to require more silence before ending a turn. Also check your VAD sensitivity - it might be detecting micro-pauses as silence.

### Q: Processing latency is too high
**A:** Use smaller frame sizes, faster models (webrtc VAD instead of Silero), and ensure you're not processing on the main thread. Profile your pipeline to find bottlenecks.

### Q: Noise cancellation removes speech
**A:** Reduce the aggressiveness level or use a model better suited for your noise type. RNNoise works well for general noise; spectral subtraction is better for stationary noise.

## Related Resources

- **[Voice Providers Guide](../../guides/voice-providers.md)**: Comprehensive STT/TTS/S2S integration
- **[Voice Sessions Use Case](../../use-cases/voice-sessions.md)**: Full voice agent implementation
- **[Voice Backends Cookbook](../../cookbook/voice-backends.md)**: Quick configuration recipes
- **[Observability Tracing Guide](../../guides/observability-tracing.md)**: Distributed tracing setup
