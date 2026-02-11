---
title: Configuring S2S Reasoning Modes
description: Tune Speech-to-Speech models for different interaction styles by configuring reasoning depth, latency targets, and dynamic mode switching.
---

Speech-to-Speech models can be configured to optimize for different interaction patterns. A fast concierge needs immediate responses, while a patient tutor benefits from deeper reasoning. The key insight is that reasoning depth and latency are directly correlated -- more internal reasoning produces more nuanced responses but delays the start of audio output. This tutorial demonstrates how to configure reasoning modes, optimize latency, and switch modes dynamically based on user intent.

## What You Will Build

An S2S voice system that supports multiple interaction modes (fast concierge, balanced assistant, deep tutor) with the ability to switch between them at runtime based on conversation context.

## Prerequisites

- Completion of [Native S2S with Amazon Nova](/tutorials/voice/s2s-amazon-nova) or an OpenAI Realtime API key
- Understanding of the S2S session lifecycle

## The Tradeoff: Speed vs. Thought

S2S models have internal reasoning steps that affect response quality and latency. More reasoning produces more nuanced and accurate responses, but increases the time before audio output begins. Understanding this tradeoff is essential for designing conversational experiences: a customer asking "What time do you close?" expects a near-instant response, while a student asking "Why does gravity bend light?" benefits from a considered explanation.

| Mode     | Latency    | Use Case                              |
|----------|------------|---------------------------------------|
| Low      | < 300ms    | Quick answers, confirmations, greetings |
| Medium   | 300-800ms  | General conversation, Q&A             |
| High     | 800ms+     | Complex reasoning, tutoring, analysis |

## Step 1: Configure Reasoning Mode

Use the `WithReasoningMode` and `WithLatencyTarget` options when creating the provider. These functional options follow Beluga's standard `WithX()` configuration pattern, allowing you to compose provider settings cleanly without a monolithic config struct.

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

The `ReasoningMode` controls how the model processes input. The built-in mode keeps reasoning inside the S2S model for the lowest latency. The external mode is useful when you need a more powerful text-based LLM (like Claude or GPT-4) to handle complex reasoning, with the S2S model handling only the audio conversion:

| Mode         | Description                                            |
|-------------|--------------------------------------------------------|
| `"built-in"` | Model handles reasoning internally (default)           |
| `"external"` | Application provides reasoning via an external LLM     |

## Step 2: System Prompts for Interaction Style

S2S models respond to system-level instructions that shape their behavior, just as text-based LLMs do. The prompt design matters more for S2S because it directly affects response length, which determines how long the user hears audio. Shorter, more directive prompts produce faster responses because the model generates fewer tokens internally before producing audio output.

```go
const (
	conciergePrompt = "You are a fast hotel concierge. Give short, 1-sentence answers. Be direct."
	tutorPrompt     = "You are a patient math tutor. Explain your reasoning step-by-step. " +
		"Use analogies. Confirm the student understands before moving on."
	assistantPrompt = "You are a helpful assistant. Provide clear, moderate-length answers."
)
```

Pass the prompt as part of the conversation context when starting a session. The `Preferences` map provides a flexible way to pass provider-specific settings without changing the core session interface:

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

To achieve glass-to-glass (microphone to speaker) latency under one second, you need to minimize delay at every stage of the pipeline. Each of these optimizations targets a specific source of latency:

1. **Reduce chunk size**: Send 20ms audio frames (640 bytes at 16kHz mono). Smaller frames mean the model receives input sooner, reducing the time before it can begin generating a response.
2. **Use server-side VAD**: Let the S2S model handle voice activity detection natively. Server-side VAD avoids the round-trip delay of client-side detection followed by a separate signal to the server.
3. **Set low latency target**: Configure the provider for speed over depth, accepting shorter, less reasoned responses in exchange for faster delivery.

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

Switch reasoning modes at runtime based on detected user intent or conversation context. This pattern is valuable for applications that serve multiple interaction styles within a single session -- for example, a virtual assistant that handles both quick factual queries and in-depth explanations. The mode map centralizes configuration for each style, making it straightforward to add new modes or adjust existing ones.

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
