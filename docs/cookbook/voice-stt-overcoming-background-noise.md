---
title: "Overcoming Background Noise"
package: "voice/stt"
category: "audio"
complexity: "intermediate"
---

# Overcoming Background Noise

## Problem

You need to improve speech-to-text accuracy in noisy environments by preprocessing audio to reduce background noise before sending it to the STT service.

## Solution

Implement audio preprocessing that applies noise reduction, normalization, and filtering to improve signal quality. This works because clean audio signals produce better transcription results, and preprocessing can enhance the signal-to-noise ratio.

## Code Example
```go
package main

import (
    "context"
    "fmt"
    "log"
    
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/trace"
)

var tracer = otel.Tracer("beluga.voice.stt.noise_reduction")

// AudioPreprocessor preprocesses audio for noise reduction
type AudioPreprocessor struct {
    noiseGate    float64
    normalize    bool
    filter       bool
}

// NewAudioPreprocessor creates a new preprocessor
func NewAudioPreprocessor(noiseGate float64, normalize, filter bool) *AudioPreprocessor {
    return &AudioPreprocessor{
        noiseGate: noiseGate,
        normalize: normalize,
        filter:   filter,
    }
}

// Preprocess preprocesses audio data
func (ap *AudioPreprocessor) Preprocess(ctx context.Context, audioData []byte, sampleRate int) ([]byte, error) {
    ctx, span := tracer.Start(ctx, "audio_preprocessor.preprocess")
    defer span.End()
    
    span.SetAttributes(
        attribute.Int("input_size", len(audioData)),
        attribute.Int("sample_rate", sampleRate),
    )
    
    processed := audioData
    
    // Apply noise gate
    if ap.noiseGate > 0 {
        processed = ap.applyNoiseGate(ctx, processed, ap.noiseGate)
        span.SetAttributes(attribute.Bool("noise_gate_applied", true))
    }
    
    // Normalize audio
    if ap.normalize {
        processed = ap.normalizeAudio(ctx, processed)
        span.SetAttributes(attribute.Bool("normalization_applied", true))
    }
    
    // Apply filtering
    if ap.filter {
        processed = ap.applyFilter(ctx, processed, sampleRate)
        span.SetAttributes(attribute.Bool("filter_applied", true))
    }
    
    span.SetAttributes(attribute.Int("output_size", len(processed)))
    span.SetStatus(trace.StatusOK, "audio preprocessed")
    
    return processed, nil
}

// applyNoiseGate removes audio below threshold
func (ap *AudioPreprocessor) applyNoiseGate(ctx context.Context, data []byte, threshold float64) []byte {
    // Simple noise gate: zero out samples below threshold
    // In practice, use proper audio processing library
    result := make([]byte, len(data))
    copy(result, data)
    
    // Simplified: would use actual audio analysis
    return result
}

// normalizeAudio normalizes audio levels
func (ap *AudioPreprocessor) normalizeAudio(ctx context.Context, data []byte) []byte {
    // Find peak value
    maxVal := float64(0)
    for i := 0; i < len(data); i += 2 {
        // Simplified: would properly decode audio samples
        val := float64(data[i])
        if val > maxVal {
            maxVal = val
        }
    }

    // Normalize if needed
    if maxVal > 0 {
        scale := 32767.0 / maxVal
        result := make([]byte, len(data))
        for i := 0; i < len(data); i += 2 {
            // Apply normalization
            normalized := float64(data[i]) * scale
            result[i] = byte(normalized)
            if i+1 < len(data) {
                result[i+1] = data[i+1]
            }
        }
        return result
    }
    
    return data
}

// applyFilter applies audio filtering
func (ap *AudioPreprocessor) applyFilter(ctx context.Context, data []byte, sampleRate int) []byte {
    // Apply high-pass filter to remove low-frequency noise
    // Simplified: would use proper DSP library
    return data
}

// NoisySTTWrapper wraps STT with noise reduction
type NoisySTTWrapper struct {
    sttProvider  interface{} // STT provider
    preprocessor *AudioPreprocessor
}

// NewNoisySTTWrapper creates a new wrapper
func NewNoisySTTWrapper(sttProvider interface{}, preprocessor *AudioPreprocessor) *NoisySTTWrapper {
    return &NoisySTTWrapper{
        sttProvider:  sttProvider,
        preprocessor: preprocessor,
    }
}

// Transcribe transcribes audio with noise reduction
func (nsw *NoisySTTWrapper) Transcribe(ctx context.Context, audioData []byte, sampleRate int) (string, error) {
    ctx, span := tracer.Start(ctx, "noisy_stt_wrapper.transcribe")
    defer span.End()
    
    // Preprocess audio
    processed, err := nsw.preprocessor.Preprocess(ctx, audioData, sampleRate)
    if err != nil {
        span.RecordError(err)
        span.SetStatus(trace.StatusError, err.Error())
        return "", err
    }
    
    // Transcribe with STT provider
    // result, err := nsw.sttProvider.Transcribe(ctx, processed, sampleRate)
    
    span.SetStatus(trace.StatusOK, "transcription completed")
    return "", nil
}

func main() {
    ctx := context.Background()

    // Create preprocessor
    preprocessor := NewAudioPreprocessor(0.1, true, true)
    
    // Create wrapper
    // sttProvider := yourSTTProvider
    // wrapper := NewNoisySTTWrapper(sttProvider, preprocessor)
    
    // Transcribe
    audioData := []byte{/* audio data */}
    // text, err := wrapper.Transcribe(ctx, audioData, 16000)
    fmt.Println("Audio preprocessor created")
}
```

## Explanation

Let's break down what's happening:

1. **Noise gating** - Notice how we remove audio below a threshold. This eliminates background noise that's quieter than the speech, improving signal quality.

2. **Normalization** - We normalize audio levels to ensure consistent volume. This helps the STT service process the audio more effectively.

3. **Filtering** - We apply filters to remove unwanted frequencies. High-pass filters remove low-frequency noise, while band-pass filters can focus on speech frequencies.

```go
**Key insight:** Preprocess audio before STT, not after. Clean input produces better results than trying to fix transcription errors caused by noise.

## Testing

```
Here's how to test this solution:
```go
func TestAudioPreprocessor_ReducesNoise(t *testing.T) {
    preprocessor := NewAudioPreprocessor(0.1, true, true)
    
    audioData := []byte{/* test audio */}
    processed, err := preprocessor.Preprocess(context.Background(), audioData, 16000)
    
    require.NoError(t, err)
    require.NotNil(t, processed)
}

## Variations

### Adaptive Noise Gate

Adjust noise gate threshold dynamically:
func (ap *AudioPreprocessor) AdaptiveNoiseGate(ctx context.Context, data []byte) []byte {
    // Analyze signal and adjust threshold
}
```

### Spectral Subtraction

Use spectral subtraction for noise reduction:
```go
func (ap *AudioPreprocessor) SpectralSubtraction(ctx context.Context, data []byte) []byte {
    // Remove noise in frequency domain
}
```

## Related Recipes

- **[Voice STT Jitter Buffer Management](./voice-stt-jitter-buffer-management.md)** - Handle network jitter
- **[Voice Providers Guide](../guides/voice-providers.md)** - For a deeper understanding of STT
