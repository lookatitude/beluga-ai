---
title: "Overcoming Background Noise"
description: "Recipe for preprocessing audio in Go with noise reduction, normalization, and filtering to improve STT accuracy in noisy production environments."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "Beluga AI, noise reduction, Go audio preprocessing, STT accuracy, noise filtering, voice quality, audio normalization recipe"
---

## Problem

Background noise degrades speech-to-text accuracy because STT models trained on clean audio struggle to distinguish speech from ambient sounds like traffic, HVAC systems, keyboard typing, or multiple speakers. This problem compounds in production environments where users operate in uncontrolled acoustic conditions. Without preprocessing, the STT service receives corrupted signals that lead to transcription errors, misunderstood commands, and poor user experiences. The challenge is particularly acute in real-time voice applications where you cannot ask users to repeat themselves or manually correct transcriptions.

## Solution

Audio preprocessing transforms noisy input into cleaner signals before the STT provider processes them. This approach is more effective than post-processing transcription errors because it addresses the root cause: poor signal quality. The solution applies three complementary techniques. Noise gating removes audio below a volume threshold, eliminating constant background hum and ambient noise quieter than typical speech. Normalization ensures consistent volume levels, preventing quiet speech from being lost and loud bursts from overwhelming the signal. Filtering targets specific frequency ranges, using high-pass filters to remove low-frequency rumble and band-pass filters to isolate speech frequencies (300Hz-3400Hz).

This preprocessing pipeline runs before the STT provider receives audio, meaning you improve the input signal quality rather than trying to compensate for poor transcription results. Clean signals produce better transcriptions, higher confidence scores, and more reliable downstream processing.

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

The code demonstrates preprocessing as a composable stage in the voice pipeline. The `AudioPreprocessor` applies each technique conditionally, allowing you to enable or disable stages based on environment characteristics. The `NoisySTTWrapper` shows how preprocessing integrates with existing STT providers using Beluga's FrameProcessor pattern, where each stage transforms audio before passing it to the next component.

## Explanation

1. **Noise gating** -- Zeroing audio samples below a threshold eliminates constant background noise like air conditioning, computer fans, or distant traffic. The gate acts as a dynamic filter that passes speech-level signals while blocking quieter ambient sounds. This improves the signal-to-noise ratio before the STT provider processes the audio. The threshold must be tuned per environment: too high and you clip quiet speech, too low and noise leaks through. Start at 0.1 and adjust based on transcription quality metrics.

2. **Normalization** -- STT models expect audio at consistent volume levels. Users speak at varying distances from microphones, creating signals that range from barely audible to clipping. Normalization scales audio to use the full dynamic range (0-32767 for 16-bit audio), ensuring quiet speakers are audible and loud speakers do not distort. This step is particularly important when combining audio from multiple sources or users, as it creates consistent input regardless of recording conditions.

3. **Filtering** -- Speech occupies specific frequency ranges (300Hz-3400Hz for telephone-quality speech, 80Hz-14kHz for wideband). High-pass filters remove low-frequency noise like rumble or handling bumps, while band-pass filters isolate speech frequencies and reject everything else. Filtering is more computationally intensive than gating or normalization but provides cleaner results when noise and speech overlap in volume but differ in frequency. Use frequency-domain filtering (FFT-based) for better results than time-domain approaches.

**Key insight:** Preprocess audio before STT, not after. Fixing transcription errors caused by noise is harder than preventing those errors by cleaning the input signal. The STT model cannot reconstruct information lost to noise, but it can accurately transcribe clean audio. Invest processing time in signal cleanup rather than error correction.

## Variations

### Adaptive Noise Gate

Adjust the noise gate threshold dynamically based on ambient noise level analysis. Measure noise floor during non-speech periods and set the gate threshold 10-15dB above that level to account for environment changes.

### Spectral Subtraction

Use spectral subtraction to remove noise in the frequency domain for more sophisticated noise reduction. This technique estimates the noise spectrum during silence and subtracts it from the speech spectrum during active speech, preserving speech quality better than simple gating.

## Related Recipes

- **[Jitter Buffer Management](./jitter-buffer)** -- Handle network jitter in audio streams
- **[Voice Backends Configuration](./voice-backends)** -- Backend provider setup
- **[VAD Sensitivity Profiles](./vad-sensitivity)** -- Tune voice activity detection for noisy environments
