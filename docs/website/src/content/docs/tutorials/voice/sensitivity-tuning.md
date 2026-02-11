---
title: VAD and Turn Detection Sensitivity Tuning
description: Optimize Voice Activity Detection and turn detection thresholds to balance responsiveness against false triggers in voice applications.
---

Voice Activity Detection (VAD) determines whether an audio frame contains speech, and turn detection determines when the user has finished speaking. These two systems work together to create the turn-taking rhythm of a conversation. Tuning their thresholds is critical: too sensitive and the agent reacts to background noise or interrupts the user mid-sentence; too conservative and users experience uncomfortable pauses while they wait for the agent to respond. This tutorial demonstrates how to configure both systems for natural conversational flow.

## What You Will Build

A tuned VAD and turn detection configuration that correctly distinguishes speech from background noise, triggers responses at natural conversation boundaries, and handles interruptions gracefully.

## Prerequisites

- Go 1.23+
- Basic understanding of audio frames and PCM encoding
- Completion of [Real-time STT Streaming](/tutorials/voice/stt-realtime-streaming) is recommended

## The Challenge: Natural Conversation Flow

If VAD is too sensitive, the agent interrupts when the user breathes, clears their throat, or a door closes. If VAD is too conservative, the agent misses the start of speech and the user must repeat themselves. Similarly, if turn detection triggers too quickly, the agent responds mid-sentence because it mistook a natural pause for the end of an utterance; if too slowly, it creates awkward silences that make users wonder whether the agent is still listening.

The goal is to find the threshold values that match your deployment environment and conversation style.

## Step 1: Configure VAD

VAD classifies each audio frame as speech or non-speech. The Silero VAD provider uses an ONNX neural network model for classification, which is more accurate than energy-based methods because it learns spectral features specific to human speech rather than relying on volume alone.

The key parameters interact with each other: `Threshold` controls per-frame sensitivity, `MinSpeechDuration` requires sustained detection before triggering, and `MaxSilenceDuration` determines how long silence can persist within speech before marking the end. Adjusting any one parameter affects the behavior of the others.

```go
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/lookatitude/beluga-ai/voice/vad"
	_ "github.com/lookatitude/beluga-ai/voice/vad/providers/silero"
)

func main() {
	ctx := context.Background()

	detector, err := vad.NewProvider(ctx, "silero", vad.DefaultConfig(),
		vad.WithThreshold(0.5),
		vad.WithSampleRate(16000),
		vad.WithFrameSize(512),
		vad.WithMinSpeechDuration(250*time.Millisecond),
		vad.WithMaxSilenceDuration(500*time.Millisecond),
	)
	if err != nil {
		log.Fatalf("create VAD provider: %v", err)
	}

	// Test with a single audio frame
	audio := make([]byte, 1024)
	hasSpeech, err := detector.Process(ctx, audio)
	if err != nil {
		log.Fatalf("process: %v", err)
	}

	fmt.Printf("Speech detected: %v\n", hasSpeech)
}
```

### Key Parameters

| Parameter             | Default | Description                                          |
|----------------------|---------|------------------------------------------------------|
| `Threshold`          | 0.5     | Confidence threshold (0.0-1.0). Higher = stricter.   |
| `MinSpeechDuration`  | 250ms   | Minimum speech duration to trigger detection.        |
| `MaxSilenceDuration` | 500ms   | Maximum silence within speech before marking as end. |
| `FrameSize`          | 512     | Audio frame size in samples.                         |
| `SampleRate`         | 16000   | Audio sample rate in Hz.                             |

## Step 2: Configure Turn Detection

Turn detection determines when the user has finished their utterance and the agent should respond. While VAD tells you whether someone is currently speaking, turn detection tells you whether they are done speaking. This distinction matters because natural speech includes pauses (thinking, breathing, mid-sentence hesitation) that VAD correctly detects as silence but that do not indicate the end of a turn.

The heuristic provider combines silence duration with transcript analysis to make this determination.

