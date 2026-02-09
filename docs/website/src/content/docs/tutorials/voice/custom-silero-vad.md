---
title: Custom VAD with Silero Models
description: Configure Silero VAD with custom ONNX models for precise voice activity detection in streaming and batch processing.
---

Silero VAD is a neural network-based voice activity detector that runs locally using ONNX Runtime. It provides more accurate speech detection than energy-based methods, especially in noisy environments. This tutorial demonstrates how to configure Silero VAD with custom model paths, tune thresholds, and integrate with streaming audio pipelines.

## What You Will Build

A voice activity detection system using Silero's ONNX model that classifies audio frames as speech or non-speech, with real-time streaming support and configurable sensitivity parameters.

## Prerequisites

- Go 1.23+
- Silero VAD ONNX model file (downloadable from the Silero VAD repository)
- Completion of [Sensitivity Tuning](/tutorials/voice/sensitivity-tuning) is recommended

## Step 1: Create a Silero VAD Provider

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/lookatitude/beluga-ai/voice/vad"
	_ "github.com/lookatitude/beluga-ai/voice/vad/providers/silero"
)

func main() {
	ctx := context.Background()

	modelPath := os.Getenv("SILERO_VAD_MODEL_PATH")
	if modelPath == "" {
		modelPath = filepath.Join(os.TempDir(), "silero_vad.onnx")
	}

	provider, err := vad.NewProvider(ctx, "silero", vad.DefaultConfig(),
		vad.WithModelPath(modelPath),
		vad.WithThreshold(0.5),
		vad.WithSampleRate(16000),
		vad.WithFrameSize(512),
	)
	if err != nil {
		log.Fatalf("create VAD provider: %v", err)
	}

	// Process a single audio frame
	audio := make([]byte, 1024)
	hasSpeech, err := provider.Process(ctx, audio)
	if err != nil {
		log.Fatalf("process: %v", err)
	}

	fmt.Printf("Speech detected: %v\n", hasSpeech)
}
```

## Configuration Options

| Option                | Default | Description                                         |
|----------------------|---------|------------------------------------------------------|
| `ModelPath`          | (built-in) | Path to the Silero VAD ONNX model file            |
| `Threshold`          | 0.5     | Speech confidence threshold (0.0-1.0)               |
| `SampleRate`         | 16000   | Audio sample rate in Hz                             |
| `FrameSize`          | 512     | Number of samples per audio frame                    |
| `MinSpeechDuration`  | 250ms   | Minimum consecutive speech frames to confirm speech  |
| `MaxSilenceDuration` | 500ms   | Maximum silence within speech before marking end     |
| `EnablePreprocessing`| false   | Apply preprocessing (normalization) before inference |

## Step 2: Real-Time Streaming Detection

For continuous audio processing, use `ProcessStream` to feed audio frames through a channel and receive VAD results in real time.

```go
	audioCh := make(chan []byte, 8)
	resultCh, err := provider.ProcessStream(ctx, audioCh)
	if err != nil {
		log.Fatalf("start process stream: %v", err)
	}

	// Consume results in a separate goroutine
	go func() {
		for result := range resultCh {
			if result.Error != nil {
				log.Printf("VAD error: %v", result.Error)
				continue
			}
			if result.HasVoice {
				fmt.Printf("Speech detected (confidence: %.2f)\n", result.Confidence)
			}
		}
	}()

	// Feed audio frames from your source
	for frame := range audioSource {
		audioCh <- frame
	}
	close(audioCh) // Signal end of audio
```

The `VADResult` struct provides:

| Field        | Type      | Description                              |
|-------------|-----------|------------------------------------------|
| `HasVoice`  | `bool`    | Whether speech was detected              |
| `Confidence`| `float64` | Model confidence score (0.0-1.0)         |
| `Error`     | `error`   | Non-nil if processing failed             |

## Step 3: Threshold Tuning by Environment

The threshold determines how confidently the model must classify a frame as speech before reporting it. Adjust based on your deployment environment.

```go
	// Quiet environment: lower threshold captures softer speech
	quietVAD, err := vad.NewProvider(ctx, "silero", vad.DefaultConfig(),
		vad.WithModelPath(modelPath),
		vad.WithThreshold(0.4),
		vad.WithMinSpeechDuration(200*time.Millisecond),
	)

	// Noisy environment: higher threshold reduces false positives
	noisyVAD, err := vad.NewProvider(ctx, "silero", vad.DefaultConfig(),
		vad.WithModelPath(modelPath),
		vad.WithThreshold(0.7),
		vad.WithMinSpeechDuration(350*time.Millisecond),
		vad.WithMaxSilenceDuration(600*time.Millisecond),
	)
```

### Tuning Guidelines

| Environment       | Threshold | MinSpeechDuration | MaxSilenceDuration |
|------------------|-----------|-------------------|--------------------|
| Quiet office     | 0.3-0.4   | 200ms             | 400ms              |
| Standard room    | 0.5       | 250ms             | 500ms              |
| Open office      | 0.6       | 300ms             | 500ms              |
| Call center      | 0.7-0.8   | 350ms             | 600ms              |

## Step 4: Combine with Noise Cancellation

For the best results in noisy environments, apply noise cancellation before VAD processing.

```go
import (
	"github.com/lookatitude/beluga-ai/voice/noise"
	_ "github.com/lookatitude/beluga-ai/voice/noise/providers/rnnoise"
)

func processWithNoiseCancellation(ctx context.Context, audio []byte) (bool, error) {
	// First, reduce noise
	noiseCanceller, err := noise.NewProvider(ctx, "rnnoise", noise.DefaultConfig())
	if err != nil {
		return false, fmt.Errorf("create noise canceller: %w", err)
	}

	cleanAudio, err := noiseCanceller.Process(ctx, audio)
	if err != nil {
		return false, fmt.Errorf("noise cancellation: %w", err)
	}

	// Then, detect voice activity on clean audio
	vadProvider, err := vad.NewProvider(ctx, "silero", vad.DefaultConfig(),
		vad.WithThreshold(0.5),
	)
	if err != nil {
		return false, fmt.Errorf("create VAD: %w", err)
	}

	return vadProvider.Process(ctx, cleanAudio)
}
```

## Verification

1. Set `SILERO_VAD_MODEL_PATH` to your ONNX model path (or omit for the built-in model).
2. Run the application and verify `Process` returns `false` for silence.
3. Feed speech audio and confirm `Process` returns `true`.
4. Test `ProcessStream` with continuous audio and verify real-time results.
5. Compare detection accuracy at different threshold levels.

## Next Steps

- [Sensitivity Tuning](/tutorials/voice/sensitivity-tuning) -- Tune VAD alongside turn detection
- [ML-Based Turn Prediction](/tutorials/voice/ml-turn-prediction) -- Neural turn-end detection
- [Voice Session Interruptions](/tutorials/voice/session-interruptions) -- Use VAD for barge-in detection
