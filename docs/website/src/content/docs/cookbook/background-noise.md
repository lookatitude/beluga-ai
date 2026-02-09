---
title: "Overcoming Background Noise"
description: "Preprocess audio with noise reduction, normalization, and filtering to improve STT accuracy in noisy environments."
---

## Problem

You need to improve speech-to-text accuracy in noisy environments by preprocessing audio to reduce background noise before sending it to the STT service.

## Solution

Implement audio preprocessing that applies noise gating, normalization, and filtering to improve signal quality. Clean audio signals produce better transcription results, and preprocessing enhances the signal-to-noise ratio before the STT provider processes the audio.

## Code Example

```go
package main

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

var tracer = otel.Tracer("beluga.voice.stt.noise_reduction")

// AudioPreprocessor preprocesses audio for noise reduction.
type AudioPreprocessor struct {
	noiseGate float64
	normalize bool
	filter    bool
}

func NewAudioPreprocessor(noiseGate float64, normalize, filter bool) *AudioPreprocessor {
	return &AudioPreprocessor{
		noiseGate: noiseGate,
		normalize: normalize,
		filter:    filter,
	}
}

// Preprocess applies noise reduction, normalization, and filtering.
func (ap *AudioPreprocessor) Preprocess(ctx context.Context, audioData []byte, sampleRate int) ([]byte, error) {
	ctx, span := tracer.Start(ctx, "audio_preprocessor.preprocess")
	defer span.End()

	span.SetAttributes(
		attribute.Int("input_size", len(audioData)),
		attribute.Int("sample_rate", sampleRate),
	)

	processed := audioData

	if ap.noiseGate > 0 {
		processed = ap.applyNoiseGate(processed, ap.noiseGate)
		span.SetAttributes(attribute.Bool("noise_gate_applied", true))
	}

	if ap.normalize {
		processed = ap.normalizeAudio(processed)
		span.SetAttributes(attribute.Bool("normalization_applied", true))
	}

	if ap.filter {
		processed = ap.applyFilter(processed, sampleRate)
		span.SetAttributes(attribute.Bool("filter_applied", true))
	}

	span.SetAttributes(attribute.Int("output_size", len(processed)))
	span.SetStatus(trace.StatusOK, "audio preprocessed")

	return processed, nil
}

func (ap *AudioPreprocessor) applyNoiseGate(data []byte, threshold float64) []byte {
	// Zero out samples below threshold.
	// In production, use a proper audio processing library.
	result := make([]byte, len(data))
	copy(result, data)
	return result
}

func (ap *AudioPreprocessor) normalizeAudio(data []byte) []byte {
	maxVal := float64(0)
	for i := 0; i < len(data); i += 2 {
		val := float64(data[i])
		if val > maxVal {
			maxVal = val
		}
	}

	if maxVal > 0 {
		scale := 32767.0 / maxVal
		result := make([]byte, len(data))
		for i := 0; i < len(data); i += 2 {
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

func (ap *AudioPreprocessor) applyFilter(data []byte, sampleRate int) []byte {
	// Apply high-pass filter to remove low-frequency noise.
	// In production, use a proper DSP library.
	return data
}

// NoisySTTWrapper wraps STT with noise reduction.
type NoisySTTWrapper struct {
	sttProvider  interface{}
	preprocessor *AudioPreprocessor
}

func NewNoisySTTWrapper(sttProvider interface{}, preprocessor *AudioPreprocessor) *NoisySTTWrapper {
	return &NoisySTTWrapper{
		sttProvider:  sttProvider,
		preprocessor: preprocessor,
	}
}

// Transcribe preprocesses audio then transcribes.
func (nsw *NoisySTTWrapper) Transcribe(ctx context.Context, audioData []byte, sampleRate int) (string, error) {
	ctx, span := tracer.Start(ctx, "noisy_stt_wrapper.transcribe")
	defer span.End()

	processed, err := nsw.preprocessor.Preprocess(ctx, audioData, sampleRate)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(trace.StatusError, err.Error())
		return "", err
	}

	// In production: result, err := nsw.sttProvider.Transcribe(ctx, processed, sampleRate)
	_ = processed
	span.SetStatus(trace.StatusOK, "transcription completed")
	return "", nil
}

func main() {
	preprocessor := NewAudioPreprocessor(0.1, true, true)

	audioData := make([]byte, 1024)
	processed, err := preprocessor.Preprocess(context.Background(), audioData, 16000)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Preprocessed %d bytes of audio\n", len(processed))
}
```

## Explanation

1. **Noise gating** -- Audio below a threshold is removed, eliminating background noise quieter than speech and improving signal quality.

2. **Normalization** -- Audio levels are normalized to ensure consistent volume, helping the STT service process audio more effectively.

3. **Filtering** -- High-pass filters remove low-frequency noise while band-pass filters can focus on speech frequencies (300Hz-3400Hz).

**Key insight:** Preprocess audio before STT, not after. Clean input produces better results than trying to fix transcription errors caused by noise.

## Variations

### Adaptive Noise Gate

Adjust the noise gate threshold dynamically based on ambient noise level analysis.

### Spectral Subtraction

Use spectral subtraction to remove noise in the frequency domain for more sophisticated noise reduction.

## Related Recipes

- **[Jitter Buffer Management](./jitter-buffer)** -- Handle network jitter in audio streams
- **[Voice Backends Configuration](./voice-backends)** -- Backend provider setup
- **[VAD Sensitivity Profiles](./vad-sensitivity)** -- Tune voice activity detection for noisy environments