```go
import (
	"github.com/lookatitude/beluga-ai/voice/turndetection"
	_ "github.com/lookatitude/beluga-ai/voice/turndetection/providers/heuristic"
)

func setupTurnDetection(ctx context.Context) {
	detector, err := turndetection.NewProvider(ctx, "heuristic", turndetection.DefaultConfig(),
		turndetection.WithMinSilenceDuration(800*time.Millisecond),
		turndetection.WithSentenceEndMarkers(".!?"),
		turndetection.WithMinTurnLength(10),
		turndetection.WithMaxTurnLength(5000),
	)
	if err != nil {
		log.Fatalf("create turn detector: %v", err)
	}

	audio := make([]byte, 1024)
	done, err := detector.DetectTurn(ctx, audio)
	if err != nil {
		log.Fatalf("detect turn: %v", err)
	}

	fmt.Printf("Turn complete: %v\n", done)
}
```

For real-time pipelines, combine audio analysis with measured silence duration. The `DetectTurnWithSilence` method accepts an externally measured silence duration, which is typically provided by your VAD component. This allows the turn detector to make decisions based on actual measured silence rather than estimating it from the audio frames it receives.

```go
	silenceDuration := 500 * time.Millisecond
	done, err := detector.DetectTurnWithSilence(ctx, audio, silenceDuration)
	if err != nil {
		log.Fatalf("detect turn: %v", err)
	}

	if done {
		fmt.Println("User finished speaking; proceeding to response generation.")
	}
```

## Step 3: Interruption Handling

When the agent is speaking and VAD detects user speech, the agent should stop immediately. This creates the natural "barge-in" experience users expect. The interruption detection should be more sensitive than normal speech detection because the cost of missing an interruption (user feels ignored) is higher than the cost of a false positive (agent briefly pauses).

```go
// handleInterruption stops agent playback when the user starts speaking.
func handleInterruption(ctx context.Context, vadProvider vad.VADProvider, session voiceSession) {
	audioCh := make(chan []byte, 8)
	resultCh, err := vadProvider.ProcessStream(ctx, audioCh)
	if err != nil {
		log.Fatalf("start VAD stream: %v", err)
	}

	go func() {
		for result := range resultCh {
			if result.Error != nil {
				continue
			}
			if result.HasVoice && session.GetState() == "speaking" {
				// User started talking while agent is speaking: interrupt
				session.StopSpeaking()
			}
		}
	}()
}
```

## Step 4: Environment-Specific Tuning

Different environments require different sensitivity levels. A quiet office needs lower thresholds than a busy call center because the noise floor is different. Rather than tuning parameters individually each time, define environment profiles that encapsulate the complete tuning for a deployment scenario. This makes it straightforward to deploy the same application in different environments.

```go
// EnvironmentProfile holds tuned VAD parameters for a specific environment.
type EnvironmentProfile struct {
	Name              string
	Threshold         float64
	MinSpeechDuration time.Duration
	MaxSilenceDuration time.Duration
}

var profiles = map[string]EnvironmentProfile{
	"quiet": {
		Name:              "Quiet Office",
		Threshold:         0.4,
		MinSpeechDuration: 200 * time.Millisecond,
		MaxSilenceDuration: 400 * time.Millisecond,
	},
	"normal": {
		Name:              "Standard Environment",
		Threshold:         0.5,
		MinSpeechDuration: 250 * time.Millisecond,
		MaxSilenceDuration: 500 * time.Millisecond,
	},
	"noisy": {
		Name:              "Noisy Environment",
		Threshold:         0.7,
		MinSpeechDuration: 350 * time.Millisecond,
		MaxSilenceDuration: 600 * time.Millisecond,
	},
}
```

## Verification

1. Run the agent in your target environment.
2. Hum, cough, or tap the desk. The agent should not react.
3. Say "Hello." The agent should respond within approximately one second.
4. Interrupt the agent mid-sentence. It should stop immediately.
5. Speak a long sentence with a natural mid-sentence pause. The agent should wait for the full sentence before responding.

## Next Steps

- [Custom Silero VAD](/tutorials/voice/custom-silero-vad) -- Use custom ONNX models for VAD
- [Native S2S with Amazon Nova](/tutorials/voice/s2s-amazon-nova) -- Models with built-in VAD
- [LiveKit and Vapi Integration](/tutorials/voice/livekit-vapi) -- Server-side VAD in production backends
