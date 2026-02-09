---
title: Configuring S2S Reasoning Modes
description: Tune Speech-to-Speech models for different interaction styles by configuring reasoning depth, latency targets, and dynamic mode switching.
---

Speech-to-Speech models can be configured to optimize for different interaction patterns. A fast concierge needs immediate responses, while a patient tutor benefits from deeper reasoning. This tutorial demonstrates how to configure reasoning modes, optimize latency, and switch modes dynamically based on user intent.

## What You Will Build

An S2S voice system that supports multiple interaction modes (fast concierge, balanced assistant, deep tutor) with the ability to switch between them at runtime based on conversation context.

## Prerequisites

- Completion of [Native S2S with Amazon Nova](/tutorials/voice/s2s-amazon-nova) or an OpenAI Realtime API key
- Understanding of the S2S session lifecycle

## The Tradeoff: Speed vs. Thought

S2S models have internal reasoning steps that affect response quality and latency. More reasoning produces more nuanced and accurate responses, but increases the time before audio output begins.

| Mode     | Latency    | Use Case                              |
|----------|------------|---------------------------------------|
| Low      | < 300ms    | Quick answers, confirmations, greetings |
| Medium   | 300-800ms  | General conversation, Q&A             |
| High     | 800ms+     | Complex reasoning, tutoring, analysis |

## Step 1: Configure Reasoning Mode

Use the `WithReasoningMode` and `WithLatencyTarget` options when creating the provider.

```go
package main

import (
	"context"
	"log"
	"os"

	"github.com/lookatitude/beluga-ai/voice/s2s"
	_ "github.com/lookatitude/beluga-ai/voice/s2s/providers/openai_realtime"
)

func main() {
	ctx := context.Background()

	// Fast concierge mode: minimal reasoning, lowest latency
	provider, err := s2s.NewProvider(ctx, "openai_realtime", s2s.DefaultConfig(),
		s2s.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
		s2s.WithLatencyTarget("low"),
		s2s.WithReasoningMode("built-in"),
	)
	if err != nil {
		log.Fatalf("create S2S provider: %v", err)
	}

	_ = provider
}
```

The `ReasoningMode` controls how the model processes input:

| Mode         | Description                                            |
|-------------|--------------------------------------------------------|
| `"built-in"` | Model handles reasoning internally (default)           |
| `"external"` | Application provides reasoning via an external LLM     |

## Step 2: System Prompts for Interaction Style

S2S models respond to system-level instructions that shape their behavior, just as LLMs do. Shorter, more directive prompts produce faster responses.

```go
const (
	conciergePrompt = "You are a fast hotel concierge. Give short, 1-sentence answers. Be direct."
	tutorPrompt     = "You are a patient math tutor. Explain your reasoning step-by-step. " +
		"Use analogies. Confirm the student understands before moving on."
	assistantPrompt = "You are a helpful assistant. Provide clear, moderate-length answers."
)
```

Pass the prompt as part of the conversation context when starting a session:

```go
	conversationCtx := &s2s.ConversationContext{
		SessionID: "session-001",
		UserID:    "user-001",
		Preferences: map[string]any{
			"system_prompt": conciergePrompt,
		},
	}

	session, err := streamingProvider.StartStreaming(ctx, conversationCtx,
		s2s.WithLatencyTarget("low"),
	)
	if err != nil {
		log.Fatalf("start session: %v", err)
	}
	defer session.Close()
```

## Step 3: Latency Optimization

To achieve glass-to-glass (microphone to speaker) latency under one second:

1. **Reduce chunk size**: Send 20ms audio frames (640 bytes at 16kHz mono).
2. **Use server-side VAD**: Let the S2S model handle voice activity detection natively.
3. **Set low latency target**: Configure the provider for speed over depth.

```go
	provider, err := s2s.NewProvider(ctx, "openai_realtime", &s2s.Config{
		Provider:      "openai_realtime",
		APIKey:        os.Getenv("OPENAI_API_KEY"),
		LatencyTarget: "low",
		SampleRate:    24000,
		ProviderSpecific: map[string]any{
			"turn_detection_type": "server_vad",
			"silence_duration":    500, // ms before triggering response
		},
	})
	if err != nil {
		log.Fatalf("create provider: %v", err)
	}
```

## Step 4: Dynamic Mode Switching

Switch reasoning modes at runtime based on detected user intent or conversation context.

```go
// ModeConfig holds the configuration for an interaction mode.
type ModeConfig struct {
	Prompt        string
	LatencyTarget string
}

var modes = map[string]ModeConfig{
	"concierge": {
		Prompt:        conciergePrompt,
		LatencyTarget: "low",
	},
	"assistant": {
		Prompt:        assistantPrompt,
		LatencyTarget: "medium",
	},
	"tutor": {
		Prompt:        tutorPrompt,
		LatencyTarget: "high",
	},
}

// SwitchMode reconfigures the S2S session for a different interaction style.
func SwitchMode(ctx context.Context, provider s2s.S2SProvider, modeName string) error {
	mode, ok := modes[modeName]
	if !ok {
		return fmt.Errorf("unknown mode: %s", modeName)
	}

	streamingProvider, ok := provider.(s2s.StreamingS2SProvider)
	if !ok {
		return fmt.Errorf("provider does not support streaming")
	}

	conversationCtx := &s2s.ConversationContext{
		SessionID: "session-001",
		UserID:    "user-001",
		Preferences: map[string]any{
			"system_prompt": mode.Prompt,
		},
	}

	session, err := streamingProvider.StartStreaming(ctx, conversationCtx,
		s2s.WithLatencyTarget(mode.LatencyTarget),
	)
	if err != nil {
		return fmt.Errorf("start session for mode %s: %w", modeName, err)
	}

	// Use session for subsequent interaction
	_ = session
	return nil
}
```

## Verification

1. Set `LatencyTarget` to `"low"`. Ask "What is 2+2?" and measure time to first audio.
2. Set `LatencyTarget` to `"high"`. Ask "Explain the theory of relativity" and compare response depth and latency.
3. Switch modes dynamically and verify the behavior changes.

## Next Steps

- [Native S2S with Amazon Nova](/tutorials/voice/s2s-amazon-nova) -- Try a different S2S provider
- [Real-time STT Streaming](/tutorials/voice/stt-realtime-streaming) -- Compare with the traditional pipeline
- [Sensitivity Tuning](/tutorials/voice/sensitivity-tuning) -- Tune VAD and turn detection thresholds
